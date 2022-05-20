package bridge

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedpublish"
	payment "github.com/numary/payments/pkg"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const (
	PaymentsTopics = "payments"
)

type Event struct {
	Date    time.Time   `json:"date"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type BatchElement struct {
	Payment payment.Payment
	Forward bool
}

type Batch []BatchElement

type Ingester[T ConnectorConfigObject, S ConnectorState, C Connector[T, S]] interface {
	Ingest(ctx context.Context, batch Batch, commitState S) error
}

type defaultIngester[T ConnectorConfigObject, S ConnectorState, C Connector[T, S]] struct {
	db        *mongo.Database
	logger    sharedlogging.Logger
	publisher sharedpublish.Publisher
}

func (i *defaultIngester[T, S, C]) Ingest(ctx context.Context, batch Batch, commitState S) error {

	i.logger.WithFields(map[string]interface{}{
		"size": len(batch),
	}).Infof("Ingest batch")

	ses, err := i.db.Client().StartSession()
	if err != nil {
		return err
	}
	defer ses.EndSession(ctx)

	err = ses.StartTransaction()
	if err != nil {
		return err
	}
	defer ses.AbortTransaction(ctx)

	for _, elem := range batch {
		logger := i.logger.WithFields(map[string]any{
			"id":   elem.Payment.Identifier.String(),
			"date": elem.Payment.Date,
		})

		items := bson.M{
			"$each": []any{elem.Payment},
		}
		if elem.Forward {
			items["$position"] = 0
		}

		_, err = i.db.Collection(payment.PaymentsCollection).UpdateOne(ctx, elem.Payment.Identifier, map[string]any{
			"$push": bson.M{
				"items": items,
			},
		}, options.Update().SetUpsert(true))
		if err != nil {
			logger.Errorf("Error persisting payment: %s", err)
			return err
		}
	}

	for _, e := range batch {
		err = i.publisher.Publish(ctx, PaymentsTopics, Event{
			Date:    time.Now(),
			Type:    "SAVED_PAYMENT",
			Payload: e.Payment,
		})
		if err != nil {
			i.logger.Errorf("Error publishing payment: %s", err)
		}
	}

	i.logger.WithFields(map[string]interface{}{
		"state": commitState,
	}).Infof("Update state")

	var connector C
	_, err = i.db.Collection("Connectors").UpdateOne(ctx, map[string]any{
		"provider": connector.Name(),
	}, bson.M{
		"$set": bson.M{
			"state": commitState,
		},
	})
	if err != nil {
		return err
	}

	err = ses.CommitTransaction(ctx)
	if err != nil {
		return err
	}

	i.logger.WithFields(map[string]interface{}{
		"size": len(batch),
	}).Infof("Batch ingested")

	return nil
}

func NewDefaultIngester[T ConnectorConfigObject, S ConnectorState, C Connector[T, S]](
	db *mongo.Database,
	logger sharedlogging.Logger,
	publisher sharedpublish.Publisher,
) *defaultIngester[T, S, C] {
	var connector C
	return &defaultIngester[T, S, C]{
		db: db,
		logger: logger.WithFields(map[string]interface{}{
			"connector": connector.Name(),
			"component": "ingester",
		}),
		publisher: publisher,
	}
}
