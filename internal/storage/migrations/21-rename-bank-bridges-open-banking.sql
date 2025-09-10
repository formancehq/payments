-- OpenBanking Connection attempts renaming
ALTER TABLE bank_bridge_connection_attempts
    RENAME TO open_banking_connection_attempts;
ALTER TABLE open_banking_connection_attempts
    RENAME CONSTRAINT bank_bridge_connection_attempts_pkey TO open_banking_connections_attempts_pkey;
ALTER TABLE open_banking_connection_attempts
    RENAME CONSTRAINT bank_bridge_connection_attempts_connector_id_fk
    TO open_banking_connection_attempts_connector_id_fk;
ALTER TABLE open_banking_connection_attempts
    RENAME CONSTRAINT bank_bridge_connection_attempts_psu_id_fk
    TO open_banking_connection_attempts_psu_id_fk;
ALTER INDEX bank_bridge_connection_attempts_created_at_sort_id
    RENAME TO open_banking_connection_attempts_created_at_sort_id;
ALTER INDEX bank_bridge_connection_attempts_connector_id
    RENAME TO open_banking_connection_attempts_connector_id;
ALTER INDEX bank_bridge_connection_attempts_psu_id
    RENAME TO open_banking_connection_attempts_psu_id;
ALTER SEQUENCE bank_bridge_connection_attempts_sort_id_seq
    RENAME TO open_banking_connection_attempts_sort_id_seq;

-- OpenBanking Provider PSUs renaming
ALTER TABLE psu_bank_bridges
    RENAME TO open_banking_forwarded_users;
ALTER TABLE open_banking_forwarded_users
    RENAME CONSTRAINT psu_bank_bridges_pkey TO open_banking_forwarded_users_pkey;
ALTER TABLE open_banking_forwarded_users
    RENAME CONSTRAINT psu_bank_bridges_connector_id_fk
    TO open_banking_forwarded_users_connector_id_fk;
ALTER TABLE open_banking_forwarded_users
    RENAME CONSTRAINT psu_bank_bridges_psu_id_fk
    TO open_banking_forwarded_users_psu_id_fk;
ALTER INDEX psu_bank_bridges_connector_id
    RENAME TO psu_open_banking_connector_id;
ALTER INDEX psu_bank_bridges_psu_id
    RENAME TO open_banking_provider_psu_psu_id;
ALTER INDEX idx_psu_bank_bridges_psp_user_id
    RENAME TO idx_open_banking_provider_psu_psp_user_id;
ALTER SEQUENCE psu_bank_bridges_sort_id_seq
    RENAME TO open_banking_forwarded_users_sort_id_seq;

-- Open Banking Connections renaming
ALTER TABLE psu_bank_bridge_connections
    RENAME TO open_banking_connections;
ALTER TABLE open_banking_connections
    RENAME CONSTRAINT psu_bank_bridge_connections_pkey TO open_banking_connections_pkey;
ALTER TABLE open_banking_connections
    RENAME CONSTRAINT psu_bank_bridge_connections_psu_id_fk
    TO open_banking_connections_psu_id_fk;
ALTER TABLE open_banking_connections
    RENAME CONSTRAINT psu_bank_bridge_connections_connector_id_fk
    TO open_banking_connections_connector_id_fk;
ALTER INDEX psu_bank_bridge_connections_connector_id
    RENAME TO open_banking_connections_connector_id;
ALTER INDEX psu_bank_bridge_connections_psu_id
    RENAME TO open_banking_connections_psu_id;
ALTER INDEX psu_bank_bridge_connections_created_at_sort_id
    RENAME TO open_banking_connections_created_at_sort_id;;
ALTER SEQUENCE psu_bank_bridge_connections_sort_id_seq
    RENAME TO open_banking_connections_sort_id_seq;

-- Psu OpenBanking access tokens renaming
ALTER TABLE psu_bank_bridge_access_tokens
    RENAME TO open_banking_access_tokens;
ALTER TABLE open_banking_access_tokens
    RENAME CONSTRAINT psu_bank_bridge_access_tokens_pkey TO open_banking_access_tokens_pkey;
ALTER TABLE open_banking_access_tokens
    RENAME CONSTRAINT psu_bank_bridge_access_tokens_psu_id_fk
    TO open_banking_access_tokens_psu_id_fk;
ALTER TABLE open_banking_access_tokens
    RENAME CONSTRAINT psu_bank_bridge_access_tokens_connector_id_fk
    TO open_banking_access_tokens_connector_id_fk;
ALTER SEQUENCE psu_bank_bridge_access_tokens_sort_id_seq
    RENAME TO open_banking_access_tokens_sort_id_seq;
