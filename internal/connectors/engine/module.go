package engine

import (
	"context"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

func Module(stack string, debug bool) fx.Option {
	ret := []fx.Option{
		fx.Provide(func(
			logger logging.Logger,
			temporalClient client.Client,
			storage storage.Storage,
		) Engine {
			return New(logger, temporalClient, storage, stack)
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
