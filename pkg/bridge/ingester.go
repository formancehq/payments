package bridge

import (
	"context"
	"errors"
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
	Date    time.Time               `json:"date"`
	Type    string                  `json:"type"`
	Payload payment.ComputedPayment `json:"payload"`
}

type Order interface {
	apply(ctx context.Context)
}

type BatchElement struct {
	Identifier payment.Identifier
	Payment    *payment.Data
	Adjustment *payment.Adjustment
	Forward    bool
}

type Batch []BatchElement

type Ingester[T payment.ConnectorConfigObject, S payment.ConnectorState, C Connector[T, S]] interface {
	Ingest(ctx context.Context, batch Batch, commitState S) error
}

type defaultIngester[T payment.ConnectorConfigObject, S payment.ConnectorState, C Connector[T, S]] struct {
	db        *mongo.Database
	logger    sharedlogging.Logger
	publisher sharedpublish.Publisher
}

func (i *defaultIngester[T, S, C]) processBatch(ctx context.Context, batch Batch) ([]payment.Payment, error) {
	payments := make([]payment.Payment, 0)
	for _, elem := range batch {
		logger := i.logger.WithFields(map[string]any{
			"id": elem.Identifier.String(),
		})

		var (
			update     bson.M
			newPayment payment.Payment
		)

		if elem.Adjustment != nil && elem.Payment != nil {
			return nil, errors.New("either adjustment or payment must be provided")
		}

		var err error
		switch {
		case elem.Forward && elem.Adjustment != nil:
			update = bson.M{
				"$push": bson.M{
					"adjustments": bson.M{
						"$each":     []any{elem.Adjustment},
						"$position": 0,
					},
				},
				"$set": bson.M{
					"status": elem.Adjustment.Status,
					"raw":    elem.Adjustment.Raw,
					"data":   elem.Adjustment.Date,
				},
			}
		case elem.Forward && elem.Payment != nil:
			newPayment = payment.Payment{
				Identifier: elem.Identifier,
				Data:       *elem.Payment,
				Adjustments: []payment.Adjustment{
					{
						Status: elem.Payment.Status,
						Amount: elem.Payment.InitialAmount,
						Date:   elem.Payment.CreatedAt,
						Raw:    elem.Payment.Raw,
					},
				},
			}
		case !elem.Forward && elem.Adjustment != nil:
			update = bson.M{
				"$push": bson.M{
					"adjustments": bson.M{
						"$each": []any{elem.Adjustment},
					},
				},
				"$setOnInsert": bson.M{
					"status": elem.Adjustment.Status,
				},
			}
		case !elem.Forward && elem.Payment != nil:
			update = bson.M{
				"$push": bson.M{
					"adjustments": bson.M{
						"$each": []any{payment.Adjustment{
							Status: elem.Payment.Status,
							Amount: elem.Payment.InitialAmount,
							Date:   elem.Payment.CreatedAt,
							Raw:    elem.Payment.Raw,
						}},
					},
				},
				"$set": bson.M{
					"raw":           elem.Payment.Raw,
					"createdAt":     elem.Payment.CreatedAt,
					"scheme":        elem.Payment.Scheme,
					"initialAmount": elem.Payment.InitialAmount,
				},
			}
		}

		if update != nil {
			ret := i.db.Collection(payment.PaymentsCollection).FindOneAndUpdate(
				ctx,
				elem.Identifier,
				update,
				options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
			)
			if ret.Err() != nil {
				logger.Errorf("Error updating payment: %s", err)
				return nil, ret.Err()
			}
			p := payment.Payment{}
			err = ret.Decode(&p)
			if err != nil {
				return nil, err
			}
			payments = append(payments, p)
		} else {
			payments = append(payments, newPayment)
			_, err = i.db.Collection(payment.PaymentsCollection).InsertOne(ctx, newPayment)
			if err != nil {
				logger.Errorf("Error inserting payment: %s", err)
				return nil, err
			}
		}

	}
	return payments, nil
}

func Filter[T any](objects []T, compareFn func(t T) bool) []T {
	ret := make([]T, 0)
	for _, o := range objects {
		if compareFn(o) {
			ret = append(ret, o)
		}
	}
	return ret
}

func (i *defaultIngester[T, S, C]) Ingest(ctx context.Context, batch Batch, commitState S) error {

	i.logger.WithFields(map[string]interface{}{
		"size": len(batch),
	}).Infof("Ingest batch")

	err := i.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) error {
		err := ctx.StartTransaction()
		if err != nil {
			return err
		}
		defer ctx.AbortTransaction(ctx)

		payments, err := i.processBatch(ctx, batch)
		if err != nil {
			return err
		}
		payments = Filter(payments, func(p payment.Payment) bool {
			return p.InitialAmount != 0
		})

		for _, e := range payments {
			err = i.publisher.Publish(ctx, PaymentsTopics, Event{
				Date:    time.Now(),
				Type:    "SAVED_PAYMENT",
				Payload: e.Computed(),
			})
			if err != nil {
				i.logger.Errorf("Error publishing payment: %s", err)
			}
		}

		i.logger.WithFields(map[string]interface{}{
			"state": commitState,
		}).Infof("Update state")

		var connector C
		_, err = i.db.Collection(payment.ConnectorsCollector).UpdateOne(ctx, map[string]any{
			"provider": connector.Name(),
		}, bson.M{
			"$set": bson.M{
				"state": commitState,
			},
		})
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
		sharedlogging.GetLogger(ctx).Errorf("Error ingesting batch: %s", err)
		return err
	}

	i.logger.WithFields(map[string]interface{}{
		"size": len(batch),
	}).Infof("Batch ingested")

	return nil
}

func NewDefaultIngester[T payment.ConnectorConfigObject, S payment.ConnectorState, C Connector[T, S]](
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
