package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if t.IdempotencyKey == "" || t.SourceAccountID <=0 || t.DestinationAccountID <=0 || t.SourceAccountID == t.DestinationAccountID || t.AmountMinor <=0 {
		return 0, errors.New("invalid transaction")
	}

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
	// row with that key already exists in the DB, so we will return the existing row
	if errors.Is(err, pgx.ErrNoRows){
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

	


	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit transfer: %w", err)
	}

	return transactionID, nil
}