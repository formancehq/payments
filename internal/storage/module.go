package storage

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/fx/storagefx"
	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/go-libs/v5/pkg/service"
	"github.com/formancehq/go-libs/v5/pkg/storage/bun/connect"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

func Module(cmd *cobra.Command, connectionOptions connect.ConnectionOptions, configEncryptionKey string) fx.Option {
	return fx.Options(
		storagefx.BunConnectModule(connectionOptions, service.IsDebug(cmd)),
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
