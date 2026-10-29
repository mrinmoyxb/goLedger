CREATE TABLE ledger_entries ( 
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    transaction_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,           
    currency VARCHAR(3) NOT NULL CHECK(currency ~ '^[A-Z]{3}$'), 
    entry_type VARCHAR(10) NOT NULL CHECK(entry_type IN ('debit', 'credit')), 
    amount_minor BIGINT NOT NULL CHECK(amount_minor > 0), 
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
    FOREIGN KEY(transaction_id, currency) REFERENCES ledger_transactions(id, currency), 
    FOREIGN KEY(account_id, currency) REFERENCES ledger_accounts(id, currency),
    UNIQUE(id, transaction_id)
); 

CREATE INDEX idx_ledger_entries_transaction ON ledger_entries (transaction_id, id); 
CREATE INDEX idx_ledger_entries_account_created ON ledger_entries (account_id, created_at, id);