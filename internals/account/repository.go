package account

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	OwnerId       int64
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

func NewAccountRepository(db *pgxpool.Pool)	*AccountRepository {
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

func (r *AccountRepository) GetAccountById(ctx context.Context, acc Account) {

}