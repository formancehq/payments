--
-- V2 Init Schema and Migration Test SQL
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;

CREATE SCHEMA accounts;
CREATE SCHEMA connectors;
CREATE SCHEMA payments;
CREATE SCHEMA tasks;
CREATE SCHEMA transfers;

CREATE TYPE public.account_type AS ENUM (
    'INTERNAL',
    'EXTERNAL',
    'UNKNOWN',
    'EXTERNAL_FORMANCE'
);

CREATE TYPE public.connector_provider AS ENUM (
    'BANKING-CIRCLE',
    'CURRENCY-CLOUD',
    'DUMMY-PAY',
    'MODULR',
    'STRIPE',
    'WISE',
    'MANGOPAY',
    'MONEYCORP',
    'ATLAR',
    'ADYEN',
    'GENERIC'
);

CREATE TYPE public.payment_status AS ENUM (
    'SUCCEEDED',
    'CANCELLED',
    'FAILED',
    'PENDING',
    'OTHER',
    'EXPIRED',
    'REFUNDED',
    'REFUNDED_FAILURE',
    'DISPUTE',
    'DISPUTE_WON',
    'DISPUTE_LOST'
);

CREATE TYPE public.payment_type AS ENUM (
    'PAY-IN',
    'PAYOUT',
    'TRANSFER',
    'OTHER'
);

CREATE TYPE public.task_status AS ENUM (
    'STOPPED',
    'PENDING',
    'ACTIVE',
    'TERMINATED',
    'FAILED'
);

CREATE TYPE public.transfer_status AS ENUM (
    'PENDING',
    'SUCCEEDED',
    'FAILED'
);

