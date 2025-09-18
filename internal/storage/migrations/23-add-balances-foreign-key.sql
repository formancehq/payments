alter table balances
    add column if not exists psu_id uuid;
alter table balances
    add constraint balances_psu_id_fk foreign key (psu_id)
        references payment_service_users (id)
        on delete cascade;
alter table balances
    add column if not exists open_banking_connection_id varchar;
alter table balances
    add constraint balances_open_banking_connection_id_fk foreign key (psu_id, connector_id, open_banking_connection_id)
        references open_banking_connections (psu_id, connector_id, connection_id)
        on delete cascade;

create index if not exists balances_psu_id_idx
    on balances (psu_id);
create index if not exists balances_open_banking_connection_id_idx
    on balances (psu_id, connector_id, open_banking_connection_id);