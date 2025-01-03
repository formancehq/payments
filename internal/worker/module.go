package worker

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/formancehq/go-libs/v2/httpserver"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/temporal"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
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
	debug bool,
) fx.Option {
	ret := []fx.Option{
		fx.Supply(worker.Options{
			MaxConcurrentWorkflowTaskPollers: temporalMaxConcurrentWorkflowTaskPollers,
		}),
		fx.Provide(func(publisher message.Publisher) *events.Events {
			return events.New(publisher, stackURL)
		}),
		fx.Provide(func(logger logging.Logger) plugins.Plugins {
			return plugins.New(logger, debug)
		}),
		fx.Provide(func(temporalClient client.Client, plugins plugins.Plugins) workflow.Workflow {
			return workflow.New(temporalClient, temporalNamespace, plugins, stack, stackURL)
		}),
		fx.Provide(func(temporalClient client.Client, storage storage.Storage, events *events.Events, plugins plugins.Plugins) activities.Activities {
			return activities.New(temporalClient, storage, events, plugins, temporalRateLimitingRetryDelay)
		}),
		fx.Provide(
			fx.Annotate(func(
				logger logging.Logger,
				temporalClient client.Client,
				workflows,
				activities []temporal.DefinitionSet,
				storage storage.Storage,
				plugins plugins.Plugins,
				options worker.Options,
			) *engine.WorkerPool {
				return engine.NewWorkerPool(logger, stack, temporalClient, workflows, activities, storage, plugins, options)
			}, fx.ParamTags(``, ``, `group:"workflows"`, `group:"activities"`, ``)),
		),
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
