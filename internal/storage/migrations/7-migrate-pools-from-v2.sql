DO $$
    BEGIN
        IF (SELECT count(*) FROM information_schema.tables WHERE table_schema ='accounts' AND table_name ='pools') > 0
        THEN
            INSERT INTO pools (id, name, created_at)
            SELECT id, name, created_at FROM accounts.pools;
        END IF;
    END;
$$;

DO $$
    BEGIN
        IF (SELECT count(*) FROM information_schema.tables WHERE table_schema ='accounts' AND table_name ='pool_accounts') > 0
        THEN
            INSERT INTO pool_accounts (pool_id, account_id)
            SELECT pool_id, account_id FROM accounts.pool_accounts;
        END IF;
    END;
$$;