package testserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/connect"
	"github.com/formancehq/go-libs/v5/pkg/observe"
	"github.com/formancehq/go-libs/v5/pkg/observe/metrics"
	"github.com/formancehq/go-libs/v5/pkg/observe/profiling"
	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	"github.com/formancehq/go-libs/v5/pkg/service"
	"github.com/formancehq/go-libs/v5/pkg/workflow/temporal"
	"github.com/formancehq/payments/cmd"
)

func Flags(command string, serverID string, configuration Configuration) []string {
	if configuration.OutboxPollingInterval < time.Second {
		configuration.OutboxPollingInterval = time.Second
	}

	args := []string{
		command,
		"--" + cmd.ListenFlag, ":0",
		"--" + connect.PostgresURIFlag, configuration.PostgresConfiguration.DatabaseSourceName,
		"--" + cmd.ConfigEncryptionKeyFlag, "dummyval",
		"--" + temporal.TemporalAddressFlag, configuration.TemporalAddress,
		"--" + temporal.TemporalNamespaceFlag, configuration.TemporalNamespace,
		"--" + temporal.TemporalInitSearchAttributesFlag, fmt.Sprintf("stack=%s", configuration.Stack),
		"--" + cmd.StackFlag, configuration.Stack,
		"--" + profiling.ProfilerEnableFlag + "=false",
	}

	if command == "worker" {
		args = append(args, "--"+cmd.OutboxPollingIntervalFlag+fmt.Sprintf("=%s", configuration.OutboxPollingInterval))
		if configuration.SkipOutboxScheduleCreation {
			args = append(args, "--"+cmd.SkipOutboxScheduleCreationFlag+"=true")
		}
	}

	if configuration.PostgresConfiguration.MaxIdleConns != 0 {
		args = append(
			args,
			"--"+connect.PostgresMaxIdleConnsFlag,
			fmt.Sprint(configuration.PostgresConfiguration.MaxIdleConns),
		)
	}
	if configuration.PostgresConfiguration.MaxOpenConns != 0 {
		args = append(
			args,
			"--"+connect.PostgresMaxOpenConnsFlag,
			fmt.Sprint(configuration.PostgresConfiguration.MaxOpenConns),
		)
	}
	if configuration.PostgresConfiguration.ConnMaxIdleTime != 0 {
		args = append(
			args,
			"--"+connect.PostgresConnMaxIdleTimeFlag,
			fmt.Sprint(configuration.PostgresConfiguration.ConnMaxIdleTime),
		)
	}
	if configuration.NatsURL != "" {
		args = append(
			args,
			"--"+publish.PublisherNatsEnabledFlag,
			"--"+publish.PublisherNatsURLFlag, configuration.NatsURL,
			"--"+publish.PublisherTopicMappingFlag, fmt.Sprintf("*:%s", serverID),
		)
	}
	if configuration.OTLPConfig != nil {
		if configuration.OTLPConfig.Metrics != nil {
			args = append(
				args,
				"--"+metrics.OtelMetricsExporterFlag, configuration.OTLPConfig.Metrics.Exporter,
			)
			if configuration.OTLPConfig.Metrics.KeepInMemory {
				args = append(
					args,
					"--"+metrics.OtelMetricsKeepInMemoryFlag,
				)
			}
			if configuration.OTLPConfig.Metrics.OTLPConfig != nil {
				args = append(
					args,
					"--"+metrics.OtelMetricsExporterOTLPEndpointFlag, configuration.OTLPConfig.Metrics.OTLPConfig.Endpoint,
					"--"+metrics.OtelMetricsExporterOTLPModeFlag, configuration.OTLPConfig.Metrics.OTLPConfig.Mode,
				)
				if configuration.OTLPConfig.Metrics.OTLPConfig.Insecure {
					args = append(args, "--"+metrics.OtelMetricsExporterOTLPInsecureFlag)
				}
			}
			if configuration.OTLPConfig.Metrics.RuntimeMetrics {
				args = append(args, "--"+metrics.OtelMetricsRuntimeFlag)
			}
			if configuration.OTLPConfig.Metrics.MinimumReadMemStatsInterval != 0 {
				args = append(
					args,
					"--"+metrics.OtelMetricsRuntimeMinimumReadMemStatsIntervalFlag,
					configuration.OTLPConfig.Metrics.MinimumReadMemStatsInterval.String(),
				)
			}
			if configuration.OTLPConfig.Metrics.PushInterval != 0 {
				args = append(
					args,
					"--"+metrics.OtelMetricsExporterPushIntervalFlag,
					configuration.OTLPConfig.Metrics.PushInterval.String(),
				)
			}
			if len(configuration.OTLPConfig.Metrics.ResourceAttributes) > 0 {
				args = append(
					args,
					"--"+observe.OtelResourceAttributesFlag,
					strings.Join(configuration.OTLPConfig.Metrics.ResourceAttributes, ","),
				)
			}
		}
		if configuration.OTLPConfig.BaseConfig.ServiceName != "" {
			args = append(args, "--"+observe.OtelServiceNameFlag, configuration.OTLPConfig.BaseConfig.ServiceName)
		}
	}

	if configuration.Debug {
		args = append(args, "--"+service.DebugFlag)
	}
	return args
}
