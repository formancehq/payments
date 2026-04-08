ALTER TABLE schedules
    ADD COLUMN IF NOT EXISTS paused_at timestamp without time zone,
    ADD COLUMN IF NOT EXISTS paused_reason text;
