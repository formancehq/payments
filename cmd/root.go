package cmd

import (
	_ "github.com/bombsimon/logrusr/v3"
	"github.com/formancehq/go-libs/v2/bun/bunmigrate"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/spf13/cobra"
)

var (
	ServiceName = "payments"
	Version     = "develop"
	BuildDate   = "-"
	Commit      = "-"
)

const (
	ConfigEncryptionKeyFlag                      = "config-encryption-key"
	ListenFlag                                   = "listen"
	StackFlag                                    = "stack"
	stackPublicURLFlag                           = "stack-public-url"
	temporalMaxConcurrentWorkflowTaskPollersFlag = "temporal-max-concurrent-workflow-task-pollers"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:               "payments",
		Short:             "payments",
		DisableAutoGenTag: true,
		Version:           Version,
	}

	root.PersistentFlags().String(ConfigEncryptionKeyFlag, "", "Config encryption key")

	version := newVersion()
	root.AddCommand(version)

	migrate := newMigrate()
	root.AddCommand(migrate)

	server := newServer()
	addAutoMigrateCommand(server)
	root.AddCommand(server)

	purge := newPurge()
	purge.Flags().String(StackFlag, "", "Stack name")
	root.AddCommand(purge)

	return root
}

func Execute() {
	service.Execute(NewRootCommand())
}

func addAutoMigrateCommand(cmd *cobra.Command) {
	cmd.Flags().Bool(autoMigrateFlag, false, "Auto migrate database")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		autoMigrate, _ := cmd.Flags().GetBool(autoMigrateFlag)
		if autoMigrate {
			return bunmigrate.Run(cmd, args, Migrate)
		}
		return nil
	}
}
