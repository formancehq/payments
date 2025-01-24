DO $$
    BEGIN
        IF (SELECT count(*) FROM information_schema.tables WHERE table_schema ='transfers' AND table_name ='transfer_initiation') > 0
        THEN
            INSERT INTO payment_initiations (id, connector_id, reference, created_at, scheduled_at, description, amount, asset, source_account_id, destination_account_id, metadata, type)
            SELECT id, connector_id, reference, created_at, COALESCE(scheduled_at, '0001-01-01 00:00:00+00'::timestamp without time zone) as scheduled_at, description, amount, asset, source_account_id, destination_account_id, metadata, 
            CASE type+1 
                WHEN 1 THEN 'TRANSFER'
                WHEN 2 THEN 'PAYOUT'
                ELSE 'UNKNOWN'
            END as type 
            from transfers.transfer_initiation
            ON CONFLICT (id) DO NOTHING;
        END IF;
    END;
$$;

DO $$
    BEGIN
        IF (SELECT count(*) FROM information_schema.tables WHERE table_schema ='transfers' AND table_name ='transfer_initiation_payments') > 0
        THEN
            INSERT INTO payment_initiation_related_payments (payment_initiation_id, payment_id, created_at)
            SELECT transfer_initiation_id, payment_id, created_at from transfers.transfer_initiation_payments
            ON CONFLICT (payment_initiation_id, payment_id) DO NOTHING;
        END IF;
    END;
$$;