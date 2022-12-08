package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	up := func(tx *sql.Tx) error {
		_, err := tx.Exec(`
				CREATE TABLE payments.adjustment (
					id uuid  NOT NULL DEFAULT gen_random_uuid(),
					payment_id uuid  NOT NULL,
					created_at timestamp with time zone  NOT NULL DEFAULT NOW() CHECK (created_at<=NOW()),
					amount bigint NOT NULL DEFAULT 0 CHECK (amount>0),
					status payment_status  NOT NULL,
					absolute boolean  NOT NULL DEFAULT FALSE,
					raw_data json  NULL,
					CONSTRAINT adjustment_pk PRIMARY KEY (id)
				);

				CREATE TABLE payments.metadata_changelog (
					payment_id uuid  NOT NULL,
					created_at timestamp with time zone  NOT NULL DEFAULT NOW() CHECK (created_at<=NOW()),
					key text  NOT NULL,
					value_before text  NOT NULL,
					value_after text  NOT NULL,
					CONSTRAINT metadata_changelog_pk PRIMARY KEY (payment_id,key)
				);

				CREATE TABLE payments.metadata (
					payment_id uuid  NOT NULL,
					created_at timestamp with time zone  NOT NULL DEFAULT NOW() CHECK (created_at<=NOW()),
					key text  NOT NULL,
					value text  NOT NULL,
					CONSTRAINT metadata_pk PRIMARY KEY (payment_id,key)
				);

				CREATE TYPE payment_type AS ENUM ('PAY-IN', 'PAYOUT', 'TRANSFER', 'OTHER');
				CREATE TYPE payment_status AS ENUM ('SUCCEEDED', 'CANCELLED', 'FAILED', 'PENDING', 'OTHER');;

				CREATE TABLE payments.payment (
					id uuid  NOT NULL DEFAULT gen_random_uuid(),
					connector_id uuid  NOT NULL,
					account_id uuid  NOT NULL,
					created_at timestamp with time zone  NOT NULL DEFAULT NOW() CHECK (created_at<=NOW()),
					reference text  NOT NULL,
					type payment_type  NOT NULL,
					status payment_status  NOT NULL,
					amount bigint NOT NULL DEFAULT 0 CHECK (amount>0),
					raw_data json  NULL,
					scheme text  NOT NULL,
					asset text  NOT NULL,
					CONSTRAINT payment_pk PRIMARY KEY (id)
				);

				ALTER TABLE payments.adjustment ADD CONSTRAINT adjustment_payment
					FOREIGN KEY (payment_id)
					REFERENCES payments.payment (id)  
					NOT DEFERRABLE 
					INITIALLY IMMEDIATE
				;

				ALTER TABLE payments.metadata_changelog ADD CONSTRAINT metadata_changelog_metadata
					FOREIGN KEY (payment_id, key)
					REFERENCES payments.metadata (payment_id, key)  
					NOT DEFERRABLE 
					INITIALLY IMMEDIATE
				;

				ALTER TABLE payments.metadata ADD CONSTRAINT metadata_payment
					FOREIGN KEY (payment_id)
					REFERENCES payments.payment (id)  
					NOT DEFERRABLE 
					INITIALLY IMMEDIATE
				;

				ALTER TABLE payments.payment ADD CONSTRAINT payment_account
					FOREIGN KEY (account_id)
					REFERENCES accounts.account (id)  
					NOT DEFERRABLE 
					INITIALLY IMMEDIATE
				;

				ALTER TABLE payments.payment ADD CONSTRAINT payment_connector
					FOREIGN KEY (connector_id)
					REFERENCES connectors.connector (id)  
					NOT DEFERRABLE 
					INITIALLY IMMEDIATE
				;
		`)
		if err != nil {
			return err
		}

		return nil
	}

	down := func(tx *sql.Tx) error {
		_, err := tx.Exec(`
				DROP TABLE payments.adjustment;
				DROP TABLE payments.metadata_changelog;
				DROP TABLE payments.metadata;
				DROP TABLE payments.payment;
				DROP TYPE payment_type;
				DROP TYPE payment_status;
		`)
		if err != nil {
			return err
		}

		return nil
	}

	goose.AddMigration(up, down)
}
