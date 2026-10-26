CREATE TABLE ledger_accounts (
    id BIGSERIAL PRIMARY KEY,
    owner_type VARCHAR(20) NOT NULL,
    owner_id BIGINT NOT NULL.
    currency CHAR(3) NOT NULL DEFAULT 'INR',
    balance_minor BIGINT NOT NULL DEFAULT 0 CHECK (balance_minor >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (id, currency)
);

CREATE TABLE ledger_transactions (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key VARCHAR(100) NOT NULL UNIQUE,
    transaction_type VARCHAR(30) NOT NULL,
    source_account_id BIGINT NOT NULL REFERENCES ledger_accounts(id),
    destination_account_id BIGINT NOT NULL REFERENCES ledger_accounts(id),
    amount_minor BIGINT NOT NULL CHECK(amount_minor > 0),
    currency CHAR(3) NOT NULL DEFAULT 'INR'
)