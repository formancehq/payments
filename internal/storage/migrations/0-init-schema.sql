create extension if not exists pgcrypto;

-- connectors
create table if not exists connectors (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id         varchar not null,
    name       text not null,
    created_at timestamp without time zone not null,
    provider   text not null,
    scheduled_for_deletion boolean not null default false,

    -- Optional fields
    config bytea,

    -- Primary key
    primary key (id)
);
create index connectors_created_at_sort_id on connectors (created_at, sort_id);
create unique index connectors_unique_name on connectors (name);

CREATE OR REPLACE FUNCTION connectors_notify_after_modifications() RETURNS TRIGGER as $$
BEGIN
    IF (TG_OP = 'DELETE') THEN
        PERFORM pg_notify('connectors', 'delete_' || OLD.id);
        RETURN NULL;
    ELSIF (TG_OP = 'INSERT') THEN
        PERFORM pg_notify('connectors', 'insert_' || NEW.id);
        RETURN NULL;
    ELSE
        PERFORM pg_notify('connectors', 'update_' || NEW.id);
        RETURN NULL;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER connectors_trigger
AFTER INSERT OR UPDATE OR DELETE ON connectors FOR EACH ROW
EXECUTE PROCEDURE connectors_notify_after_modifications();

-- accounts
create table if not exists accounts (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id           varchar not null,
    connector_id varchar not null,
    created_at   timestamp without time zone not null,
    reference    text not null,
    type         text not null,
    raw          json not null,

    -- Optional fields
    default_asset text,
    name          text,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index accounts_created_at_sort_id on accounts (created_at, sort_id);
alter table accounts 
    add constraint accounts_connector_id_fk foreign key (connector_id) 
    references connectors (id)
    on delete cascade;

-- balances
create table if not exists balances (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    account_id      varchar not null,
    connector_id    varchar not null,
    created_at      timestamp without time zone not null,
    last_updated_at timestamp without time zone not null,
    asset           text not null,
    balance         numeric not null,

    -- Primary key
    primary key (account_id, created_at, asset)
);
create index balances_created_at_sort_id on balances (created_at, sort_id);
create index balances_account_id_created_at_asset on balances (account_id, last_updated_at desc, asset);
alter table balances
    add constraint balances_connector_id foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- bank accounts
create table if not exists bank_accounts (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id uuid    not null,
    created_at timestamp without time zone not null,
    name       text not null,

    -- Optional fields
    account_number bytea,
    iban           bytea,
    swift_bic_code bytea,
    country        text,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index bank_accounts_created_at_sort_id on bank_accounts (created_at, sort_id);
create table if not exists bank_accounts_related_accounts (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    bank_account_id uuid not null,
    account_id      varchar not null,
    connector_id    varchar not null,
    created_at      timestamp without time zone not null,

    -- Primary key
    primary key (bank_account_id, account_id)
);
create index bank_accounts_related_accounts_created_at_sort_id on bank_accounts_related_accounts (created_at, sort_id);
alter table bank_accounts_related_accounts
    add constraint bank_accounts_related_accounts_bank_account_id_fk foreign key (bank_account_id)
    references bank_accounts (id)
    on delete cascade;
alter table bank_accounts_related_accounts
    add constraint bank_accounts_related_accounts_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- payments
create table if not exists payments (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id             varchar not null,
    connector_id   varchar not null,
    reference      text not null,
    created_at     timestamp without time zone not null,
    type           text not null,
    initial_amount numeric not null,
    amount         numeric not null,
    asset          text not null,
    scheme         text not null,

    -- Optional fields
    source_account_id      varchar,
    destination_account_id varchar,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index payments_created_at_sort_id on payments (created_at, sort_id);
alter table payments
    add constraint payments_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- payment adjustments
create table if not exists payment_adjustments (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id          varchar not null,
    payment_id  varchar not null,
    reference   text not null,
    created_at  timestamp without time zone not null,
    status      text not null,
    raw         json not null,

    -- Optional fields
    amount numeric,
    asset  text,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index payment_adjustments_created_at_sort_id on payment_adjustments (created_at, sort_id);
alter table payment_adjustments
    add constraint payment_adjustments_payment_id_fk foreign key (payment_id)
    references payments (id)
    on delete cascade;

-- pools
create table if not exists pools (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id         uuid not null,
    name       text not null,
    created_at timestamp without time zone not null,

    -- Primary key
    primary key (id)
);
create index pools_created_at_sort_id on pools (created_at, sort_id);
create unique index pools_unique_name on pools (name);

create table if not exists pool_accounts (
    -- Autoincrement fields
    sort_id     bigserial not null,

    -- Mandatory fields
    pool_id      uuid not null,
    account_id   varchar not null,
    connector_id varchar not null,

    -- Primary key
    primary key (pool_id, account_id)
);
create unique index pool_accounts_unique_sort_id on pool_accounts (sort_id);
alter table pool_accounts
    add constraint pool_accounts_pool_id_fk foreign key (pool_id)
    references pools (id)
    on delete cascade;

-- schedules
create table if not exists schedules (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id text not null,
    connector_id varchar not null,
    created_at timestamp without time zone not null,

    -- Primary key
    primary key (id, connector_id)
);
create index schedules_created_at_sort_id on schedules (created_at, sort_id);
alter table schedules
    add constraint schedules_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- states
create table if not exists states (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id           varchar not null,
    connector_id varchar not null,

    -- Optional fields with default
    state json not null default '{}'::json,

    -- Primary key
    primary key (id)
);
create unique index states_unique_sort_id on states (sort_id);
alter table states
    add constraint states_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- connector tasks tree
create table if not exists connector_tasks_tree (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    connector_id varchar not null,
    tasks        json not null,

    -- Primary key
    primary key (connector_id)
);
create unique index connector_tasks_tree_unique_sort_id on connector_tasks_tree (sort_id);
alter table connector_tasks_tree
    add constraint connector_tasks_tree_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- Workflow instance
create table if not exists workflows_instances (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id           text not null,
    schedule_id  text not null,
    connector_id varchar not null,
    created_at   timestamp without time zone not null,
    updated_at   timestamp without time zone not null,

    -- Optional fields with default
    terminated boolean not null default false,

    -- Optional fields
    terminated_at timestamp without time zone,
    error         text,

    -- Primary key
    primary key (id, schedule_id, connector_id)
);
create index workflows_instances_created_at_sort_id on workflows_instances (created_at, sort_id);
alter table workflows_instances
    add constraint workflows_instances_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;
alter table workflows_instances
    add constraint workflows_instances_schedule_id_fk foreign key (schedule_id, connector_id)
    references schedules (id, connector_id)
    on delete cascade;

-- Webhook configs
create table if not exists webhooks_configs (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    name         text not null,
    connector_id varchar not null,
    url_path     text not null,

    -- Primary key
    primary key (name, connector_id)
);
create unique index webhooks_configs_unique_sort_id on webhooks_configs (sort_id);
alter table webhooks_configs
    add constraint webhooks_configs_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- Webhooks
create table if not exists webhooks (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id           text not null,
    connector_id varchar not null,

    -- Optional fields
    headers      json,
    query_values json,
    body         bytea,

    -- Primary key
    primary key (id)
);
create unique index webhooks_unique_sort_id on webhooks (sort_id);
alter table webhooks
    add constraint webhooks_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- Payment Initiations
create table if not exists payment_initiations (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id                     text not null,
    connector_id           varchar not null,
    reference              text not null,
    created_at             timestamp without time zone not null,
    scheduled_at           timestamp without time zone not null,
    description            text not null,
    type                   text not null,
    amount                 numeric not null,
    asset                  text not null,

    -- Optional fields
    source_account_id varchar,
    destination_account_id varchar,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index payment_initiations_created_at_sort_id on payment_initiations (created_at, sort_id);
alter table payment_initiations
    add constraint payment_initiations_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- Payment Initiation Related Payments
create table if not exists payment_initiation_related_payments(
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    payment_initiation_id varchar not null,
    payment_id            varchar not null,
    created_at           timestamp without time zone not null,

    -- Primary key
    primary key (payment_initiation_id, payment_id)
);
create index payment_initiation_related_payments_created_at_sort_id on payment_initiation_related_payments (created_at, sort_id);
alter table payment_initiation_related_payments
    add constraint payment_initiation_related_payments_payment_initiation_id_fk foreign key (payment_initiation_id)
    references payment_initiations (id)
    on delete cascade;

-- Payment Initiation Adjustments
create table if not exists payment_initiation_adjustments(
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id                    varchar not null,
    payment_initiation_id varchar not null,
    created_at            timestamp without time zone not null,
    status                text not null,

    -- Optional fields
    error  text,
    amount numeric,
    asset  text,

     -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index payment_initiation_adjustments_created_at_sort_id on payment_initiation_adjustments (created_at, sort_id);
create index payment_initiation_adjustments_pi_id on payment_initiation_adjustments (payment_initiation_id);
create index payment_initiation_adjustments_sort_id on payment_initiation_adjustments (sort_id);
alter table payment_initiation_adjustments
    add constraint payment_initiation_adjustments_payment_initiation_id_fk foreign key (payment_initiation_id)
    references payment_initiations (id)
    on delete cascade;

-- Payment Initiations reversals
create table if not exists payment_initiation_reversals (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id                     text not null,
    connector_id           varchar not null,
    payment_initiation_id  varchar not null,
    reference              text not null,
    created_at             timestamp without time zone not null,
    description            text not null,
    amount                 numeric not null,
    asset                  text not null,
    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,
    -- Primary key
    primary key (id)
);
create index payment_initiation_reversals_created_at_sort_id on payment_initiation_reversals (created_at, sort_id);
alter table payment_initiation_reversals
    add constraint payment_initiation_reversals_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;
alter table payment_initiation_reversals
    add constraint payment_initiation_reversals_payment_initiation_id_fk foreign key (payment_initiation_id)
    references payment_initiations (id)
    on delete cascade;
-- Payment Initiation Reversal Adjustments
create table if not exists payment_initiation_reversal_adjustments(
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id                             varchar not null,
    payment_initiation_reversal_id varchar not null,
    created_at                     timestamp without time zone not null,
    status                         text not null,
    -- Optional fields
    error                 text,
     -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,
    -- Primary key
    primary key (id)
);
create index payment_initiation_reversal_adjustments_created_at_sort_id on payment_initiation_reversal_adjustments (created_at, sort_id);
alter table payment_initiation_reversal_adjustments
    add constraint payment_initiation_reversal_adjustments_payment_initiation_id_fk foreign key (payment_initiation_reversal_id)
    references payment_initiation_reversals (id)
    on delete cascade;

-- Events sent
create table if not exists events_sent (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id           varchar not null,
    sent_at      timestamp without time zone not null,

    -- Optional fields
    connector_id varchar,

    -- Primary key
    primary key (id)
);
create unique index events_sent_unique_sort_id on events_sent (sort_id);
alter table events_sent
    add constraint events_sent_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- tasks
create table if not exists tasks (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id varchar not null,
    status text not null,
    created_at timestamp without time zone not null,
    updated_at timestamp without time zone not null,

    -- Optional fields
    connector_id varchar,
    created_object_id varchar,
    error text,

    -- Primary key
    primary key (id)
);
create index tasks_created_at_sort_id on tasks (created_at, sort_id);
alter table tasks
    add constraint tasks_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;
