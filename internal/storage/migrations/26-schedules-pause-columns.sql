ALTER TABLE schedules
    ADD COLUMN paused_at timestamp without time zone,
    ADD COLUMN paused_reason text;

ALTER TABLE connectors
    ADD COLUMN updated_at timestamp without time zone;

UPDATE connectors SET updated_at = created_at WHERE updated_at IS NULL;

CREATE OR REPLACE FUNCTION set_connectors_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS connectors_updated_at_trigger ON connectors;

CREATE TRIGGER connectors_updated_at_trigger
    BEFORE UPDATE ON connectors
    FOR EACH ROW
    EXECUTE FUNCTION set_connectors_updated_at();

