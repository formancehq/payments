create table if not exists bank_bridge_connection_attempts (
    sort_id bigserial not null,

    -- Mandatory fields
    id uuid not null,
    psu_id uuid not null,
    created_at timestamp without time zone not null,
    connector_id varchar not null,
    status text not null,

    -- Optional fields
    client_redirect_url text,
    temporary_token text,
    expires_at timestamp without time zone,
    state jsonb,
    error text,

    -- Primary key
    primary key (id)
);

create index bank_bridge_connection_attempts_created_at_sort_id on bank_bridge_connection_attempts (created_at, sort_id);
create index bank_bridge_connection_attempts_connector_id on bank_bridge_connection_attempts (connector_id);
create index bank_bridge_connection_attempts_psu_id on bank_bridge_connection_attempts (psu_id);
alter table bank_bridge_connection_attempts 
    add constraint bank_bridge_connection_attempts_connector_id_fk foreign key (connector_id) 
    references connectors (id)
    on delete cascade;

alter table bank_bridge_connection_attempts
    add constraint bank_bridge_connection_attempts_psu_id_fk foreign key (psu_id) 
    references payment_service_users (id)
    on delete cascade;

create table if not exists psu_bank_bridges (
    sort_id bigserial not null,

    -- Mandatory fields
    psu_id uuid not null,
    connector_id varchar not null,

    -- Optional fields
    access_token text,
    expires_at timestamp without time zone,
    metadata jsonb,

    -- Primary key
    primary key (psu_id, connector_id)
);

create index psu_bank_bridges_connector_id on psu_bank_bridges (connector_id);
create index psu_bank_bridges_psu_id on psu_bank_bridges (psu_id);
alter table psu_bank_bridges 
    add constraint psu_bank_bridges_connector_id_fk foreign key (connector_id) 
    references connectors (id)
    on delete cascade;

alter table psu_bank_bridges 
    add constraint psu_bank_bridges_psu_id_fk foreign key (psu_id) 
    references payment_service_users (id)
    on delete cascade;

create table if not exists psu_bank_bridge_connections (
    sort_id bigserial not null,

    -- Mandatory fields
    psu_id uuid not null,
    connector_id varchar not null,
    connection_id varchar not null,
    created_at timestamp without time zone not null,
    data_updated_at timestamp without time zone not null,
    status text not null,

    -- Optional fields
    access_token text,
    expires_at timestamp without time zone,
    error text,
    metadata jsonb,

    -- Primary key
    primary key (psu_id, connector_id, connection_id)
);

create index psu_bank_bridge_connections_connector_id on psu_bank_bridge_connections (connector_id);
create index psu_bank_bridge_connections_psu_id on psu_bank_bridge_connections (psu_id);
create index psu_bank_bridge_connections_created_at_sort_id on psu_bank_bridge_connections (created_at, sort_id);
alter table psu_bank_bridge_connections
    add constraint psu_bank_bridge_connections_psu_id_fk foreign key (psu_id) 
    references payment_service_users (id)
    on delete cascade;

alter table psu_bank_bridge_connections 
    add constraint psu_bank_bridge_connections_connector_id_fk foreign key (connector_id) 
    references connectors (id)
    on delete cascade;