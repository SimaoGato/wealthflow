-- WealthFlow Initial Schema Migration
-- Creates all core tables for the double-entry ledger system

-- 1. Buckets (Physical, Virtual, Income, Expense, Equity, System)
CREATE TABLE buckets (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    bucket_type VARCHAR NOT NULL, -- 'PHYSICAL', 'VIRTUAL', 'INCOME', 'EXPENSE', 'EQUITY', 'SYSTEM'
    parent_physical_bucket_id UUID, -- NULL if PHYSICAL/INCOME/EXPENSE. NOT NULL if VIRTUAL.
    current_balance DECIMAL DEFAULT 0, -- Represents BOOK VALUE (Cash in/out)
    FOREIGN KEY (parent_physical_bucket_id) REFERENCES buckets(id)
);

-- 1.1 Market Value History (For charting and historical P/L)
CREATE TABLE market_value_history (
    id UUID PRIMARY KEY,
    bucket_id UUID NOT NULL REFERENCES buckets(id),
    date TIMESTAMP DEFAULT NOW(),
    market_value DECIMAL NOT NULL -- The actual value at this point in time
);

-- 2. Split Rules
CREATE TABLE split_rules (
    id UUID PRIMARY KEY,
    name TEXT,
    source_bucket_id UUID REFERENCES buckets(id)
);

CREATE TABLE split_rule_items (
    id UUID PRIMARY KEY,
    split_rule_id UUID NOT NULL REFERENCES split_rules(id) ON DELETE CASCADE,
    target_bucket_id UUID NOT NULL REFERENCES buckets(id), -- Enforces Referential Integrity
    rule_type VARCHAR NOT NULL, -- 'FIXED', 'PERCENT' (of Remainder), or 'REMAINDER' (Catch-all)
    value DECIMAL NOT NULL,
    priority INT NOT NULL -- Lower number = Executed first (Important for Fixed logic)
);

-- 3. Transactions (Double Entry Header)
CREATE TABLE transactions (
    id UUID PRIMARY KEY,
    description TEXT,
    date TIMESTAMP DEFAULT NOW(),
    is_internal_transfer BOOLEAN DEFAULT FALSE,
    is_external_inflow BOOLEAN DEFAULT FALSE
);

CREATE TABLE transaction_entries (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    bucket_id UUID NOT NULL REFERENCES buckets(id),
    amount DECIMAL NOT NULL, -- ABSOLUTE VALUE (Always Positive)
    type VARCHAR NOT NULL, -- 'DEBIT' or 'CREDIT' (Source of Truth for direction)
    layer VARCHAR DEFAULT 'PHYSICAL' -- 'PHYSICAL', 'VIRTUAL'.
);

-- 4. Transfer Tasks (The Bridge to Reality)
CREATE TABLE transfer_tasks (
    id UUID PRIMARY KEY,
    related_transaction_id UUID, -- The Virtual move that caused this task
    completed_transaction_id UUID, -- The Physical move that resolved this task
    from_physical_bucket_id UUID REFERENCES buckets(id),
    to_physical_bucket_id UUID REFERENCES buckets(id),
    amount DECIMAL,
    is_completed BOOLEAN DEFAULT FALSE
);

-- 5. Wants (Wishlist Engine)
CREATE TABLE wants (
    id UUID PRIMARY KEY,
    title TEXT,
    url TEXT,
    price DECIMAL,
    linked_virtual_bucket_id UUID REFERENCES buckets(id),
    priority INT NOT NULL DEFAULT 99, -- Low number = High Priority
    status VARCHAR -- 'QUEUED', 'READY_TO_BUY', 'PURCHASED'
);