CREATE TABLE accounts.account (
    id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    reference text NOT NULL,
    type public.account_type,
    raw_data json,
    default_currency text DEFAULT ''::text NOT NULL,
    account_name text DEFAULT ''::text NOT NULL,
    connector_id character varying NOT NULL,
    metadata jsonb,
    CONSTRAINT account_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE accounts.balances (
    created_at timestamp with time zone NOT NULL,
    account_id character varying NOT NULL,
    currency text NOT NULL,
    balance numeric DEFAULT 0 NOT NULL,
    last_updated_at timestamp with time zone NOT NULL
);

CREATE TABLE accounts.bank_account (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    name text NOT NULL,
    account_number bytea,
    iban bytea,
    swift_bic_code bytea,
    country text,
    metadata jsonb,
    CONSTRAINT bank_account_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE accounts.bank_account_related_accounts (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    bank_account_id uuid NOT NULL,
    connector_id character varying NOT NULL,
    account_id character varying NOT NULL
);

CREATE TABLE accounts.pool_accounts (
    pool_id uuid NOT NULL,
    account_id character varying NOT NULL
);

CREATE TABLE accounts.pools (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT pools_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE connectors.connector (
    id character varying NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    provider public.connector_provider NOT NULL,
    config bytea,
    CONSTRAINT connector_v2_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE connectors.webhook (
    id uuid NOT NULL,
    connector_id character varying NOT NULL,
    request_body bytea NOT NULL
);

CREATE TABLE payments.adjustment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    payment_id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    amount numeric DEFAULT 0 NOT NULL,
    reference text NOT NULL,
    status public.payment_status NOT NULL,
    absolute boolean DEFAULT false NOT NULL,
    raw_data json,
    CONSTRAINT adjustment_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE payments.metadata (
    payment_id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    key text NOT NULL,
    value text NOT NULL,
    changelog jsonb NOT NULL,
    CONSTRAINT metadata_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE payments.payment (
    id character varying NOT NULL,
    connector_id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    reference text NOT NULL,
    type public.payment_type NOT NULL,
    status public.payment_status NOT NULL,
    amount numeric DEFAULT 0 NOT NULL,
    raw_data json,
    scheme text NOT NULL,
    asset text NOT NULL,
    source_account_id character varying,
    destination_account_id character varying,
    initial_amount numeric DEFAULT 0 NOT NULL,
    CONSTRAINT payment_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE payments.transfers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    connector_id character varying NOT NULL,
    payment_id character varying,
    reference text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    amount numeric DEFAULT 0 NOT NULL,
    currency text NOT NULL,
    source text NOT NULL,
    destination text NOT NULL,
    status public.transfer_status DEFAULT 'PENDING'::public.transfer_status NOT NULL,
    error text,
    CONSTRAINT transfers_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE public.goose_db_version (
    id integer NOT NULL,
    version_id bigint NOT NULL,
    is_applied boolean NOT NULL,
    tstamp timestamp without time zone DEFAULT now()
);

CREATE SEQUENCE public.goose_db_version_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE tasks.task (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    connector_id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    name text NOT NULL,
    descriptor json,
    status public.task_status NOT NULL,
    error text,
    state json,
    scheduler_options json,
    CONSTRAINT task_check CHECK ((created_at <= updated_at)),
    CONSTRAINT task_created_at_check CHECK ((created_at <= now()))
);

CREATE TABLE transfers.transfer_initiation (
    id character varying NOT NULL,
    created_at timestamp with time zone NOT NULL,
    description text,
    type integer NOT NULL,
    source_account_id character varying,
    destination_account_id character varying NOT NULL,
    provider public.connector_provider NOT NULL,
    amount numeric NOT NULL,
    asset text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    scheduled_at timestamp with time zone,
    connector_id character varying NOT NULL,
    metadata jsonb,
    initial_amount numeric DEFAULT 0 NOT NULL,
    CONSTRAINT amount_non_negative CHECK ((amount >= (0)::numeric)),
    CONSTRAINT initial_amount_non_negative CHECK ((initial_amount >= (0)::numeric))
);

CREATE TABLE transfers.transfer_initiation_adjustments (
    id uuid NOT NULL,
    transfer_initiation_id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    status integer NOT NULL,
    error text,
    metadata jsonb
);

CREATE TABLE transfers.transfer_initiation_payments (
    transfer_initiation_id character varying NOT NULL,
    payment_id character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    status integer NOT NULL,
    error text
);

CREATE TABLE transfers.transfer_reversal (
    id character varying NOT NULL,
    transfer_initiation_id character varying NOT NULL,
    reference text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    description text NOT NULL,
    connector_id character varying NOT NULL,
    amount numeric NOT NULL,
    asset text NOT NULL,
    status integer NOT NULL,
    error text,
    metadata jsonb
);

ALTER TABLE ONLY accounts.account
    ADD CONSTRAINT account_pk PRIMARY KEY (id);

ALTER TABLE ONLY accounts.balances
    ADD CONSTRAINT balances_pkey PRIMARY KEY (account_id, created_at, currency);

ALTER TABLE ONLY accounts.bank_account
    ADD CONSTRAINT bank_account_pk PRIMARY KEY (id);

ALTER TABLE ONLY accounts.pool_accounts
    ADD CONSTRAINT pool_accounts_pk PRIMARY KEY (pool_id, account_id);

ALTER TABLE ONLY accounts.pools
    ADD CONSTRAINT pools_name_key UNIQUE (name);

ALTER TABLE ONLY accounts.pools
    ADD CONSTRAINT pools_pk PRIMARY KEY (id);

ALTER TABLE ONLY accounts.bank_account_related_accounts
    ADD CONSTRAINT transfer_initiation_adjustments_pk PRIMARY KEY (id);

ALTER TABLE ONLY connectors.connector
    ADD CONSTRAINT connector_v2_name_key UNIQUE (name);

ALTER TABLE ONLY connectors.connector
    ADD CONSTRAINT connector_v2_pk PRIMARY KEY (id);

ALTER TABLE ONLY connectors.webhook
    ADD CONSTRAINT webhook_pk PRIMARY KEY (id);

ALTER TABLE ONLY payments.adjustment
    ADD CONSTRAINT adjustment_pk PRIMARY KEY (id);

ALTER TABLE ONLY payments.adjustment
    ADD CONSTRAINT adjustment_reference_key UNIQUE (reference);

ALTER TABLE ONLY payments.metadata
    ADD CONSTRAINT metadata_pk PRIMARY KEY (payment_id, key);

ALTER TABLE ONLY payments.payment
    ADD CONSTRAINT payment_pk PRIMARY KEY (id);

ALTER TABLE ONLY payments.payment
    ADD CONSTRAINT payment_reference_key UNIQUE (reference);

ALTER TABLE ONLY payments.transfers
    ADD CONSTRAINT transfer_pk PRIMARY KEY (id);

ALTER TABLE ONLY payments.transfers
    ADD CONSTRAINT transfers_reference_key UNIQUE (reference);

ALTER TABLE ONLY public.goose_db_version
    ADD CONSTRAINT goose_db_version_pkey PRIMARY KEY (id);

ALTER TABLE ONLY tasks.task
    ADD CONSTRAINT task_pk PRIMARY KEY (id);

ALTER TABLE ONLY transfers.transfer_initiation_adjustments
    ADD CONSTRAINT transfer_initiation_adjustments_pk PRIMARY KEY (id);

ALTER TABLE ONLY transfers.transfer_initiation_payments
    ADD CONSTRAINT transfer_initiation_payments_pkey PRIMARY KEY (transfer_initiation_id, payment_id);

ALTER TABLE ONLY transfers.transfer_initiation
    ADD CONSTRAINT transfer_initiation_pkey PRIMARY KEY (id);

ALTER TABLE ONLY transfers.transfer_reversal
    ADD CONSTRAINT transfer_reversal_pkey PRIMARY KEY (id);

CREATE INDEX idx_created_at_account_id_currency ON accounts.balances USING btree (account_id, last_updated_at DESC, currency);

CREATE INDEX task_connector_id_descriptor ON tasks.task USING btree (connector_id, ((descriptor)::text));

CREATE UNIQUE INDEX transfer_reversal_processing_unique_constraint ON transfers.transfer_reversal USING btree (transfer_initiation_id) WHERE (status = 1);

ALTER TABLE ONLY accounts.account
    ADD CONSTRAINT accounts_connector FOREIGN KEY (connector_id) REFERENCES connectors.connector(id) ON DELETE CASCADE;

ALTER TABLE ONLY accounts.balances
    ADD CONSTRAINT balances_account FOREIGN KEY (account_id) REFERENCES accounts.account(id) ON DELETE CASCADE;

ALTER TABLE ONLY accounts.bank_account_related_accounts
    ADD CONSTRAINT bank_account_adjustments_account_id FOREIGN KEY (account_id) REFERENCES accounts.account(id) ON DELETE CASCADE;

ALTER TABLE ONLY accounts.bank_account_related_accounts
    ADD CONSTRAINT bank_account_adjustments_bank_account_id FOREIGN KEY (bank_account_id) REFERENCES accounts.bank_account(id) ON DELETE CASCADE;

ALTER TABLE ONLY accounts.bank_account_related_accounts
    ADD CONSTRAINT bank_account_adjustments_connector_id FOREIGN KEY (connector_id) REFERENCES connectors.connector(id) ON DELETE CASCADE;

ALTER TABLE ONLY accounts.pool_accounts
    ADD CONSTRAINT pool_accounts_account_id FOREIGN KEY (account_id) REFERENCES accounts.account(id) ON DELETE CASCADE;

ALTER TABLE ONLY accounts.pool_accounts
    ADD CONSTRAINT pool_accounts_pool_id FOREIGN KEY (pool_id) REFERENCES accounts.pools(id) ON DELETE CASCADE;

ALTER TABLE ONLY connectors.webhook
    ADD CONSTRAINT webhook_connector_id FOREIGN KEY (connector_id) REFERENCES connectors.connector(id) ON DELETE CASCADE;

ALTER TABLE ONLY payments.adjustment
    ADD CONSTRAINT adjustment_payment FOREIGN KEY (payment_id) REFERENCES payments.payment(id) ON DELETE CASCADE;

ALTER TABLE ONLY payments.metadata
    ADD CONSTRAINT metadata_payment FOREIGN KEY (payment_id) REFERENCES payments.payment(id) ON DELETE CASCADE;

ALTER TABLE ONLY payments.payment
    ADD CONSTRAINT payment_connector FOREIGN KEY (connector_id) REFERENCES connectors.connector(id) ON DELETE CASCADE;

ALTER TABLE ONLY payments.payment
    ADD CONSTRAINT payment_destination_account FOREIGN KEY (destination_account_id) REFERENCES accounts.account(id) ON DELETE CASCADE;

ALTER TABLE ONLY payments.payment
    ADD CONSTRAINT payment_source_account FOREIGN KEY (source_account_id) REFERENCES accounts.account(id) ON DELETE CASCADE;

ALTER TABLE ONLY tasks.task
    ADD CONSTRAINT task_connector FOREIGN KEY (connector_id) REFERENCES connectors.connector(id) ON DELETE CASCADE;

ALTER TABLE ONLY transfers.transfer_initiation_adjustments
    ADD CONSTRAINT adjusmtents_transfer_initiation_id_constraint FOREIGN KEY (transfer_initiation_id) REFERENCES transfers.transfer_initiation(id) ON DELETE CASCADE;

ALTER TABLE ONLY transfers.transfer_initiation
    ADD CONSTRAINT destination_account FOREIGN KEY (destination_account_id) REFERENCES accounts.account(id) ON DELETE CASCADE;

ALTER TABLE ONLY transfers.transfer_initiation
    ADD CONSTRAINT source_account FOREIGN KEY (source_account_id) REFERENCES accounts.account(id) ON DELETE CASCADE;

ALTER TABLE ONLY transfers.transfer_initiation
    ADD CONSTRAINT transfer_initiation_connector_id FOREIGN KEY (connector_id) REFERENCES connectors.connector(id) ON DELETE CASCADE;

ALTER TABLE ONLY transfers.transfer_initiation_payments
    ADD CONSTRAINT transfer_initiation_id_constraint FOREIGN KEY (transfer_initiation_id) REFERENCES transfers.transfer_initiation(id) ON DELETE CASCADE;

ALTER TABLE ONLY transfers.transfer_reversal
    ADD CONSTRAINT transfer_reversal_connector_id FOREIGN KEY (connector_id) REFERENCES connectors.connector(id) ON DELETE CASCADE;

ALTER TABLE ONLY transfers.transfer_reversal
    ADD CONSTRAINT transfer_reversal_transfer_initiation_id FOREIGN KEY (transfer_initiation_id) REFERENCES transfers.transfer_initiation(id) ON DELETE CASCADE;

--
-- Insert Test Data
--

INSERT INTO connectors.connector (id, name, created_at, provider, config)
VALUES
    ('eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9', '', '2025-01-01 10:24:02.854346+00', 'MONEYCORP', '\xc30d0409030238de3d15ff59937e73d27d01bfcd6fea9148015d71df4971cc89b8cf2964ad3685abed94fe574199b429bc00e0dbe5a8fa8da4eaa6b7ff2bd4d433ef004d207a3b36bd44a375af781430a7713702d03c9c7e284de83dfb7b2328d1a3c26341870729fbd66ce3746e575df81faa132d3df1325ba41b2999bd5e42d3299aa2ec2132577dbe42de0e38'),
    ('eyJQcm92aWRlciI6ImFkeWVuIiwiUmVmZXJlbmNlIjoiNGEwNzUyYWUtYWIxYS00NWI4LTgyNzItMjI2ODA3OTE2NTQ0In0', 'adyen', '2025-01-07 10:42:50.892706+00', 'ADYEN', '\xc30d040903029aab4850e29744c369d26b018438c4669398d2cd839e7f8e1d17bf1b5816dbfd4da65ad68fed8445134fb7213c460d393e958a935d96693326acb5140cb52507621289051ea09fce72b7f653bc49492e2e1ad545702ebb61c30591cd5b3be025beee3d537af0e3cd62b2637ebc20973d0b1d48a6dfb2'),
    ('eyJQcm92aWRlciI6ImF0bGFyIiwiUmVmZXJlbmNlIjoiN2JkZTk4NGUtYzY1OC00MzNiLWE1OGEtZTUxMGMwMTYwMDYwIn0', 'atlar', '2025-01-07 10:44:23.911698+00', 'ATLAR', '\xc30d04090302823792e07ed831bc63d2b101c85e65a22c6e980605f27ee7fa0b2831f465918176370823ef07e902c127fc542686a744be2bae5470e54b3f0ab4aed18a843606b4a9837bb2dda287f5dd6249d6a26eea05c049cb5e358acbf84e59a4d7ddee776f2209225f003facb4b7ec8bd64913b5f0141c15d04096a6be4a436d1bfa3b15ea36a9976a6f095a6a0546eb595b61bbfc35486f45fc89f450876d28a419970dd962ce605449ca97f58050ab1dd488a449cdd20157dfe242307c2f7e'),
    ('eyJQcm92aWRlciI6ImJhbmtpbmdjaXJjbGUiLCJSZWZlcmVuY2UiOiIwODQ4OWFlNC0zOGUxLTQwMTEtYjViMS1mZjkxMTliYWEzNDkifQ', 'bc', '2025-01-07 10:45:55.940438+00', 'BANKING-CIRCLE', '\xc30d040903025984229990f8420879d2a3012a39f1722b72be9fc0cf16378d9e65a49bd215a31562361a78a58d8d6f1f5f388359ddce3da098a622bc29e1015334a57af709dc949c52951508022cdbf25863bac41918ceb5d5b9766b8e08f56e012cafad41346b7973f72dd9baf091f4f37b6c37fcdf3925f473a3ca48aa088f1012b6d7afd15b31c5cd03a7b32ac47c0f371aa70bc3b50830936dac2e0188c391385ca3171457111a631d44236feedf7dcbf1a8'),
    ('eyJQcm92aWRlciI6ImN1cnJlbmN5Y2xvdWQiLCJSZWZlcmVuY2UiOiJlNmI4OGFlZS05OTI0LTQ4ZmYtYTZkMS1mYmIwZjJjMjRkYWYifQ', 'cc', '2025-01-07 10:47:29.452547+00', 'CURRENCY-CLOUD', '\xc30d04090302288feb29565456a76ad27801be28a7f882a9098a30f6653720d1ff0ff27fd5ed3407c726aeca7e986d01bc232926f61de88323dd7ae37332dfa37cc3b75c55b203f932d4e060e93b197ea5cb482840b34088d6ee84fc6c46d603693f72030e3c7c8c84c4422233faa4595c1a85fd8e80014bde42756cb3a26cbe6f70cc9674c2b8862c'),
    ('eyJQcm92aWRlciI6ImdlbmVyaWMiLCJSZWZlcmVuY2UiOiIwYmE0MDNiYi0zYzlmLTQ2OTUtYmQxZC0yYmQ5ZDdiMjgwOTQifQ', 'generic', '2025-01-07 10:48:45.320781+00', 'GENERIC', '\xc30d04090302701086c4c945bee56bd274019c8be59bddd596f5afda46c9c9d093fc56f12c8deb52e2a42c5fa5cf6ed9004e446f591e34e46533cdd05e2d1e48cc17beccd4ab8297255a17a6ec3f8c772a30edc8d618fa323676477559e2f043dae273c8e6aaf863cdfcf3e7de9b42f405fbad1adedc28d198cbdfbadc462c3664ae95fc33'),
    ('eyJQcm92aWRlciI6Im1hbmdvcGF5IiwiUmVmZXJlbmNlIjoiZTQ0MGIyMzgtM2RkNi00YzhlLTk5MDktZTJjOTgzODA2MTgyIn0', 'mango', '2025-01-07 10:49:41.421539+00', 'MANGOPAY', '\xc30d040903025af64e08c05c8c9c63d27c017540e62f74adba184ffa54608ce01d135e282aaebc222b544de675539e68087d8e677fd30e868f86778bc12f6d03e49b8bd95c3595aee950a49f48e7c2beb99e441df75673a83b6797dce56cff115380bbb456b0f7cdca3567dfb9c09fea0a57033c1b68d4841bc2238b3f7bb8d2ccae99eed3bb1d9f83fbb0cd94'),
    ('eyJQcm92aWRlciI6Im1vZHVsciIsIlJlZmVyZW5jZSI6IjYzZTZlNDIyLWQ5MWMtNDQ3YS1hODU0LTE5ODJkYTU1YzljYyJ9', 'modulr', '2025-01-07 10:50:29.225116+00', 'MODULR', '\xc30d04090302a026d21655bf707d6ad26f0115803d620e46b60da9541b45dcad524cf12a096d06f03a43ff9a950f01d69d246853e2a198ded9991561732f46e81ef642b00a7028acaa9a066e613be0a52f47e75de6972a2548b4e7df6cf76077267441a6a63f2b60f3dc3f33954e4410246323791908799460744da933be55d8'),
    ('eyJQcm92aWRlciI6InN0cmlwZSIsIlJlZmVyZW5jZSI6ImIwYzZjNTdhLTM3MDYtNDRmMi1iMDdmLTE3YjNiYTdhZDhkYyJ9', 'stripe', '2025-01-07 10:51:20.188637+00', 'STRIPE', '\xc30d040903027c5dfdd5ccfcc1a077d272014178b943cb16a1718d8cdd6fb02b874bf0e699254b1d46e4b51154c138f5eeae6a6d1698d6f6e6acc4ec7cfdc781ba3aeaab6592b2efb43be3d98557d691c2e28ba17896564af2a38297b05bfa1b556c095ca67a5e131c65134c8bfbbfef0fb83af94d0d3de7419551c2f94c5322736263'),
    ('eyJQcm92aWRlciI6Indpc2UiLCJSZWZlcmVuY2UiOiI4OWJlZDg1MS1kMjIyLTQ2NzItYjEwYy00ZDczZWE2ZGY0NGEifQ', 'wise', '2025-01-07 10:52:36.109388+00', 'WISE', '\xc30d04090302419ae943a4432cfb7dd264019566d167d65b3a9100f46437664ce10789cdc3f50c7f371471c90ed23eee084bcc02dc9af40a12b9f6d114d8535507ce1491dd7145db43d195aec206f1811e2e24f81a9bb22e4d5474e60cd99f2ad0ddebc3a86a71a6a56e3c5ab67509c3a5c63edd99'),
    ('eyJQcm92aWRlciI6IkRVTU1ZLVBBWSIsIlJlZmVyZW5jZSI6ImIxZTQ0NDFkLTcxNDYtNDk5MC04NDlhLTE3YjkzMTQxMzNhNCJ9', 'dummy', '2025-01-07 10:52:38.109388+00', 'DUMMY-PAY', '\xc30d04090302419ae943a4432cfb7dd264019566d167d65b3a9100f46437664ce10789cdc3f50c7f371471c90ed23eee084bcc02dc9af40a12b9f6d114d8535507ce1491dd7145db43d195aec206f1811e2e24f81a9bb22e4d5474e60cd99f2ad0ddebc3a86a71a6a56e3c5ab67509c3a5c63edd99')
;

INSERT INTO accounts.account (id, created_at, reference, type, raw_data, default_currency, account_name, connector_id, metadata)
VALUES
    ('eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', '2025-01-06 10:24:02.854346+00', 'test', 'INTERNAL', '{}', 'USD/2', 'test', 'eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9', '{}')
;

INSERT INTO payments.payment (id, connector_id, created_at, reference, type, status, amount, raw_data, scheme, asset, source_account_id, destination_account_id, initial_amount)
VALUES
    ('eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0IiwiVHlwZSI6IlBBWS1JTiJ9', 'eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9', '2025-01-07 10:24:02.854346+00', 'test', 'PAY-IN', 'SUCCEEDED', 100, '{}', 'sepa credit', 'USD/2', null, 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', 100)
;

INSERT INTO payments.adjustment (id, payment_id, created_at, amount, reference, status, raw_data)
VALUES
    ('83064af3-bb81-4514-a6d4-afba340825ca', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0IiwiVHlwZSI6IlBBWS1JTiJ9', '2025-01-07 10:24:02.854346+00', 100, 'test', 'SUCCEEDED', '{}'),
    ('83064af3-bb81-4514-a6d4-afba340825cb', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0IiwiVHlwZSI6IlBBWS1JTiJ9', '2025-01-07 11:25:02.854346+00', 200, 'test2', 'FAILED', '{}')
;

INSERT INTO accounts.balances (created_at, account_id, currency, balance, last_updated_at)
VALUES
    ('2025-01-06 10:30:02.854346+00', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', 'USD/2', 1000, '2025-01-06 11:30:02.854346+00')
;

INSERT INTO accounts.bank_account (id, created_at, name, account_number, iban, swift_bic_code, country, metadata)
VALUES
    ('83064af3-bb81-4514-a6d4-afba340825cd', '2025-01-06 14:30:02.854346+00', 'test', pgp_sym_encrypt('123456789', 'default-encryption-key', 'compress-algo=1, cipher-algo=aes256'), pgp_sym_encrypt('', 'default-encryption-key', 'compress-algo=1, cipher-algo=aes256'), pgp_sym_encrypt('', 'default-encryption-key', 'compress-algo=1, cipher-algo=aes256'), 'GB', '{}')
;

INSERT INTO accounts.bank_account_related_accounts (id, created_at, bank_account_id, connector_id, account_id)
VALUES
    ('9dbc5e96-a48e-45d6-82d1-bc65cd061411', '2025-01-06 17:30:02.854346+00', '83064af3-bb81-4514-a6d4-afba340825cd', 'eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0')
;

INSERT INTO transfers.transfer_initiation (id, created_at, description, type, source_account_id, destination_account_id, provider, amount, asset, attempts, scheduled_at, connector_id, metadata, initial_amount)
VALUES
    ('eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', '2025-01-06 13:30:02.854346+00', 'test', 1, 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', 'MONEYCORP', 50, 'USD/2', 1, '2025-01-06 13:30:02.854346+00', 'eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9', '{}', 50)
;

INSERT INTO transfers.transfer_initiation_adjustments (id, transfer_initiation_id, created_at, status, error, metadata)
VALUES
    ('9dbc5e96-a48e-45d6-82d1-bc65cd061410', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', '2025-01-06 13:30:02.854346+00', 0, '', null),
    ('9dbc5e96-a48e-45d6-82d1-bc65cd061411', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', '2025-01-06 13:31:02.854346+00', 6, '', null)
;

INSERT INTO transfers.transfer_initiation_payments (transfer_initiation_id, payment_id, created_at, status, error)
VALUES
    ('eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0IiwiVHlwZSI6IlBBWS1JTiJ9', '2025-01-06 19:31:02.854346+00', 0, '')
;

INSERT INTO accounts.pools (id, name, created_at)
VALUES
    ('83064af3-bb81-4514-a6d4-afba340825ce', 'test', '2025-01-06 14:30:02.854346+00')
;

INSERT INTO accounts.pool_accounts (pool_id, account_id)
VALUES
    ('83064af3-bb81-4514-a6d4-afba340825ce', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0')
;

INSERT INTO transfers.transfer_reversal (id, transfer_initiation_id, reference, created_at, updated_at, description, connector_id, amount, asset, status, error, metadata)
VALUES
    ('eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0X3JldmVyc2FsIn0', 'eyJDb25uZWN0b3JJRCI6eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9LCJSZWZlcmVuY2UiOiJ0ZXN0In0', 'test_reversal', '2025-01-06 14:40:02.854346+00', '2025-01-06 14:40:02.854346+00', 'test_reversal', 'eyJQcm92aWRlciI6Im1vbmV5Y29ycCIsIlJlZmVyZW5jZSI6IjdkNGU1MjM3LTNjMDktNDUwZS04ODY5LTI2YzA2MGFmMjM3NyJ9', 50, 'USD/2', 1, '', '{}')
;