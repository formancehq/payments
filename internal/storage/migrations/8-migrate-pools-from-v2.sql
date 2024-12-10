DO $$
    BEGIN
        IF (SELECT count(*) FROM information_schema.tables WHERE table_schema ='accounts' AND table_name ='pools') > 0
        THEN
            INSERT INTO pools (id, name, created_at)
            SELECT id, name, created_at FROM accounts.pools
            ON CONFLICT (id) DO NOTHING;
        END IF;
    END;
$$;

DO $$
    BEGIN
        IF (SELECT count(*) FROM information_schema.tables WHERE table_schema ='accounts' AND table_name ='pool_accounts') > 0
        THEN
            INSERT INTO pool_accounts (pool_id, account_id)
            SELECT pool_id, account_id FROM accounts.pool_accounts
            ON CONFLICT (pool_id, account_id) DO NOTHING;
        END IF;
    END;
$$;