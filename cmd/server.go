package cmd

import (
	"github.com/formancehq/go-libs/v2/auth"
	"github.com/formancehq/go-libs/v2/aws/iam"
	"github.com/formancehq/go-libs/v2/bun/bunconnect"
	"github.com/formancehq/go-libs/v2/licence"
	"github.com/formancehq/go-libs/v2/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/v2/otlp/otlptraces"
	"github.com/formancehq/go-libs/v2/profiling"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/go-libs/v2/temporal"
	"github.com/spf13/cobra"
)

func newServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Aliases:      []string{"server"},
		Short:        "Launch api server",
		SilenceUsage: true,
		RunE:         runServer(),
	}

	service.AddFlags(cmd.Flags())
	otlpmetrics.AddFlags(cmd.Flags())
	otlptraces.AddFlags(cmd.Flags())
	auth.AddFlags(cmd.Flags())
	publish.AddFlags(ServiceName, cmd.Flags())
	bunconnect.AddFlags(cmd.Flags())
	iam.AddFlags(cmd.Flags())
	profiling.AddFlags(cmd.Flags())
	temporal.AddFlags(cmd.Flags())
	licence.AddFlags(cmd.Flags())

	return cmd
}

func runServer() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		setLogger()

		options, err := commonOptions(cmd)
		if err != nil {
			return err
		}

		return service.New(cmd.OutOrStdout(), options).Run(cmd)
	}
}
