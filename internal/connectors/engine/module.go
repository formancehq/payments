package engine

import (
	"context"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

func Module(
	stack string,
	stackPublicURL string,
	debug bool,
	pollingPeriodDefault time.Duration,
	pollingPeriodMinimum time.Duration,
) fx.Option {
	ret := []fx.Option{
		fx.Provide(func(logger logging.Logger) connectors.Manager {
			return connectors.NewManager(logger, debug, pollingPeriodDefault, pollingPeriodMinimum)
		}),
		fx.Provide(func(
			logger logging.Logger,
			temporalClient client.Client,
			storage storage.Storage,
			manager connectors.Manager,
		) Engine {
			return New(logger, temporalClient, storage, manager, stack, stackPublicURL)
		}),
		fx.Invoke(func(lc fx.Lifecycle, engine Engine) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return engine.OnStart(ctx)
				},
				OnStop: func(ctx context.Context) error {
					engine.OnStop(ctx)
					return nil
				},
			})
		}),
	}

	return fx.Options(ret...)
}
