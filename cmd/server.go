package cmd

import (
	"github.com/formancehq/go-libs/v2/service"
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
	commonFlags(cmd)
	cmd.Flags().String(ListenFlag, ":8080", "Listen address")
	cmd.Flags().String(stackPublicURLFlag, "", "Stack public url")
	// MaxConcurrentWorkflowTaskPollers should not be set to a number < 2, otherwise
	// temporal will panic.
	// After meeting with the temporal team, we decided to set it to 20 as per
	// their recommendation.
	cmd.Flags().Int(temporalMaxConcurrentWorkflowTaskPollersFlag, 20, "Max concurrent workflow task pollers")
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
