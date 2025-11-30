-- Create accounts table
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_number VARCHAR(16) UNIQUE NOT NULL,
    cvv VARCHAR(3) NOT NULL,
    expiry_month INT NOT NULL,
    expiry_year INT NOT NULL,
    balance_cents BIGINT NOT NULL DEFAULT 0,
    available_balance_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_account_number ON accounts(account_number);

-- Create transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id),
    type VARCHAR(20) NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    reference_id UUID,
    status VARCHAR(20) NOT NULL,
    expires_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_transactions_reference_id ON transactions(reference_id);
CREATE INDEX idx_transactions_type_status ON transactions(type, status);

-- Prevent duplicate captures/voids/refunds for the same authorization/capture
CREATE UNIQUE INDEX idx_transactions_reference_type_unique ON transactions(reference_id, type)
WHERE type IN ('CAPTURE', 'VOID', 'REFUND') AND reference_id IS NOT NULL;

-- Create idempotency keys table
CREATE TABLE idempotency_keys (
    key VARCHAR(255) NOT NULL,
    request_path VARCHAR(100) NOT NULL,
    response_status INT NOT NULL,
    response_body TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (key, request_path)
);

-- Seed test accounts
INSERT INTO accounts (account_number, cvv, expiry_month, expiry_year, balance_cents, available_balance_cents) VALUES
    ('4111111111111111', '123', 12, 2030, 1000000, 1000000),   -- $10,000 primary
    ('4242424242424242', '456', 6, 2030, 50000, 50000),        -- $500 secondary
    ('5555555555554444', '789', 9, 2030, 0, 0),                 -- $0 zero balance
    ('5105105105105100', '321', 3, 2020, 500000, 500000);      -- $5,000 expired
