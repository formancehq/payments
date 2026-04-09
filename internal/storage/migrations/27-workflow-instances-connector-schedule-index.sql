CREATE INDEX CONCURRENTLY IF NOT EXISTS workflows_instances_connector_schedule_created
    ON workflows_instances (connector_id, schedule_id, created_at DESC);
