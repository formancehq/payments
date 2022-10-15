package database

import (
	"context"
	"reflect"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/fx"
)

func MongoModule(uri string, dbName string) fx.Option {
	return fx.Options(
		fx.Provide(func() *options.ClientOptions {
			tM := reflect.TypeOf(bson.M{})
			reg := bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, tM).Build()
			return options.Client().
				SetRegistry(reg).
				ApplyURI(uri)
		}),

		fx.Provide(func(opts *options.ClientOptions) (*mongo.Client, error) {
			return mongo.NewClient(opts)
		}),

		fx.Provide(func(client *mongo.Client) *mongo.Database {
			return client.Database(dbName)
		}),

		fx.Invoke(func(lc fx.Lifecycle, client *mongo.Client, db *mongo.Database) {
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

					err = createIndexes(ctx, db)
					if err != nil {
						return errors.Wrap(err, "creating indices")
					}
					return nil
				},
			})
		}),
	)
}