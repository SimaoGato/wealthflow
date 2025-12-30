-- WealthFlow Balance Update Trigger Migration
-- Automatically updates bucket.current_balance when transaction_entries are inserted

-- Create PL/pgSQL function to update bucket balance
CREATE OR REPLACE FUNCTION update_bucket_balance()
RETURNS TRIGGER AS $$
BEGIN
    -- DEBIT increases balance (Asset increase)
    IF NEW.type = 'DEBIT' THEN
        UPDATE buckets 
        SET current_balance = current_balance + NEW.amount 
        WHERE id = NEW.bucket_id;
    -- CREDIT decreases balance (Asset decrease)
    ELSIF NEW.type = 'CREDIT' THEN
        UPDATE buckets 
        SET current_balance = current_balance - NEW.amount 
        WHERE id = NEW.bucket_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger that fires AFTER INSERT on transaction_entries
CREATE TRIGGER balance_update_trigger
    AFTER INSERT ON transaction_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_bucket_balance();

