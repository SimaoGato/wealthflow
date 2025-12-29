-- WealthFlow Initial Schema Rollback
-- Drops all tables in reverse order of dependencies

-- Drop tables in reverse order to respect foreign key constraints
DROP TABLE IF EXISTS wants;
DROP TABLE IF EXISTS transfer_tasks;
DROP TABLE IF EXISTS transaction_entries;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS split_rule_items;
DROP TABLE IF EXISTS split_rules;
DROP TABLE IF EXISTS market_value_history;
DROP TABLE IF EXISTS buckets;

