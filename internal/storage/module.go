package storage

import (
	"context"

	"github.com/formancehq/go-libs/v2/bun/bunconnect"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

func Module(cmd *cobra.Command, connectionOptions bunconnect.ConnectionOptions, configEncryptionKey string) fx.Option {
	return fx.Options(
		bunconnect.Module(connectionOptions, service.IsDebug(cmd)),
		fx.Provide(func(logger logging.Logger, db *bun.DB) Storage {
			return newStorage(logger, db, configEncryptionKey)
		}),
		fx.Invoke(func(s Storage, lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					return s.Close()
				},
			})
		}),
	)
}
