package bridge

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	payment "github.com/numary/payments/pkg"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

var (
	ErrNotFound = errors.New("not found")
)

type ConnectorManager[CONFIG payment.ConnectorConfigObject, STATE payment.ConnectorState] struct {
	connector        Connector[CONFIG, STATE]
	db               *mongo.Database
	name             string
	logObjectStorage LogObjectStorage
}

func (l *ConnectorManager[CONFIG, STATE]) logger(ctx context.Context) sharedlogging.Logger {
	return sharedlogging.GetLogger(ctx).WithFields(map[string]interface{}{
		"connector": l.name,
	})
}

func (l *ConnectorManager[CONFIG, STATE]) Configure(ctx context.Context, config CONFIG) error {

	l.logger(ctx).WithFields(map[string]interface{}{
		"config": config,
	}).Info("Updating connector config")
	_, err := l.db.Collection(payment.ConnectorConfigsCollection).UpdateOne(ctx, map[string]any{
		"provider": l.name,
	}, map[string]any{
		"$set": map[string]any{
			"provider": l.name,
			"config":   config,
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}

	return nil
}

func (l *ConnectorManager[CONFIG, STATE]) Enable(ctx context.Context) error {

	l.logger(ctx).Info("Enabling connector")
	_, err := l.db.Collection(payment.ConnectorConfigsCollection).UpdateOne(ctx, map[string]any{
		"provider": l.name,
	}, map[string]any{
		"$set": map[string]any{
			"disabled": false,
		},
	}, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}

	return nil
}

func (l *ConnectorManager[CONFIG, STATE]) ReadConfig(ctx context.Context) (*CONFIG, bool, error) {
	c := &payment.Connector[CONFIG]{}

	l.logger(ctx).Infof("Will find config")
	ret := l.db.Collection(payment.ConnectorConfigsCollection).FindOne(ctx, map[string]any{
		"provider": l.name,
	})
	l.logger(ctx).Infof("Config query terminated")
	if ret.Err() != nil {
		if ret.Err() == mongo.ErrNoDocuments {
			return nil, false, ErrNotFound
		}
		return nil, false, ret.Err()
	}
	l.logger(ctx).Infof("Decode config")
	err := ret.Decode(c)
	l.logger(ctx).Infof("Decode config terminated")
	if err != nil {
		return nil, false, err
	}

	config := l.connector.ApplyDefaults(c.Config)

	return &config, c.Disabled, nil
}

func (l *ConnectorManager[CONFIG, STATE]) ReadState(ctx context.Context) (STATE, error) {
	connectorState := &payment.State[STATE]{}

	var zero STATE
	ret := l.db.Collection(payment.ConnectorStatesCollection).FindOne(ctx, map[string]any{
		"provider": l.name,
	})
	if ret.Err() != nil {
		if ret.Err() == mongo.ErrNoDocuments {
			return zero, nil
		}
		return zero, ret.Err()
	}
	err := ret.Decode(connectorState)
	if err != nil {
		return zero, err
	}

	return connectorState.State, nil
}

func (l *ConnectorManager[CONFIG, STATE]) Restart(ctx context.Context) error {
	l.logger(ctx).Infof("Restarting connector %s", l.name)
	err := func() error {
		err := l.Stop(ctx)
		if err != nil {
			return err
		}
		return l.Start(ctx)
	}()
	if err != nil {
		l.logger(ctx).Errorf("Error restarting connector: %s", err)
	}
	return err
}

func (l *ConnectorManager[CONFIG, STATE]) Stop(ctx context.Context) error {
	l.logger(ctx).Infof("Stopping connector")

	err := l.connector.Stop(ctx)
	if err != nil {
		l.logger(ctx).Errorf("Error stopping connector: %s", err)
	}
	l.logger(ctx).Infof("Connector stopped")
	return err
}

func (l *ConnectorManager[CONFIG, STATE]) StartWithConfig(ctx context.Context, config CONFIG) error {
	config = l.connector.ApplyDefaults(config)
	l.logger(ctx).WithFields(map[string]interface{}{
		"config": config,
	}).Infof("Starting connector %s", l.name)

	state, err := l.ReadState(ctx)
	if err != nil {
		return err
	}

	return l.StartWithConfigAndState(ctx, config, state)
}

func (l *ConnectorManager[CONFIG, STATE]) StartWithConfigAndState(ctx context.Context, config CONFIG, state STATE) error {
	config = l.connector.ApplyDefaults(config)
	l.logger(ctx).WithFields(map[string]interface{}{
		"config": config,
		"state":  state,
	}).Infof("Starting connector %s", l.name)

	go func() {
		err := l.connector.Start(context.Background(), config, state)
		if err != nil {
			l.logger(ctx).Errorf("Error starting connector: %s", err)
		}
	}()

	return nil
}

func (l *ConnectorManager[CONFIG, STATE]) Start(ctx context.Context) error {
	l.logger(ctx).Info("Start")
	config, _, err := l.ReadConfig(ctx)
	if err != nil {
		return err
	}

	return l.StartWithConfig(ctx, *config)
}

func (l *ConnectorManager[CONFIG, STATE]) Restore(ctx context.Context) error {
	l.logger(ctx).Info("Restoring state")

	config, disabled, err := l.ReadConfig(ctx)
	if err != nil {
		if err == ErrNotFound {
			l.logger(ctx).Info("Not enabled, skip")
			return nil
		}
		return err
	}
	if disabled {
		l.logger(ctx).Errorf("Connector disabled")
		return nil
	}

	err = l.StartWithConfig(ctx, *config)
	if err != nil && err != mongo.ErrNoDocuments {
		l.logger(ctx).Errorf("Unable to restore state: %s", err)
		return err
	}
	l.logger(ctx).Info("State restored")
	return nil
}

func (l *ConnectorManager[CONFIG, STATE]) Disable(ctx context.Context) error {
	l.logger(ctx).Info("Disabling connector")

	_, err := l.db.Collection(payment.ConnectorConfigsCollection).UpdateOne(ctx, map[string]any{
		"provider": l.name,
	}, map[string]any{
		"$set": map[string]any{
			"disabled": true,
		},
	})
	return err
}

func (l *ConnectorManager[CONFIG, STATE]) Reset(ctx context.Context) error {
	l.logger(ctx).Infof("Reset connector")

	err := l.Stop(ctx)
	if err != nil {
		return err
	}

	err = l.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) error {
		var deleted int64
		_, err = ctx.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			ret, err := l.db.Collection(payment.Collection).DeleteMany(ctx, map[string]interface{}{
				"provider": l.name,
			})
			if err != nil {
				return nil, errors.Wrap(err, "Removing payments")
			}
			deleted = ret.DeletedCount
			return nil, l.ResetState(ctx)
		}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.New(writeconcern.WMajority())))
		if err == nil {
			l.logger(ctx).Infof("%d payments deleted", deleted)
		}

		return err
	})
	if err != nil {
		l.logger(ctx).Errorf("Error cleaning data: %s", err)
		return err
	}

	err = l.logObjectStorage.drop(ctx)
	if err != nil {
		return err
	}

	config, disabled, err := l.ReadConfig(ctx)
	if err != nil {
		return err
	}

	if !disabled {
		var state STATE
		err = l.StartWithConfigAndState(ctx, *config, state)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *ConnectorManager[CONFIG, STATE]) ResetState(ctx context.Context) error {
	_, err := l.db.Collection(payment.ConnectorStatesCollection).DeleteOne(ctx, map[string]any{
		"provider": l.name,
	})
	return err
}

func NewConnectorManager[CONFIG payment.ConnectorConfigObject, STATE payment.ConnectorState, CONNECTOR Connector[CONFIG, STATE]](
	db *mongo.Database,
	ctrl Loader[CONFIG, STATE, CONNECTOR],
	ingester Ingester[STATE],
) *ConnectorManager[CONFIG, STATE] {
	var connector CONNECTOR
	logger := sharedlogging.WithFields(map[string]interface{}{
		"connector": connector.Name(),
	})

	logObjectStorage := NewDefaultLogObjectStorage(connector.Name(), db, logger.WithFields(map[string]interface{}{
		"component": "log-object-storage",
	}))

	connector, err := ctrl.Load(logObjectStorage, logger, ingester)
	if err != nil {
		panic(err)
	}

	return &ConnectorManager[CONFIG, STATE]{
		db:               db,
		connector:        connector,
		name:             connector.Name(),
		logObjectStorage: logObjectStorage,
	}
}
