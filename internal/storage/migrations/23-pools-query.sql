ALTER TABLE pools
    ADD COLUMN IF NOT EXISTS query jsonb;

-- Optional index to speed up querying dynamic pools by query content if needed later
-- CREATE INDEX IF NOT EXISTS pools_query_gin ON pools USING GIN (query);

