package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/bombsimon/logrusr/v3"
	"github.com/formancehq/go-libs/v5/pkg/authn/jwt"
	"github.com/formancehq/go-libs/v5/pkg/authn/licence"
	"github.com/formancehq/go-libs/v5/pkg/cloud/aws/iam"
	"github.com/formancehq/go-libs/v5/pkg/fx/authnfx"
	"github.com/formancehq/go-libs/v5/pkg/fx/messagingfx"
	"github.com/formancehq/go-libs/v5/pkg/fx/observefx"
	"github.com/formancehq/go-libs/v5/pkg/fx/servicefx"
	"github.com/formancehq/go-libs/v5/pkg/fx/workflowfx"
	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	"github.com/formancehq/go-libs/v5/pkg/observe/metrics"
	"github.com/formancehq/go-libs/v5/pkg/observe/profiling"
	"github.com/formancehq/go-libs/v5/pkg/observe/traces"
	"github.com/formancehq/go-libs/v5/pkg/service"
	"github.com/formancehq/go-libs/v5/pkg/storage/bun/connect"
	sharedapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/go-libs/v5/pkg/workflow/temporal"
	"github.com/formancehq/payments/internal/connectors/engine"
	connectormetrics "github.com/formancehq/payments/pkg/domain/metrics"
	"github.com/formancehq/payments/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
)

func setLogger() {
	// Add a dedicated logger for opentelemetry in case of error
	otel.SetLogger(logrusr.New(logrus.New().WithField("component", "otlp")))
}

func commonFlags(cmd *cobra.Command) {
	cmd.Flags().String(StackFlag, "", "Stack name")
	cmd.Flags().String(ListenFlag, ":8080", "Listen address")
	cmd.Flags().Duration(ConnectorPollingPeriodDefault, 30*time.Minute, "Default polling period for connectors")
	cmd.Flags().Duration(ConnectorPollingPeriodMinimum, 20*time.Minute, "Minimum polling period for connectors")
	service.AddFlags(cmd.Flags())
	metrics.AddFlags(cmd.Flags())
	traces.AddFlags(cmd.Flags())
	jwt.AddFlags(cmd.Flags())
	publish.AddFlags(ServiceName, cmd.Flags())
	connect.AddFlags(cmd.Flags())
	iam.AddFlags(cmd.Flags())
	profiling.AddFlags(cmd.Flags())
	temporal.AddFlags(cmd.Flags())
	licence.AddFlags(cmd.Flags())
}

func commonOptions(cmd *cobra.Command) (fx.Option, error) {
	configEncryptionKey, _ := cmd.Flags().GetString(ConfigEncryptionKeyFlag)
	if configEncryptionKey == "" {
		return nil, errors.New("missing config encryption key")
	}

	connectionOptions, err := connect.ConnectionOptionsFromFlags(cmd.Flags(), cmd.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to get connection options: %w", err)
	}

	return fx.Options(
		fx.Provide(func() *connect.ConnectionOptions {
			return connectionOptions
		}),
		observefx.ResourceModuleFromFlags(cmd),
		observefx.TracesModuleFromFlags(cmd),
		observefx.MetricsModuleFromFlags(cmd),
		fx.Provide(connectormetrics.RegisterMetricsRegistry),
		fx.Invoke(func(connectormetrics.MetricsRegistry) {}),
		workflowfx.TemporalClientModuleFromFlags(
			cmd,
			engine.Tracer,
			temporal.SearchAttributes{
				SearchAttributes: engine.SearchAttributes,
			},
		),
		fx.Provide(func() sharedapi.ServiceInfo {
			return sharedapi.ServiceInfo{
				Version: Version,
			}
		}),
		servicefx.HealthModule(),
		messagingfx.PublishModuleFromFlags(cmd, service.IsDebug(cmd)),
		authnfx.LicenceModuleFromFlags(cmd, ServiceName),
		storage.Module(cmd, *connectionOptions, configEncryptionKey),
		observefx.ProfilingModuleFromFlags(cmd),
	), nil
}
