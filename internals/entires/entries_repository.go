package entries

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerEntry struct {
	TransactionID int64
	AccountID     int64
	Currency      string
	EntryType     string
	AmountMinor   int64
}

type EntriesRepository struct {
	db *pgxpool.Pool
}

func NewEntriesRepository(db *pgxpool.Pool) *EntriesRepository {
	return &EntriesRepository{db: db}
}

func (r *EntriesRepository) CreateLedgerEntry(ctx context.Context, l LedgerEntry, tx pgx.Tx) (int64, error) {
	if l.TransactionID <= 0 {
		return 0, errors.New("invalid transaction id")
	}

	if l.AccountID <= 0 {
		return 0, errors.New("invalid account id")
	}

	if len(l.Currency) != 3 || l.Currency != strings.ToUpper(l.Currency){
			return 0, errors.New("invalid currency")
	}

	if l.EntryType != "debit" && l.EntryType != "credit" {
		return -1, errors.New("invalid entry type")
	}

	if l.AmountMinor <= 0 {
		return -1, errors.New("invalid amount")
	}

	var entryID int64

	err := tx.QueryRow(ctx,
		`INSERT INTO ledger_entries (transaction_id, account_id, currency, entry_type, amount_minor) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		l.TransactionID,
		l.AccountID,
		l.Currency,
		l.EntryType,
		l.AmountMinor,
	).Scan(&entryID)
	if err != nil {
		return -1, fmt.Errorf("failed to insert entry in db: %w", err)
	}
	
	return entryID, nil
}
