CREATE INDEX CONCURRENTLY IF NOT EXISTS schedules_connector_id_paused_at ON schedules (connector_id, paused_at);
