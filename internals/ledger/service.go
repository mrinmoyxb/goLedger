package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidTransfer = errors.New("invalid transfer")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountNotFound = errors.New("account not found")
	ErrCurrencyMismatch = errors.New("currency mismatch")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different request")
)

// PostTransfer transfers money between two ledger accounts.
// amountMinor is the amount in the currency's smallest unit.
// For INR, 50000 means ₹500.
func PostTransfer(
	ctx context.Context,
	db *pgxpool.Pool,
	idempotencyKey string,
	sourceAccountID int64,
	destinationAccountID int64,
	amountMinor int64,
) (int64, error) {
	if idempotencyKey == "" ||
		sourceAccountID <= 0 ||
		destinationAccountID <= 0 ||
		sourceAccountID == destinationAccountID ||
		amountMinor <= 0 {
		return 0, ErrInvalidTransfer
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	currency := "INR"

	// 1. Create the transaction record.
	// If this idempotency key already exists, no new record is created.
	var transactionID int64

	err = tx.QueryRow(ctx, `
		INSERT INTO ledger_transactions (
			idempotency_key,
			transaction_type,
			source_account_id,
			destination_account_id,
			amount_minor,
			currency
		)
		VALUES ($1, 'wallet_transfer', $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id
	`,
		idempotencyKey,
		sourceAccountID,
		destinationAccountID,
		amountMinor,
		currency,
	).Scan(&transactionID)

	if errors.Is(err, pgx.ErrNoRows) {
		// A request with this key has already been processed.
		var existingSourceID int64
		var existingDestinationID int64
		var existingAmount int64
		var existingCurrency string

		err = tx.QueryRow(ctx, `
			SELECT id, source_account_id,
			       destination_account_id, amount_minor, currency
			FROM ledger_transactions
			WHERE idempotency_key = $1
		`, idempotencyKey).Scan(
			&transactionID,
			&existingSourceID,
			&existingDestinationID,
			&existingAmount,
			&existingCurrency,
		)

		if err != nil {
			return 0, fmt.Errorf("find existing transaction: %w", err)
		}

		// The same key must represent the same transfer.
		if existingSourceID != sourceAccountID ||
			existingDestinationID != destinationAccountID ||
			existingAmount != amountMinor ||
			existingCurrency != currency {
			return 0, ErrIdempotencyConflict
		}

		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit idempotent retry: %w", err)
		}

		// Return the original transaction ID without transferring again.
		return transactionID, nil
	}

	if err != nil {
		return 0, fmt.Errorf("insert ledger transaction: %w", err)
	}

	// 2. Lock both accounts in a consistent order.
	// This prevents concurrent transfers from overspending.
	rows, err := tx.Query(ctx, `
		SELECT id, currency, balance_minor
		FROM ledger_accounts
		WHERE id = ANY($1::bigint[])
		ORDER BY id
		FOR UPDATE
	`, []int64{sourceAccountID, destinationAccountID})
	if err != nil {
		return 0, fmt.Errorf("lock accounts: %w", err)
	}

	type account struct {
		currency string
		balance  int64
	}

	accounts := make(map[int64]account, 2)

	for rows.Next() {
		var id int64
		var acc account

		if err := rows.Scan(&id, &acc.currency, &acc.balance); err != nil {
			rows.Close()
			return 0, fmt.Errorf("read account: %w", err)
		}

		accounts[id] = acc
	}

	err = rows.Err()
	rows.Close()

	if err != nil {
		return 0, fmt.Errorf("iterate accounts: %w", err)
	}

	source, sourceExists := accounts[sourceAccountID]
	destination, destinationExists := accounts[destinationAccountID]

	if !sourceExists || !destinationExists {
		return 0, ErrAccountNotFound
	}

	// 3. Validate currency and available funds.
	if source.currency != currency ||
		destination.currency != currency {
		return 0, ErrCurrencyMismatch
	}

	if source.balance < amountMinor {
		return 0, ErrInsufficientFunds
	}

	// 4. Insert the debit entry.
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (
			transaction_id, account_id, currency,
			entry_type, amount_minor
		)
		VALUES ($1, $2, $3, 'debit', $4)
	`,
		transactionID,
		sourceAccountID,
		currency,
		amountMinor,
	)
	if err != nil {
		return 0, fmt.Errorf("insert debit entry: %w", err)
	}

	// 5. Insert the credit entry.
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (
			transaction_id, account_id, currency,
			entry_type, amount_minor
		)
		VALUES ($1, $2, $3, 'credit', $4)
	`,
		transactionID,
		destinationAccountID,
		currency,
		amountMinor,
	)
	if err != nil {
		return 0, fmt.Errorf("insert credit entry: %w", err)
	}

	// 6. Update the source wallet balance.
	_, err = tx.Exec(ctx, `
		UPDATE ledger_accounts
		SET balance_minor = balance_minor - $1
		WHERE id = $2
	`,
		amountMinor,
		sourceAccountID,
	)
	if err != nil {
		return 0, fmt.Errorf("debit source account: %w", err)
	}

	// 7. Update the destination wallet balance.
	_, err = tx.Exec(ctx, `
		UPDATE ledger_accounts
		SET balance_minor = balance_minor + $1
		WHERE id = $2
	`,
		amountMinor,
		destinationAccountID,
	)
	if err != nil {
		return 0, fmt.Errorf("credit destination account: %w", err)
	}

	// 8. Commit every change together.
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit transfer: %w", err)
	}

	return transactionID, nil
}