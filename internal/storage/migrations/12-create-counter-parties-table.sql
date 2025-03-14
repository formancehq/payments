-- counter parties
create table if not exists counter_parties (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id uuid    not null,
    created_at timestamp without time zone not null,

    -- Encrypted fields
    name bytea,
    street_name bytea,
    street_number bytea,
    postal_code bytea,
    city bytea,
    country bytea,
    email bytea,
    phone bytea,

    -- Optional fields
    bank_account_id uuid,

     -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index counter_parties_created_at_sort_id on counter_parties (created_at, sort_id);
alter table counter_parties
    add constraint counter_parties_bank_account_id_fkey foreign key (bank_account_id)
    references bank_accounts (id)
    on delete set null;

create table if not exists counter_parties_related_accounts (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    counter_party_id uuid not null,
    account_id      varchar not null,
    connector_id    varchar not null,
    created_at      timestamp without time zone not null,

    -- Primary key
    primary key (counter_party_id, account_id)
);
create index counter_parties_related_accounts_created_at_sort_id on counter_parties_related_accounts (created_at, sort_id);
alter table counter_parties_related_accounts
    add constraint counter_parties_related_accounts_counter_party_id_fk foreign key (counter_party_id)
    references counter_parties (id)
    on delete cascade;
alter table counter_parties_related_accounts
    add constraint counter_parties_related_accounts_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;
