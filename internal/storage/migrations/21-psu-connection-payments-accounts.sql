alter table payments add column if not exists psu_id uuid;
alter table payments add constraint payments_psu_id_fk foreign key (psu_id) references payment_service_users (id) on delete cascade;

alter table payments add column if not exists open_banking_connection_id varchar;
alter table payments add constraint payments_open_banking_connection_id_fk foreign key (psu_id, connector_id, open_banking_connection_id) references psu_bank_bridge_connections (psu_id, connector_id, connection_id) on delete cascade;

alter table accounts add column if not exists psu_id uuid;
alter table accounts add constraint accounts_psu_id_fk foreign key (psu_id) references payment_service_users (id) on delete cascade;

alter table accounts add column if not exists open_banking_connection_id varchar;
alter table accounts add constraint accounts_open_banking_connection_id_fk foreign key (psu_id, connector_id, open_banking_connection_id) references psu_bank_bridge_connections (psu_id, connector_id, connection_id) on delete cascade;