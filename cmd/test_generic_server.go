package cmd

import (
	"github.com/formancehq/go-libs/v2/health"
	"github.com/formancehq/go-libs/v2/service"
	testserver "github.com/formancehq/payments/internal/connectors/plugins/public/generic/client/test_server"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const (
	nbAccountsFlag = "nb-accounts"
)

func newTestGenericServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "test_generic_server",
		Aliases:      []string{"tgs"},
		Short:        "Launch test generic server",
		SilenceUsage: true,
		RunE:         runTestGenericServer(),
	}

	service.AddFlags(cmd.Flags())

	cmd.Flags().Int(nbAccountsFlag, 500, "number of accounts to create")

	return cmd
}

func runTestGenericServer() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		nbAccounts, _ := cmd.Flags().GetInt(nbAccountsFlag)

		options := fx.Options(
			testserver.NewModule(nbAccounts, ":8080"),
			health.Module(),
		)

		return service.New(cmd.OutOrStdout(), options).Run(cmd)
	}
}
