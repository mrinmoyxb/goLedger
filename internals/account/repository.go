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
		if errors.Is(err, pgx.ErrNoRows){
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("get ledger account by id: %w", err)
	}

	return account, nil
}
