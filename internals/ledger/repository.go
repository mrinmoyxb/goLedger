package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Account struct {
	Currency string
	Balance  int64
}

type Transaction struct {
	IdempotencyKey       string
	SourceAccountID      int64
	DestinationAccountID int64
	AmountMinor          int64
}

type LedgerRepository struct {
	db *pgxpool.Pool
}

func NewLedgerRepository(db *pgxpool.Pool) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) PostTransfer(ctx context.Context, t Transaction) (int64, error) {

	//* Step 1: Validation of input:-> idempotency_key, source_account, destination_account, amount_minor,
	if t.IdempotencyKey == "" || t.SourceAccountID <= 0 || t.DestinationAccountID <= 0 || t.SourceAccountID == t.DestinationAccountID || t.AmountMinor <= 0 {
		return 0, errors.New("invalid transaction")
	}

	//* Step 2: Begin transaction:-> insert a record using idempotency_key: prevents duplicate transfers
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	currency := "INR"
	var transactionID int64

	err = tx.QueryRow(ctx,
		`INSERT INTO ledger_transactions 
			(idempotency_key, transaction_type, source_account_id, destination_account_id, amount_minor, currency)
			VALUES ($1, 'wallet_transfer', $2, $3, $4, $5)
			ON CONFLICT (idempotency_key) DO NOTHING
			RETURNING id`,
		t.IdempotencyKey,
		t.SourceAccountID,
		t.DestinationAccountID,
		t.AmountMinor,
		currency,
	).Scan(&transactionID)

	//* Step 3: Since it doesn't return any row, then let's check for existing row with the help of the idempotency_key and return the existing info
	if errors.Is(err, pgx.ErrNoRows) {
		var existingSourceID int64
		var existingDestinationID int64
		var existingAmount int64
		var existingCurrency string

		err = tx.QueryRow(ctx,
			`SELECT id, source_account_id, destination_account_id, amount_minor, currency 
			FROM ledger_transactions 
			WHERE idempotency_key = $1`,
			t.IdempotencyKey,
		).Scan(&transactionID, &existingSourceID, &existingDestinationID, &existingAmount, &existingCurrency)

		if err != nil {
			return 0, fmt.Errorf("find existing transaction: %w", err)
		}

		if existingSourceID != t.SourceAccountID || existingDestinationID != t.DestinationAccountID || existingAmount != t.AmountMinor || existingCurrency != currency {
			return 0, fmt.Errorf("idempotency conflict")
		}

		if err = tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit idempotent retry: %w", err)
		}

		return transactionID, nil
	}

	if err != nil {
		return 0, fmt.Errorf("insert ledger transaction: %w", err)
	}

	//* Step 4: applied row level lock on both accounts
	rows, err := tx.Query(ctx,
		`SELECT id, currency, balance_minor FROM ledger_accounts WHERE id = ANY($1::bigint[]) ORDER BY id FOR UPDATE`,
		[]int64{t.SourceAccountID, t.DestinationAccountID})
	if err != nil {
		return 0, fmt.Errorf("lock accounts: %w", err)
	}

	accounts := make(map[int64]Account, 2)

	for rows.Next() {
		var id int64
		var acc Account

		if err := rows.Scan(&id, &acc.Currency, &acc.Balance); err != nil {
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

	source, sourceExists := accounts[t.SourceAccountID]
	destination, destinationExists := accounts[t.DestinationAccountID]
	if !sourceExists || !destinationExists {
		return 0, errors.New("account not found")
	}

	//* Step 5: Validation
	if source.Currency != currency || destination.Currency != currency {
		return 0, errors.New("currency mismatch")
	}

	if source.Balance < t.AmountMinor {
		return 0, errors.New("insufficient balance")
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit transfer: %w", err)
	}

	//* Step 6: Insert the debit and credit entry
	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (transaction_id, account_id, currency, entry_type, amount_minor) 
	VALUES ($1, $2, $3, 'debit', $4)
	`, transactionID, t.SourceAccountID, currency, t.AmountMinor)
	if err != nil {
		return 0, fmt.Errorf("insert debit entry: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (transaction_id, account_id, currency, entry_type, amount_minor) 
	VALUES ($1, $2, $3, 'debit', $4)
	`, transactionID, t.DestinationAccountID, currency, t.AmountMinor)
	if err != nil {
		return 0, fmt.Errorf("insert credit entry: %w", err)
	}

	//* Step 7: Update source and destination balance
	_, err = tx.Exec(ctx,
		`UPDATE ledger_accounts SET balance_minor = balance_minor - $1 WHERE id = $2`,
		t.AmountMinor, t.SourceAccountID,
	)
	if err != nil {
		return 0, fmt.Errorf("debit source account: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE ledger_accounts
		SET balance_minor = balance_minor + $1
		WHERE id = $2
	`,
		t.AmountMinor,
		t.DestinationAccountID,
	)
	if err != nil {
		return 0, fmt.Errorf("credit destination account: %w", err)
	}

	//* Step 8: transaction ends
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit idempotent retry: %w", err)
	}

	return transactionID, nil
}
