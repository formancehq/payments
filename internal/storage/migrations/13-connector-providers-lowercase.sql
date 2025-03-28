DO $$
    BEGIN
        UPDATE connectors SET provider = CASE provider
            WHEN 'CURRENCY-CLOUD' THEN 'currencycloud'
            WHEN 'BANKING-CIRCLE' THEN 'bankingcircle'
            WHEN 'DUMMY-PAY' THEN 'dummypay'
            ELSE LOWER(provider)
        END;
    END;
$$;