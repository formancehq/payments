DO $$
    BEGIN
        IF (SELECT count(*) FROM information_schema.tables WHERE table_schema ='accounts' AND table_name ='bank_account') > 0
        THEN
            INSERT INTO bank_accounts (id, created_at, name, country, account_number, iban, swift_bic_code, metadata)
            SELECT id, created_at, name, country, account_number, iban, swift_bic_code, metadata from accounts.bank_account
            On CONFLICT (id) DO NOTHING;
        END IF;
    END;
$$;

DO $$
    BEGIN
        IF (SELECT count(*) FROM information_schema.tables WHERE table_schema ='accounts' AND table_name ='bank_account_related_accounts') > 0
        THEN
            INSERT INTO bank_accounts_related_accounts (bank_account_id, account_id, connector_id, created_at)
            SELECT bank_account_id, account_id, connector_id, created_at from accounts.bank_account_related_accounts
            ON CONFLICT (bank_account_id, account_id) DO NOTHING;
        END IF;
    END;
$$;