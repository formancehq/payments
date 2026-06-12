package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

// encryptOpenBankingTokens converts the open-banking token columns to bytea and
// encrypts any existing plaintext values in place, mirroring the pgcrypto
// mechanism used for connector configs and bank account details.
func encryptOpenBankingTokens(ctx context.Context, db bun.IDB, encryptionKey string) error {
	// access_token is NOT NULL, so every row holds a value to encrypt.
	_, err := db.NewRaw(`
		ALTER TABLE open_banking_access_tokens
			ALTER COLUMN access_token TYPE bytea
			USING pgp_sym_encrypt(access_token, ?, ?);
	`, encryptionKey, encryptionOptions).Exec(ctx)
	if err != nil {
		return err
	}

	// temporary_token is nullable, so leave NULLs untouched.
	_, err = db.NewRaw(`
		ALTER TABLE open_banking_connection_attempts
			ALTER COLUMN temporary_token TYPE bytea
			USING CASE
				WHEN temporary_token IS NULL THEN NULL
				ELSE pgp_sym_encrypt(temporary_token, ?, ?)
			END;
	`, encryptionKey, encryptionOptions).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
