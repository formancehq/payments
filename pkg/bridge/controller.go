package bridge

import (
	"github.com/numary/go-libs/sharedlogging"
	"go.mongodb.org/mongo-driver/mongo"
)

type Controller[T ConnectorConfigObject, S ConnectorState, C Connector[T, S]] interface {
	New(db *mongo.Database, logger sharedlogging.Logger, ingester Ingester[T, S, C]) (C, error)
}
