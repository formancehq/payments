package worker

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/formancehq/go-libs/v3/httpserver"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/temporal"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
)

func NewHealthCheckModule(bind string, debug bool) fx.Option {
	return fx.Options(
		fx.Provide(fx.Annotate(NewRouter, fx.ResultTags(`name:"healthCheck"`))),
		fx.Invoke(fx.Annotate(func(m *chi.Mux, lc fx.Lifecycle) {
			lc.Append(httpserver.NewHook(m, httpserver.WithAddress(bind)))
		}, fx.ParamTags(`name:"healthCheck"`, ``))),
	)
}

func NewModule(
	stack string,
	stackURL string,
	temporalNamespace string,
	temporalRateLimitingRetryDelay time.Duration,
	temporalMaxConcurrentWorkflowTaskPollers int,
	temporalMaxConcurrentActivityTaskPollers int,
	temporalMaxSlotsPerPoller int,
	temporalMaxLocalActivitySlots int,
	debug bool,
	skipOutboxScheduleCreation bool,
	pollingPeriodDefault time.Duration,
	pollingPeriodMinimum time.Duration,
	outboxPollingInterval time.Duration,
	outboxCleanupInterval time.Duration,
) fx.Option {
	ret := []fx.Option{
		fx.Supply(worker.Options{
			MaxConcurrentWorkflowTaskPollers:        temporalMaxConcurrentWorkflowTaskPollers,
			MaxConcurrentWorkflowTaskExecutionSize:  temporalMaxConcurrentWorkflowTaskPollers * temporalMaxSlotsPerPoller,
			MaxConcurrentActivityTaskPollers:        temporalMaxConcurrentActivityTaskPollers,
			MaxConcurrentActivityExecutionSize:      temporalMaxConcurrentActivityTaskPollers * temporalMaxSlotsPerPoller,
			MaxConcurrentLocalActivityExecutionSize: temporalMaxLocalActivitySlots,
		}),
		fx.Provide(func(publisher message.Publisher) *events.Events {
			return events.New(publisher, stackURL)
		}),
		fx.Provide(func(logger logging.Logger) connectors.Manager {
			return connectors.NewManager(logger, debug, pollingPeriodDefault, pollingPeriodMinimum)
		}),
		fx.Provide(func(temporalClient client.Client, manager connectors.Manager, logger logging.Logger) workflow.Workflow {
			return workflow.New(temporalClient, temporalNamespace, manager, stack, stackURL, logger)
		}),
		fx.Provide(func(
			logger logging.Logger,
			temporalClient client.Client,
			storage storage.Storage,
			events *events.Events,
			connectors connectors.Manager,
		) activities.Activities {
			return activities.New(logger, temporalClient, storage, events, connectors, temporalRateLimitingRetryDelay)
		}),
		fx.Provide(
			fx.Annotate(func(
				logger logging.Logger,
				temporalClient client.Client,
				workflows,
				activities []temporal.DefinitionSet,
				storage storage.Storage,
				connectors connectors.Manager,
				options worker.Options,
			) *engine.WorkerPool {
				return engine.NewWorkerPool(
					logger,
					stack,
					temporalClient,
					workflows,
					activities,
					storage,
					connectors,
					options,
					outboxPollingInterval,
					outboxCleanupInterval,
				)
			}, fx.ParamTags(``, ``, `group:"workflows"`, `group:"activities"`, ``)),
		),
		fx.Invoke(func(workers *engine.WorkerPool) {
			workers.SetSkipScheduleCreation(skipOutboxScheduleCreation)
		}),
		fx.Invoke(func(lc fx.Lifecycle, workers *engine.WorkerPool) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return workers.OnStart(ctx)
				},
				OnStop: func(ctx context.Context) error {
					workers.Close()
					return nil
				},
			})
		}),
		fx.Provide(fx.Annotate(func(a activities.Activities) temporal.DefinitionSet {
			return a.DefinitionSet()
		}, fx.ResultTags(`group:"activities"`))),
		fx.Provide(fx.Annotate(func(workflow workflow.Workflow) temporal.DefinitionSet {
			return workflow.DefinitionSet()
		}, fx.ResultTags(`group:"workflows"`))),
	}

	return fx.Options(ret...)
}
