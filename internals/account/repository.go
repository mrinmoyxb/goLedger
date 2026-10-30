package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Account struct {
	OwnerType     string
	OwnerId       int64
	AccountType   string
	Purpose       string
	Currency      string
	AccountStatus string
}

type AccountInfo struct {
	ID            int64
	OwnerType     string
	OwnerID       int64
	AccountType   string
	Purpose       string
	Currency      string
	BalanceMinor  int64
	AccountStatus string
	CreatedAt     time.Time
}

type Pagination struct {
	Limit int
	Offset int 
}

type AccountRepository struct {
	db *pgxpool.Pool
}

func NewAccountRepository(db *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) CreateAccount(ctx context.Context, acc Account) (int64, error) {
	if acc.OwnerType == "" || acc.OwnerId <= 0 || acc.AccountType == "" || acc.Purpose == "" || acc.Currency == "" || acc.AccountStatus == "" {
		return 0, errors.New("all fields are mandatory")
	}

	var accountID int64

	err := r.db.QueryRow(ctx, `INSERT INTO ledger_accounts (owner_type, owner_id, account_type, purpose, currency, 
			account_status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		acc.OwnerType, acc.OwnerId, acc.AccountType, acc.Purpose, acc.Currency, acc.AccountStatus,
	).Scan(&accountID)

	if err != nil {
		return 0, fmt.Errorf("create ledger account: %w", err)
	}

	return accountID, nil
}

func (r *AccountRepository) GetAccountById(ctx context.Context, id int64) (*AccountInfo, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}

	account := &AccountInfo{}

	err := r.db.QueryRow(ctx,
		`SELECT id, owner_type, owner_id, account_type, purpose, currency, balance_minor, account_status, created_at
	FROM ledger_accounts WHERE id = $1`, id,
	).Scan(&account.ID, &account.OwnerType, &account.OwnerID, &account.AccountType, &account.Purpose, &account.Currency,
		&account.BalanceMinor, &account.AccountStatus, &account.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("get ledger account by id: %w", err)
	}

	return account, nil
}

func (r *AccountRepository) GetAccountByIdForUpdate(ctx context.Context, tx pgx.Tx, id int64) (*AccountInfo, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}

	account := &AccountInfo{}

	err := tx.QueryRow(ctx,
		`SELECT id, owner_type, owner_id, account_type, purpose, currency, balance_minor, account_status, created_at
	FROM ledger_accounts WHERE id = $1 FOR UPDATE`, id,
	).Scan(&account.ID, &account.OwnerType, &account.OwnerID, &account.AccountType, &account.Purpose, &account.Currency,
		&account.BalanceMinor, &account.AccountStatus, &account.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account %d not found %w", id, err)
		}

		return nil, fmt.Errorf("get account for update: %w", err)
	}

	return account, nil
}

func (r *AccountRepository) GetAccountByIdsForUpdate(ctx context.Context, tx pgx.Tx, ids []int64) (map[int64]AccountInfo, error) {
	if len(ids) == 0 {
		return nil, errors.New("account IDs can't be empty")
	}

	accounts := make(map[int64]AccountInfo, len(ids))

	rows, err := tx.Query(ctx, 
	`SELECT id, owner_type, owner_id, account_type, purpose, currency, balance_minor, account_status, created_at FROM ledger_accounts 
		WHERE id = ANY($1::bigint[]) ORDER BY id FOR UPDATE`, 
	ids)

	if err != nil {
        return nil, fmt.Errorf("get accounts: %w", err)
    }

	defer rows.Close()

	for rows.Next() {
		var account AccountInfo
		err := rows.Scan(
			&account.ID,
			&account.OwnerType,
			&account.OwnerID,
			&account.AccountType,
			&account.Purpose,
			&account.Currency,
			&account.BalanceMinor,
			&account.AccountStatus,
			&account.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("Scan account: %w", err)
		}
		accounts[account.ID] = account
	}

	 if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate accounts: %w", err)
    }

    if len(accounts) != len(ids) {
        return nil, errors.New("one or more accounts not found")
    }

    return accounts, nil

}

func (r *AccountRepository) GetAccountBalance(ctx context.Context, id int64) (int64, error) {
	if id <= 0 {
		return -1, errors.New("invalid id")
	}

	balance := 0

	err := r.db.QueryRow(ctx,
		`SELECT balance_minor FROM ledger_accounts WHERE id = $1`, id).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("get account balance: %w", err)
	}

	return int64(balance), nil
}

func (r *AccountRepository) UpdateAccountBalance(ctx context.Context, id int64, balance int64) error {
	if id <= 0 {
		return errors.New("invalid account id")
	}

	if balance < 0 {
		return errors.New("balance can't be negative")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `UPDATE ledger_accounts SET balance_minor = $1 WHERE id = $2 FOR UPDATE`, balance, id)

	if err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return errors.New("account not found")
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit %w", err)
	}

	return nil
}

func (r *AccountRepository) UpdateAccountStatus(ctx context.Context, id int64, status string) error {
	if status == "" && status != "active" && status != "frozen" && status != "closed" {
		return errors.New("invalid account status")
	}

	if id <= 0 {
		return errors.New("invalid account id")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `UPDATE ledger_accounts SET account_status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return errors.New("account not found")
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit %w", err)
	}

	return nil
}

func (r *AccountRepository) ListAccounts(ctx context.Context, pagination Pagination)([]AccountInfo, error){
	if pagination.Limit <= 0 {
		return nil, errors.New("invalid pagination limit")
	}

	if pagination.Offset < 0 {
		return nil, errors.New("invalid pagination offset")
	}

	rows, err := r.db.Query(ctx,
	`SELECT id, owner_type, owner_id, account_type, purpose, currency, balance_minor, account_status, created_at
		FROM ledger_accounts ORDER BY id LIMIT $1 OFFSET $2`,
		pagination.Limit, pagination.Offset,
	)

	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	defer rows.Close()

	accounts := make([]AccountInfo, 0)

	for rows.Next() {
		var account AccountInfo

		err := rows.Scan(
			&account.ID,
			&account.OwnerType,
			&account.OwnerID,
			&account.Purpose,
			&account.Currency,
			&account.BalanceMinor,
			&account.AccountStatus,
			&account.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}

		accounts = append(accounts, account)
	}

	 if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate accounts: %w", err)
    }

    return accounts, nil
}

func (r *AccountRepository) AccountExists(ctx context.Context, id int64) (bool, error) {
	if id <= 0 {
		return false, errors.New("invalid account id")
	}

	var accountID int64

	err := r.db.QueryRow(ctx,
	`SELECT id FROM ledger_accounts WHERE id = $1`, 
	id).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, errors.New("account not found")
	}

	return true, nil
}