package stripe

import (
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"go.mongodb.org/mongo-driver/mongo"
)

type Controller struct{}

func (c *Controller) New(db *mongo.Database, logger sharedlogging.Logger, ingester bridge.Ingester[Config, State, *Connector]) (*Connector, error) {
	return NewConnector(db, logger, ingester), nil
}

const (
	ConnectorName = "stripe"
)
