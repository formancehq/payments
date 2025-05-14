alter table webhooks
    add column if not exists idempotency_key text;

create unique index webhooks_unique_idempotency_key on webhooks (connector_id, idempotency_key);
