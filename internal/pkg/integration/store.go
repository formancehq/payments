package integration

import (
	"context"
	"reflect"

	"github.com/numary/payments/internal/pkg/payments"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConnectorStore interface {
	IsInstalled(ctx context.Context, name string) (bool, error)
	Install(ctx context.Context, name string, config any) error
	Uninstall(ctx context.Context, name string) error
	UpdateConfig(ctx context.Context, name string, config any) error
	Enable(ctx context.Context, name string) error
	Disable(ctx context.Context, name string) error
	IsEnabled(ctx context.Context, name string) (bool, error)
	ReadConfig(ctx context.Context, name string, to interface{}) error
}

type InMemoryConnectorStore struct {
	installed map[string]bool
	disabled  map[string]bool
	configs   map[string]any
}

func (i *InMemoryConnectorStore) Uninstall(ctx context.Context, name string) error {
	delete(i.installed, name)
	delete(i.configs, name)
	delete(i.disabled, name)

	return nil
}

func (i *InMemoryConnectorStore) IsInstalled(ctx context.Context, name string) (bool, error) {
	return i.installed[name], nil
}

func (i *InMemoryConnectorStore) Install(ctx context.Context, name string, config any) error {
	i.installed[name] = true
	i.configs[name] = config
	i.disabled[name] = false

	return nil
}

func (i *InMemoryConnectorStore) UpdateConfig(ctx context.Context, name string, config any) error {
	i.configs[name] = config

	return nil
}

func (i *InMemoryConnectorStore) Enable(ctx context.Context, name string) error {
	i.disabled[name] = false

	return nil
}

func (i *InMemoryConnectorStore) Disable(ctx context.Context, name string) error {
	i.disabled[name] = true

	return nil
}

func (i *InMemoryConnectorStore) IsEnabled(ctx context.Context, name string) (bool, error) {
	disabled, ok := i.disabled[name]
	if !ok {
		return false, nil
	}

	return !disabled, nil
}

func (i *InMemoryConnectorStore) ReadConfig(ctx context.Context, name string, to interface{}) error {
	cfg, ok := i.configs[name]
	if !ok {
		return ErrNotFound
	}

	reflect.ValueOf(to).Elem().Set(reflect.ValueOf(cfg))

	return nil
}

var _ ConnectorStore = &InMemoryConnectorStore{}

func NewInMemoryStore() *InMemoryConnectorStore {
	return &InMemoryConnectorStore{
		installed: make(map[string]bool),
		disabled:  make(map[string]bool),
		configs:   make(map[string]any),
	}
}

type MongodbConnectorStore struct {
	db *mongo.Database
}

func (m *MongodbConnectorStore) Uninstall(ctx context.Context, name string) error {
	return m.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) error {
		_, err := ctx.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			_, err := m.db.Collection(payments.TasksCollection).DeleteMany(ctx, map[string]any{
				"provider": name,
			})
			if err != nil {
				return nil, errors.Wrap(err, "deleting tasks")
			}

			_, err = m.db.Collection(payments.Collection).DeleteMany(ctx, map[string]any{
				"provider": name,
			})
			if err != nil {
				return nil, errors.Wrap(err, "deleting payments")
			}

			_, err = m.db.Collection(payments.ConnectorsCollection).DeleteOne(ctx, map[string]any{
				"provider": name,
			})
			if err != nil {
				return nil, errors.Wrap(err, "deleting configuration")
			}

			return nil, nil
		})

		return err
	})
}

func (m *MongodbConnectorStore) IsInstalled(ctx context.Context, name string) (bool, error) {
	ret := m.db.Collection(payments.ConnectorsCollection).FindOne(ctx, map[string]any{
		"provider": name,
	})
	if ret.Err() != nil {
		if errors.Is(ret.Err(), mongo.ErrNoDocuments) {
			return false, nil
		}

		return false, ret.Err()
	}

	return true, nil
}

func (m *MongodbConnectorStore) Install(ctx context.Context, name string, config any) error {
	_, err := m.db.Collection(payments.ConnectorsCollection).UpdateOne(ctx, map[string]any{
		"provider": name,
	}, map[string]any{
		"$set": map[string]any{
			"config": config,
		},
	}, options.Update().SetUpsert(true))

	return err
}

func (m *MongodbConnectorStore) UpdateConfig(ctx context.Context, name string, config any) error {
	_, err := m.db.Collection(payments.ConnectorsCollection).UpdateOne(ctx, map[string]any{
		"provider": name,
	}, map[string]any{
		"$set": map[string]any{
			"config": config,
		},
	})

	return err
}

func (m *MongodbConnectorStore) Enable(ctx context.Context, name string) error {
	_, err := m.db.Collection(payments.ConnectorsCollection).UpdateOne(ctx, map[string]any{
		"provider": name,
	}, map[string]any{
		"$set": map[string]any{
			"disabled": false,
		},
	})

	return err
}

func (m *MongodbConnectorStore) Disable(ctx context.Context, name string) error {
	_, err := m.db.Collection(payments.ConnectorsCollection).UpdateOne(ctx, map[string]any{
		"provider": name,
	}, map[string]any{
		"$set": map[string]any{
			"disabled": true,
		},
	})

	return err
}

func (m *MongodbConnectorStore) IsEnabled(ctx context.Context, name string) (bool, error) {
	ret := m.db.Collection(payments.ConnectorsCollection).FindOne(ctx, map[string]any{
		"provider": name,
	})

	if ret.Err() != nil {
		if errors.Is(ret.Err(), mongo.ErrNoDocuments) {
			return false, ErrNotInstalled
		}

		return false, ret.Err()
	}

	paymentsConnector := payments.Connector[payments.EmptyConnectorConfig]{}

	if err := ret.Decode(&paymentsConnector); err != nil {
		return false, err
	}

	return !paymentsConnector.Disabled, nil
}

func (m *MongodbConnectorStore) ReadConfig(ctx context.Context, name string, to interface{}) error {
	ret := m.db.Collection(payments.ConnectorsCollection).FindOne(ctx, map[string]any{
		"provider": name,
	})
	if ret.Err() != nil {
		if errors.Is(ret.Err(), mongo.ErrNoDocuments) {
			return errors.New("not installed")
		}

		return ret.Err()
	}

	paymentsConnector := payments.Connector[bson.Raw]{}

	if err := ret.Decode(&paymentsConnector); err != nil {
		return err
	}

	return bson.Unmarshal(paymentsConnector.Config, to)
}

var _ ConnectorStore = &MongodbConnectorStore{}

func NewMongoDBConnectorStore(db *mongo.Database) *MongodbConnectorStore {
	return &MongodbConnectorStore{
		db: db,
	}
}
