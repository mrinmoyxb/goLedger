package entries

import (
	"context"
	"errors"
	"fmt"
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

func (r *EntriesRepository) GetEntriesByTransactionID(ctx context.Context, id int64, limit int, offset int) ([]LedgerEntryInfo, error) {
	if id <= 0 {
		return nil, errors.New("invalid transaction id")
	}

	if limit <= 0 || limit > 100 {
		return nil, errors.New("invalid limit")
	}

	if offset < 0 {
		return nil, errors.New("invalid offset")
	}

	rows, err := r.db.Query(ctx, 
		`SELECT transaction_id, account_id, currency, entry_type, amount_minor, created_at 
			FROM ledger_entries 
			WHERE transaction_id = $1 
			ORDER BY id ASC
			LIMIT $2 OFFSET $3`,
		id,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("get ledger entries: %w", err)
	}

	defer rows.Close()

	entries := make([]LedgerEntryInfo, 0)

	for rows.Next() {
		var entry LedgerEntryInfo
		err := rows.Scan(
			&entry.TransactionID,
			&entry.AccountID,
			&entry.Currency,
			&entry.EntryType,
			&entry.AmountMinor,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ledger entry: %w", err)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ledger entries: %w", err)
	}

	return entries, nil
}

func (r *EntriesRepository) GetEntriesByAccountID(ctx context.Context, id int64) ([]LedgerEntryInfo, error) {
	if id <=0 {
		return nil, errors.New("invalid account id")
	}

	rows, err := r.db.Query(ctx,
	`SELECT transaction_id, account_id, currency, entry_type, amount_minor, created_at
		FROM ledger_entries
		WHERE account_id = $1 
		ORDER BY id ASC`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get ledger entries: %w", err)
	}

	defer rows.Close()

	entries := make([]LedgerEntryInfo, 0)

	for rows.Next() {
		var entry LedgerEntryInfo
		err := rows.Scan(
			&entry.TransactionID,
			&entry.AccountID,
			&entry.Currency,
			&entry.EntryType,
			&entry.AmountMinor,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ledger entries: %w", err)
	}

	return entries, nil
}

func (r *EntriesRepository) GetEntriesByAccountIDpaginated(ctx context.Context, id int64, limit int, offset int) ([]LedgerEntryInfo, error) {
	if id <=0 {
		return nil, errors.New("invalid account id")
	}

	if limit <= 0 || limit > 100 {
		return nil, errors.New("invalid limit")
	}

	if offset < 0 {
		return nil, errors.New("invalid offset")
	}

	rows, err := r.db.Query(ctx,
	`SELECT transaction_id, account_id, currency, entry_type, amount_minor, created_at
		FROM ledger_entries
		WHERE account_id = $1 
		ORDER BY id ASC
		LIMIT $2 OFFSET $3`,
		id,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("get ledger entries: %w", err)
	}

	defer rows.Close()

	entries := make([]LedgerEntryInfo, 0)

	for rows.Next() {
		var entry LedgerEntryInfo
		err := rows.Scan(
			&entry.TransactionID,
			&entry.AccountID,
			&entry.Currency,
			&entry.EntryType,
			&entry.AmountMinor,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ledger entries: %w", err)
	}

	return entries, nil
}

func (r *EntriesRepository) SumEntriesByAccountID(ctx context.Context, id int64) (int64, error) {
	if id <=0 {
		return 0, errors.New("invalid account ID")
	}

	var sum int64

	err := r.db.QueryRow(ctx,
	`SELECT COALESCE(SUM(
		CASE 
			WHEN entry_type = 'credit' THEN amount_minor
			WHEN entry_type = 'debit' THEN -amount_minor
		END
		), 0)
		FROM ledger_entries
		WHERE account_id = $1
	`, id,
	).Scan(&sum)

	if err != nil {
		return 0, fmt.Errorf("sum of ledger entries: %w", err)
	}

	return sum, nil
}

func (r *EntriesRepository) SumDebitsAndCreditsByTransactionID(ctx context.Context) {

}

func (r *EntriesRepository) GetEntriesByReference(ctx context.Context) {

}