ALTER TABLE schedules
    ADD COLUMN IF NOT EXISTS paused_at timestamp without time zone,
    ADD COLUMN IF NOT EXISTS paused_reason text;

CREATE INDEX CONCURRENTLY IF NOT EXISTS schedules_connector_id_paused_at ON schedules (connector_id, paused_at);

CREATE INDEX CONCURRENTLY IF NOT EXISTS workflows_instances_error ON workflows_instances (error);
