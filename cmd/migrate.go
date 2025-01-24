package cmd

import (
	"github.com/formancehq/go-libs/v2/bun/bunmigrate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/storage"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"

	// Import the postgres driver.
	_ "github.com/lib/pq"
)

var (
	autoMigrateFlag = "auto-migrate"
)

func newMigrate() *cobra.Command {
	cmd := bunmigrate.NewDefaultCommand(Migrate, func(cmd *cobra.Command) {
		cmd.Flags().String(ConfigEncryptionKeyFlag, "", "Config encryption key")
	})

	return cmd
}

func Migrate(cmd *cobra.Command, args []string, db *bun.DB) error {
	cfgEncryptionKey, _ := cmd.Flags().GetString(ConfigEncryptionKeyFlag)
	if cfgEncryptionKey == "" {
		cfgEncryptionKey = cmd.Flag(ConfigEncryptionKeyFlag).Value.String()
	}

	if cfgEncryptionKey != "" {
		storage.EncryptionKey = cfgEncryptionKey
	}

	logger := logging.NewDefaultLogger(cmd.OutOrStdout(), true, true, false)

	return storage.Migrate(cmd.Context(), logger, db, cfgEncryptionKey)
}
