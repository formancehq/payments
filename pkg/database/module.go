package database

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.uber.org/fx"
	"time"
)

func MongoModule(uri string, dbName string) fx.Option {
	return fx.Options(
		fx.Supply(options.Client().ApplyURI(uri)),
		fx.Provide(func(opts *options.ClientOptions) (*mongo.Client, error) {
			return mongo.NewClient(opts)
		}),
		fx.Provide(func(client *mongo.Client) *mongo.Database {
			return client.Database(dbName)
		}),
		fx.Invoke(func(lc fx.Lifecycle, client *mongo.Client) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					err := client.Connect(context.Background())
					if err != nil {
						return err
					}
					sharedlogging.Debug("Ping database...")
					ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
					defer cancel()

					err = client.Ping(ctx, readpref.Primary())
					if err != nil {
						return err
					}
					return nil
				},
			})
		}),
	)
}

func MongoMonitor() fx.Option {
	return fx.Decorate(func(opts *options.ClientOptions) *options.ClientOptions {
		opts.SetMonitor(otelmongo.NewMonitor())
		return opts
	})
}
