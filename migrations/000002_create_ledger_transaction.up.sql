CREATE TABLE ledger_transactions ( 
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    idempotency_key VARCHAR(100) NOT NULL UNIQUE, 
    request_hash VARCHAR(64) NOT NULL CHECK(request_hash ~ '^[0-9a-f]{64}$'),
    transaction_type VARCHAR(30) NOT NULL CHECK(transaction_type IN('transfer', 'opening_balance', 'fee', 'refund', 'reversal', 'settlement', 'adjustment')),  
    amount_minor BIGINT NOT NULL CHECK(amount_minor > 0), 
    currency VARCHAR(3) NOT NULL DEFAULT 'INR' CHECK (currency ~ '^[A-Z]{3}$'), 
    transaction_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK(transaction_status IN ('pending', 'posted', 'failed')),
    reference VARCHAR(150),
    reverses_transaction_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), 
    posted_at TIMESTAMPTZ,
    UNIQUE(id, currency),
    FOREIGN KEY(reverses_transaction_id) REFERENCES ledger_transactions(id),
    CHECK (reverses_transaction_id IS NULL OR reverses_transaction_id <> id),
    CHECK (
        (transaction_status = 'posted' AND posted_at IS NOT NULL) OR 
        (transaction_status <> 'posted' AND posted_at IS NULL)
    )
);

CREATE INDEX idx_ledger_transactions_posted ON ledger_transactions (posted_at DESC, id DESC) WHERE transaction_status = 'posted';