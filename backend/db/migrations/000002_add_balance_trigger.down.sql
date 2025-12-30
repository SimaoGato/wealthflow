-- WealthFlow Balance Update Trigger Rollback
-- Drops the trigger and function

DROP TRIGGER IF EXISTS balance_update_trigger ON transaction_entries;
DROP FUNCTION IF EXISTS update_bucket_balance();

