-- payment service users
create table if not exists payment_service_users (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id uuid not null,
    created_at timestamp without time zone not null,
    
    -- Encrypted fields
    name bytea,
    street_name bytea,
    street_number bytea,
    postal_code bytea,
    city bytea,
    region bytea,
    country bytea,
    email bytea,
    phone_number bytea,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);

create index psu_created_at_sort_id on payment_service_users (created_at, sort_id);

create table if not exists psu_bank_accounts (
    -- Mandatory fields
    psu_id uuid not null,
    bank_account_id uuid not null,

    primary key (psu_id, bank_account_id)
);

alter table psu_bank_accounts
    add constraint fk_psu_bank_accounts_psu_id
    foreign key (psu_id)
    references payment_service_users (id)
    on delete cascade;
alter table psu_bank_accounts
    add constraint fk_psu_bank_accounts_bank_account_id
    foreign key (bank_account_id)
    references bank_accounts (id)
    on delete cascade;