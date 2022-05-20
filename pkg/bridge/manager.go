package bridge

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	payment "github.com/numary/payments/pkg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type connectorWithRawConfig struct {
	Config bson.Raw `bson:"config"`
	State  bson.Raw `bson:"state"`
}

type ConnectorManager[T ConnectorConfigObject, S ConnectorState] struct {
	connector Connector[T, S]
	db        *mongo.Database
	name      string
}

func (l *ConnectorManager[T, S]) logger(ctx context.Context) sharedlogging.Logger {
	return sharedlogging.GetLogger(ctx).WithFields(map[string]interface{}{
		"connector": l.name,
	})
}

func (l *ConnectorManager[T, S]) Configure(ctx context.Context, config T) error {

	l.logger(ctx).WithFields(map[string]interface{}{
		"config": config,
	}).Info("Updating connector config")
	_, err := l.db.Collection("Connectors").UpdateOne(ctx, map[string]any{
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

func (l *ConnectorManager[T, S]) Enable(ctx context.Context) error {

	l.logger(ctx).Info("Enabling connector")
	_, err := l.db.Collection("Connectors").UpdateOne(ctx, map[string]any{
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

func (l *ConnectorManager[T, S]) ReadConfig(ctx context.Context, config *T) error {
	c := &connectorWithRawConfig{}

	ret := l.db.Collection("Connectors").FindOne(ctx, map[string]any{
		"provider": l.name,
	})
	if ret.Err() != nil {
		return ret.Err()
	}
	err := ret.Decode(c)
	if err != nil {
		return err
	}

	err = bson.Unmarshal(c.Config, config)
	if err != nil {
		return err
	}

	*config = l.connector.ApplyDefaults(*config)
	return nil
}

func (l *ConnectorManager[T, S]) ReadState(ctx context.Context, state *S) error {
	c := &connectorWithRawConfig{}

	ret := l.db.Collection("Connectors").FindOne(ctx, map[string]any{
		"provider": l.name,
	})
	if ret.Err() != nil {
		if ret.Err() == mongo.ErrNoDocuments {
			return nil
		}
		return ret.Err()
	}
	err := ret.Decode(c)
	if err != nil {
		return err
	}

	if c.State == nil || len(c.State) == 0 {
		return nil
	}

	return bson.Unmarshal(c.State, state)
}

func (l *ConnectorManager[T, S]) Restart(ctx context.Context) error {
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

func (l *ConnectorManager[T, S]) Stop(ctx context.Context) error {
	l.logger(ctx).Infof("Stopping connector %s", l.name)

	err := l.connector.Stop(ctx)
	if err != nil {
		l.logger(ctx).Errorf("Error stopping connector: %s", err)
	}
	return err
}

func (l *ConnectorManager[T, S]) StartWithConfig(ctx context.Context, config T) error {
	config = l.connector.ApplyDefaults(config)
	l.logger(ctx).WithFields(map[string]interface{}{
		"config": config,
	}).Infof("Starting connector %s", l.name)

	var state S
	err := l.ReadState(ctx, &state)
	if err != nil {
		return err
	}

	return l.StartWithConfigAndState(ctx, config, state)
}

func (l *ConnectorManager[T, S]) StartWithState(ctx context.Context, state S) error {
	var config T
	err := l.ReadConfig(ctx, &config)
	if err != nil {
		return err
	}

	return l.StartWithConfigAndState(ctx, config, state)
}

func (l *ConnectorManager[T, S]) StartWithConfigAndState(ctx context.Context, config T, state S) error {
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

func (l *ConnectorManager[T, S]) Start(ctx context.Context) error {

	l.logger(ctx).Info("Start")
	var config T
	err := l.ReadConfig(ctx, &config)
	if err != nil {
		return err
	}

	return l.StartWithConfig(ctx, config)
}

func (l *ConnectorManager[T, S]) Restore(ctx context.Context) error {
	l.logger(ctx).Info("Restoring state")
	err := l.Start(ctx)
	if err != nil && err != mongo.ErrNoDocuments {
		l.logger(ctx).Errorf("Unable to restore state: %s", err)
		return err
	}
	if err == mongo.ErrNoDocuments {
		l.logger(ctx).Info("Not enabled, skip")
		return nil
	}
	l.logger(ctx).Info("State restored")
	return nil
}

func (l *ConnectorManager[T, S]) Disable(ctx context.Context) error {
	l.logger(ctx).Info("Disabling connector")

	_, err := l.db.Collection(payment.ConnectorsCollector).UpdateOne(ctx, map[string]any{
		"provider": l.name,
	}, map[string]any{
		"$set": map[string]any{
			"disabled": true,
		},
	})
	return err
}

func (l *ConnectorManager[T, S]) Reset(ctx context.Context) error {
	l.logger(ctx).Infof("Reset connector")

	err := l.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) error {
		err := ctx.StartTransaction()
		if err != nil {
			return err
		}
		defer ctx.AbortTransaction(ctx)

		ret, err := l.db.Collection(payment.PaymentsCollection).DeleteMany(ctx, map[string]interface{}{
			"provider": l.name,
		})
		if err != nil {
			return err
		}
		l.logger(ctx).Infof("%d payments deleted", ret.DeletedCount)

		err = l.ResetState(ctx)
		if err != nil {
			return err
		}

		err = l.Stop(ctx)
		if err != nil {
			return err
		}

		var state S
		err = l.StartWithState(ctx, state)
		if err != nil {
			return err
		}

		err = ctx.CommitTransaction(ctx)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		sharedlogging.GetLogger(ctx).Errorf("Error resetting connector: %s", err)
		return err
	}

	return nil
}

func (l *ConnectorManager[T, S]) ResetState(ctx context.Context) error {
	var zeroState S
	_, err := l.db.Collection(payment.ConnectorsCollector).UpdateOne(ctx, map[string]any{
		"provider": l.name,
	}, map[string]any{
		"$set": map[string]any{
			"state": zeroState,
		},
	})
	return err
}

func NewConnectorManager[T ConnectorConfigObject, S ConnectorState, C Connector[T, S]](
	db *mongo.Database,
	ctrl Controller[T, S, C],
	ingester Ingester[T, S, C],
) *ConnectorManager[T, S] {
	var connector C
	logObjectStorage := NewDefaultLogObjectStorage(connector.Name(), db)
	logger := sharedlogging.WithFields(map[string]interface{}{
		"connector": connector.Name(),
	})
	connector, err := ctrl.New(logObjectStorage, logger, ingester)
	if err != nil {
		panic(err)
	}
	return &ConnectorManager[T, S]{
		db:        db,
		connector: connector,
		name:      connector.Name(),
	}
}
