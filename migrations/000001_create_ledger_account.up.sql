CREATE TABLE ledger_accounts ( 
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    owner_type VARCHAR(20) NOT NULL CHECK(owner_type IN ('user', 'merchant', 'platform', 'system')), 
    owner_id BIGINT NOT NULL, 
    account_type VARCHAR(20) NOT NULL CHECK(account_type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
    purpose VARCHAR(30) NOT NULL DEFAULT 'main',
    currency VARCHAR(3) NOT NULL DEFAULT 'INR' CHECK(currency ~ '^[A-Z]{3}$'), 
    balance_minor BIGINT NOT NULL DEFAULT 0, 
    account_status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK(account_status IN ('active', 'frozen', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    UNIQUE (id, currency),
    UNIQUE (owner_type, owner_id, account_type, purpose, currency),
    CHECK((account_status = 'closed' AND closed_at IS NOT NULL) OR (account_status <> 'closed' AND closed_at IS NULL))
); 
 


CREATE INDEX idx_ledger_accounts_owner ON ledger_accounts (owner_type, owner_id);

