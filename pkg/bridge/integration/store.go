package integration

import (
	"context"
	"reflect"

	"github.com/numary/payments/pkg/core"
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

type inMemoryConnectorStore struct {
	installed map[string]bool
	disabled  map[string]bool
	configs   map[string]any
}

func (i *inMemoryConnectorStore) Uninstall(ctx context.Context, name string) error {
	delete(i.installed, name)
	delete(i.configs, name)
	delete(i.disabled, name)
	return nil
}

func (i *inMemoryConnectorStore) IsInstalled(ctx context.Context, name string) (bool, error) {
	return i.installed[name], nil
}

func (i *inMemoryConnectorStore) Install(ctx context.Context, name string, config any) error {
	i.installed[name] = true
	i.configs[name] = config
	i.disabled[name] = false
	return nil
}

func (i inMemoryConnectorStore) UpdateConfig(ctx context.Context, name string, config any) error {
	i.configs[name] = config
	return nil
}

func (i inMemoryConnectorStore) Enable(ctx context.Context, name string) error {
	i.disabled[name] = false
	return nil
}

func (i inMemoryConnectorStore) Disable(ctx context.Context, name string) error {
	i.disabled[name] = true
	return nil
}

func (i inMemoryConnectorStore) IsEnabled(ctx context.Context, name string) (bool, error) {
	disabled, ok := i.disabled[name]
	if !ok {
		return false, nil
	}
	return !disabled, nil
}

func (i inMemoryConnectorStore) ReadConfig(ctx context.Context, name string, to interface{}) error {
	cfg, ok := i.configs[name]
	if !ok {
		return ErrNotFound
	}

	reflect.ValueOf(to).Elem().Set(reflect.ValueOf(cfg))
	return nil
}

var _ ConnectorStore = &inMemoryConnectorStore{}

func NewInMemoryStore() *inMemoryConnectorStore {
	return &inMemoryConnectorStore{
		installed: make(map[string]bool),
		disabled:  make(map[string]bool),
		configs:   make(map[string]any),
	}
}

type mongodbConnectorStore struct {
	db *mongo.Database
}

func (m *mongodbConnectorStore) Uninstall(ctx context.Context, name string) error {
	return m.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) error {
		_, err := ctx.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			_, err := m.db.Collection(core.TasksCollection).DeleteMany(ctx, map[string]any{
				"provider": name,
			})
			if err != nil {
				return nil, errors.Wrap(err, "deleting tasks")
			}
			_, err = m.db.Collection(core.Collection).DeleteMany(ctx, map[string]any{
				"provider": name,
			})
			if err != nil {
				return nil, errors.Wrap(err, "deleting payments")
			}
			_, err = m.db.Collection(core.ConnectorsCollection).DeleteOne(ctx, map[string]any{
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

func (m *mongodbConnectorStore) IsInstalled(ctx context.Context, name string) (bool, error) {
	ret := m.db.Collection(core.ConnectorsCollection).FindOne(ctx, map[string]any{
		"provider": name,
	})
	if ret.Err() != nil && ret.Err() != mongo.ErrNoDocuments {
		return false, ret.Err()
	}
	if ret.Err() == mongo.ErrNoDocuments {
		return false, nil
	}
	return true, nil
}

func (m *mongodbConnectorStore) Install(ctx context.Context, name string, config any) error {
	_, err := m.db.Collection(core.ConnectorsCollection).UpdateOne(ctx, map[string]any{
		"provider": name,
	}, map[string]any{
		"$set": map[string]any{
			"config": config,
		},
	}, options.Update().SetUpsert(true))
	return err
}

func (m *mongodbConnectorStore) UpdateConfig(ctx context.Context, name string, config any) error {
	_, err := m.db.Collection(core.ConnectorsCollection).UpdateOne(ctx, map[string]any{
		"provider": name,
	}, map[string]any{
		"$set": map[string]any{
			"config": config,
		},
	})
	return err
}

func (m *mongodbConnectorStore) Enable(ctx context.Context, name string) error {
	_, err := m.db.Collection(core.ConnectorsCollection).UpdateOne(ctx, map[string]any{
		"provider": name,
	}, map[string]any{
		"$set": map[string]any{
			"disabled": false,
		},
	})
	return err
}

func (m *mongodbConnectorStore) Disable(ctx context.Context, name string) error {
	_, err := m.db.Collection(core.ConnectorsCollection).UpdateOne(ctx, map[string]any{
		"provider": name,
	}, map[string]any{
		"$set": map[string]any{
			"disabled": true,
		},
	})
	return err
}

func (m *mongodbConnectorStore) IsEnabled(ctx context.Context, name string) (bool, error) {
	ret := m.db.Collection(core.ConnectorsCollection).FindOne(ctx, map[string]any{
		"provider": name,
	})
	if ret.Err() != nil && ret.Err() != mongo.ErrNoDocuments {
		return false, ret.Err()
	}
	if ret.Err() == mongo.ErrNoDocuments {
		return false, ErrNotInstalled
	}
	p := core.Connector[core.EmptyConnectorConfig]{}
	err := ret.Decode(&p)
	if err != nil {
		return false, err
	}

	return !p.Disabled, nil
}

func (m *mongodbConnectorStore) ReadConfig(ctx context.Context, name string, to interface{}) error {
	ret := m.db.Collection(core.ConnectorsCollection).FindOne(ctx, map[string]any{
		"provider": name,
	})
	if ret.Err() != nil && ret.Err() != mongo.ErrNoDocuments {
		return ret.Err()
	}
	if ret.Err() == mongo.ErrNoDocuments {
		return errors.New("not installed")
	}
	p := core.Connector[bson.Raw]{}
	err := ret.Decode(&p)
	if err != nil {
		return err
	}

	return bson.Unmarshal(p.Config, to)
}

var _ ConnectorStore = &mongodbConnectorStore{}

func NewMongoDBConnectorStore(db *mongo.Database) *mongodbConnectorStore {
	return &mongodbConnectorStore{
		db: db,
	}
}
