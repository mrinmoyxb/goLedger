CREATE TABLE ledger_accounts ( 
    id BIGSERIAL PRIMARY KEY, 
    owner_type VARCHAR(20) NOT NULL CHECK(owner_type IN ('user', 'merchant', 'platform', 'system')), 
    owner_id BIGINT NOT NULL, 
    account_type VARCHAR(20) NOT NULL CHECK(account_type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
    purpose VARCHAR(30) NOT NULL DEFAULT 'main',
    currency CHAR(3) NOT NULL DEFAULT 'INR' CHECK(currency ~ '^[A-Z]{3}$'), 
    balance_minor BIGINT NOT NULL DEFAULT 0, 
    account_status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK(account_status IN ('active', 'frozen', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    UNIQUE (id, currency),
    UNIQUE (owner_type, owner_id, account_type, purpose, currency)
); 
 
CREATE TABLE ledger_transactions ( 
    id BIGSERIAL PRIMARY KEY, 
    idempotency_key VARCHAR(100) NOT NULL UNIQUE, 
    request_hash CHAR(64) NOT NULL CHECK(request_hash ~ '^[0-9a-f]{64}$'),
    transaction_type VARCHAR(30) NOT NULL CHECK(transaction_type IN('transfer', 'opening_balance', 'fee', 'refund', 'reversal', 'settlement')),  
    amount_minor BIGINT NOT NULL CHECK(amount_minor > 0), 
    currency CHAR(3) NOT NULL DEFAULT 'INR' CHECK (currency ~ '^[A-Z]{3}$'), 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    UNIQUE(id, currency)
); 

CREATE TABLE ledger_entries ( 
    id BIGSERIAL PRIMARY KEY, 
    transaction_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL, 
    currency CHAR(3) NOT NULL CHECK(currency ~ '^[A-Z]{3}$'), 
    entry_type VARCHAR(10) NOT NULL CHECK(entry_type IN ('debit', 'credit')), 
    amount_minor BIGINT NOT NULL CHECK(amount_minor > 0), 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    FOREIGN KEY(transaction_id, currency) REFERENCES ledger_transactions(id, currency), 
    FOREIGN KEY(account_id, currency) REFERENCES ledger_accounts(id, currency)
); 

CREATE INDEX idx_ledger_entries_transaction_id ON ledger_entries(transaction_id); 
CREATE INDEX idx_ledger_entries_account_id ON ledger_entries(account_id);
CREATE INDEX idx_ledger_transactions_source_account ON ledger_transactions(source_account_id);
CREATE INDEX idx_ledger_transactions_destination_account ON ledger_transactions(destination_account_id);

