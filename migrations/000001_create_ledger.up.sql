CREATE TABLE ledger_accounts ( 
    id BIGSERIAL PRIMARY KEY, 
    owner_type VARCHAR(20) NOT NULL, 
    owner_id BIGINT NOT NULL, 
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
    currency CHAR(3) NOT NULL DEFAULT 'INR', 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    CHECK(source_account_id <> destination_account_id), UNIQUE(id, currency)
); 


CREATE TABLE ledger_entries ( 
    id BIGSERIAL PRIMARY KEY, 
    transaction_id BIGINT NOT NULL,
    transaction_type BIGINT NOT NULL, 
    account_id BIGINT NOT NULL, 
    currency CHAR(3) NOT NULL, 
    entry_type VARCHAR(10) NOT NULL CHECK(entry_type IN ('debit', 'credit')), 
    amount_minor BIGINT NOT NULL CHECK(amount_minor > 0), 
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, 
    FOREIGN KEY(transaction_id, currency) REFERENCES ledger_transactions(id, currency), 
    FOREIGN KEY(account_id, currency) REFERENCES ledger_accounts(id, currency)
); 

CREATE INDEX idx_ledger_entries_transaction_id ON ledger_entries(transaction_id); 
CREATE INDEX idex_ledger_entries_account_id ON ledger_entries(account_id);

