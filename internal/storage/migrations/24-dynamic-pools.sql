-- This is not a critical table, so we can add a default and locking the table
-- without worries.
ALTER TABLE pools ADD COLUMN IF NOT EXISTS type text NOT NULL DEFAULT 'STATIC';
ALTER TABLE pools ADD COLUMN IF NOT EXISTS query jsonb;