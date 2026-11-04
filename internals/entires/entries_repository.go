package entries

import (
	"context"
	"errors"
	"fmt"
	"goLedger/internals/transactions"
	"strings"
	"time"

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

type LedgerEntryInfo struct {
	TransactionID int64
	AccountID     int64
	Currency      string
	EntryType     string
	AmountMinor   int64
	CreatedAt     time.Time
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
		return 0, errors.New("invalid entry type")
	}

	if l.AmountMinor <= 0 {
		return 0, errors.New("invalid amount")
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

func (r *EntriesRepository) CreateLedgerEntries(ctx context.Context, l []LedgerEntry, tx pgx.Tx) (int64, error) {
	if len(l) == 0 {
		return 0, nil
	}

	query := `INSERT INTO ledger_entries (transaction_id, account_id, currency, entry_type, amount_minor) VALUES`
	args := make([]any, 0, len(l)*5)

	for i, entry := range l {
		if i > 0 {
			query += ", "
		}
		offset := i * 5
		query += fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d)",
			offset+1,
			offset+2,
			offset+3,
			offset+4,
			offset+5,
		)

		args = append(args, entry.TransactionID, entry.AccountID, entry.Currency, entry.EntryType, entry.AmountMinor)
	}

	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return tag.RowsAffected(), nil
}

func (r *EntriesRepository) GetEntriesByTransactionID(ctx context.Context, id int64) ([]LedgerEntryInfo, error) {
	if id <= 0 {
		return nil, errors.New("invalid transaction id")
	}

	rows, err := r.db.Query(ctx, 
		`SELECT transaction_id, account_id, currency, entry_type, amount_minor, created_at WHERE transaction_id = $1`,
		id,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return nil, errors.New("no rows available")
		}
		return nil, err
	}

	defer rows.Close()

	entries := make([]LedgerEntryInfo, 0)
	for rows.Next() {
		var entry LedgerEntryInfo
		err := rows.Scan(
			&entry.TransactionID,
			&entry.AccountID,
			&entry.Currency,
			&entry.AmountMinor,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan entries: %w", err)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	return entries, nil

}
