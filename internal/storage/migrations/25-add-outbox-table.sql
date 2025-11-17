-- Outbox events table for reliable event publishing
create table if not exists outbox_events (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id uuid not null,
    idempotency_key text not null,
    event_type text not null,
    entity_id varchar not null,
    payload jsonb not null,
    created_at timestamp without time zone not null,
    status text not null default 'pending',

    -- Optional fields
    connector_id varchar,
    retry_count integer not null default 0,
    last_retry_at timestamp without time zone,
    error text,

    -- Primary key
    primary key (id)
);

create unique index outbox_events_unique_sort_id on outbox_events (sort_id);
create unique index outbox_events_unique_idempotency_key_connector_id on outbox_events (idempotency_key, connector_id);
create index outbox_events_status_created_at on outbox_events (status, created_at);
create index outbox_events_connector_id on outbox_events (connector_id);

alter table outbox_events
    add constraint outbox_events_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete CASCADE; -- maybe debatable, but events_sent is using delete cascade
