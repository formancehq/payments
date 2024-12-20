package testserver

import (
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/v2/bun/bunconnect"
	"github.com/formancehq/go-libs/v2/otlp"
	"github.com/formancehq/go-libs/v2/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/v2/profiling"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/go-libs/v2/temporal"
	"github.com/formancehq/payments/cmd"
)

func Flags(command string, serverID string, configuration Configuration) []string {
	args := []string{
		command,
		"--" + cmd.ListenFlag, ":0",
		"--" + bunconnect.PostgresURIFlag, configuration.PostgresConfiguration.DatabaseSourceName,
		"--" + bunconnect.PostgresMaxOpenConnsFlag, fmt.Sprint(configuration.PostgresConfiguration.MaxOpenConns),
		"--" + bunconnect.PostgresConnMaxIdleTimeFlag, fmt.Sprint(configuration.PostgresConfiguration.ConnMaxIdleTime),
		"--" + cmd.ConfigEncryptionKeyFlag, "dummyval",
		"--" + temporal.TemporalAddressFlag, configuration.TemporalAddress,
		"--" + temporal.TemporalNamespaceFlag, configuration.TemporalNamespace,
		"--" + temporal.TemporalInitSearchAttributesFlag, fmt.Sprintf("stack=%s", configuration.Stack),
		"--" + cmd.StackFlag, configuration.Stack,
		"--" + profiling.ProfilerEnableFlag, "false",
	}
	if configuration.PostgresConfiguration.MaxIdleConns != 0 {
		args = append(
			args,
			"--"+bunconnect.PostgresMaxIdleConnsFlag,
			fmt.Sprint(configuration.PostgresConfiguration.MaxIdleConns),
		)
	}
	if configuration.PostgresConfiguration.MaxOpenConns != 0 {
		args = append(
			args,
			"--"+bunconnect.PostgresMaxOpenConnsFlag,
			fmt.Sprint(configuration.PostgresConfiguration.MaxOpenConns),
		)
	}
	if configuration.PostgresConfiguration.ConnMaxIdleTime != 0 {
		args = append(
			args,
			"--"+bunconnect.PostgresConnMaxIdleTimeFlag,
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
				"--"+otlpmetrics.OtelMetricsExporterFlag, configuration.OTLPConfig.Metrics.Exporter,
			)
			if configuration.OTLPConfig.Metrics.KeepInMemory {
				args = append(
					args,
					"--"+otlpmetrics.OtelMetricsKeepInMemoryFlag,
				)
			}
			if configuration.OTLPConfig.Metrics.OTLPConfig != nil {
				args = append(
					args,
					"--"+otlpmetrics.OtelMetricsExporterOTLPEndpointFlag, configuration.OTLPConfig.Metrics.OTLPConfig.Endpoint,
					"--"+otlpmetrics.OtelMetricsExporterOTLPModeFlag, configuration.OTLPConfig.Metrics.OTLPConfig.Mode,
				)
				if configuration.OTLPConfig.Metrics.OTLPConfig.Insecure {
					args = append(args, "--"+otlpmetrics.OtelMetricsExporterOTLPInsecureFlag)
				}
			}
			if configuration.OTLPConfig.Metrics.RuntimeMetrics {
				args = append(args, "--"+otlpmetrics.OtelMetricsRuntimeFlag)
			}
			if configuration.OTLPConfig.Metrics.MinimumReadMemStatsInterval != 0 {
				args = append(
					args,
					"--"+otlpmetrics.OtelMetricsRuntimeMinimumReadMemStatsIntervalFlag,
					configuration.OTLPConfig.Metrics.MinimumReadMemStatsInterval.String(),
				)
			}
			if configuration.OTLPConfig.Metrics.PushInterval != 0 {
				args = append(
					args,
					"--"+otlpmetrics.OtelMetricsExporterPushIntervalFlag,
					configuration.OTLPConfig.Metrics.PushInterval.String(),
				)
			}
			if len(configuration.OTLPConfig.Metrics.ResourceAttributes) > 0 {
				args = append(
					args,
					"--"+otlp.OtelResourceAttributesFlag,
					strings.Join(configuration.OTLPConfig.Metrics.ResourceAttributes, ","),
				)
			}
		}
		if configuration.OTLPConfig.BaseConfig.ServiceName != "" {
			args = append(args, "--"+otlp.OtelServiceNameFlag, configuration.OTLPConfig.BaseConfig.ServiceName)
		}
	}

	if configuration.Debug {
		args = append(args, "--"+service.DebugFlag)
	}
	return args
}
