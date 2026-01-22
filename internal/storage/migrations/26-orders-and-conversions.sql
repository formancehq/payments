-- Orders table for trading orders on crypto exchanges
create table if not exists orders (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id             varchar not null,
    connector_id   varchar not null,
    reference      text not null,
    created_at     timestamp without time zone not null,
    updated_at     timestamp without time zone not null,
    direction      text not null,
    source_asset   text not null,
    target_asset   text not null,
    type           text not null,
    status         text not null,
    base_quantity_ordered numeric not null,
    time_in_force  text not null,

    -- Optional fields
    base_quantity_filled numeric,
    limit_price    numeric,
    expires_at     timestamp without time zone,
    fee            numeric,
    fee_asset      text,
    average_fill_price numeric,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index orders_created_at_sort_id on orders (created_at, sort_id);
create index orders_connector_id on orders (connector_id);
create index orders_status on orders (status);
create index orders_reference on orders (reference);
alter table orders
    add constraint orders_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;

-- Order adjustments table for tracking order status changes
create table if not exists order_adjustments (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id          varchar not null,
    order_id    varchar not null,
    reference   text not null,
    created_at  timestamp without time zone not null,
    status      text not null,
    raw         json not null,

    -- Optional fields
    base_quantity_filled numeric,
    fee         numeric,
    fee_asset   text,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Primary key
    primary key (id)
);
create index order_adjustments_created_at_sort_id on order_adjustments (created_at, sort_id);
alter table order_adjustments
    add constraint order_adjustments_order_id_fk foreign key (order_id)
    references orders (id)
    on delete cascade;

-- Conversions table for stablecoin conversions (USD↔USDC, USD↔PYUSD)
create table if not exists conversions (
    -- Autoincrement fields
    sort_id bigserial not null,

    -- Mandatory fields
    id             varchar not null,
    connector_id   varchar not null,
    reference      text not null,
    created_at     timestamp without time zone not null,
    updated_at     timestamp without time zone not null,
    source_asset   text not null,
    target_asset   text not null,
    source_amount  numeric not null,
    status         text not null,
    wallet_id      text not null,

    -- Optional fields
    target_amount  numeric,

    -- Optional fields with default
    metadata jsonb not null default '{}'::jsonb,

    -- Raw PSP response
    raw json not null,

    -- Primary key
    primary key (id)
);
create index conversions_created_at_sort_id on conversions (created_at, sort_id);
create index conversions_connector_id on conversions (connector_id);
create index conversions_status on conversions (status);
create index conversions_reference on conversions (reference);
alter table conversions
    add constraint conversions_connector_id_fk foreign key (connector_id)
    references connectors (id)
    on delete cascade;
