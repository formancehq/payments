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

type BatchElement struct {
	Identifier payment.Identifier
	Payment    *payment.Data
	Adjustment *payment.Adjustment
	Forward    bool
}

type Batch []BatchElement

type Ingester[STATE payment.ConnectorState] interface {
	Ingest(ctx context.Context, batch Batch, commitState STATE) error
}
type IngesterFn[STATE payment.ConnectorState] func(ctx context.Context, batch Batch, commitState STATE) error

func (fn IngesterFn[STATE]) Ingest(ctx context.Context, batch Batch, commitState STATE) error {
	return fn(ctx, batch, commitState)
}

func NoOpIngester[STATE payment.ConnectorState]() IngesterFn[STATE] {
	return IngesterFn[STATE](func(ctx context.Context, batch Batch, commitState STATE) error {
		return nil
	})
}

type defaultIngester[STATE payment.ConnectorState] struct {
	db        *mongo.Database
	logger    sharedlogging.Logger
	publisher sharedpublish.Publisher
	name      string
}

func (i *defaultIngester[STATE]) processBatch(ctx context.Context, batch Batch) ([]payment.Payment, error) {
	payments := make([]payment.Payment, 0)
	for _, elem := range batch {
		logger := i.logger.WithFields(map[string]any{
			"id": elem.Identifier.String(),
		})

		var (
			update bson.M
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
			update = bson.M{
				"$set": payment.Payment{
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
					"asset":         elem.Payment.Asset,
				},
				"$setOnInsert": bson.M{
					"status": elem.Payment.Status,
				},
			}
		}

		ret := i.db.Collection(payment.Collection).FindOneAndUpdate(
			ctx,
			elem.Identifier,
			update,
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		)
		if ret.Err() != nil {
			logger.Errorf("Error updating payment: %s", ret.Err())
			return nil, ret.Err()
		}
		p := payment.Payment{}
		err = ret.Decode(&p)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
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

func (i *defaultIngester[STATE]) Ingest(ctx context.Context, batch Batch, commitState STATE) error {

	startingAt := time.Now()
	i.logger.WithFields(map[string]interface{}{
		"size":       len(batch),
		"startingAt": startingAt,
	}).Debugf("Ingest batch")

	err := i.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) (err error) {
		var payments []payment.Payment
		_, err = ctx.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			payments, err = i.processBatch(ctx, batch)
			if err != nil {
				return nil, err
			}

			i.logger.Debugf("Update state")

			_, err = i.db.Collection(payment.ConnectorStatesCollection).UpdateOne(ctx, map[string]any{
				"provider": i.name,
			}, map[string]any{
				"$set": map[string]any{
					"state": commitState,
				},
			}, options.Update().SetUpsert(true))

			if err != nil {
				return nil, err
			}
			return nil, nil
		})
		payments = Filter(payments, payment.Payment.HasInitialValue)

		if i.publisher != nil {
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
		}
		return err
	})
	if err != nil {
		sharedlogging.GetLogger(ctx).Errorf("Error ingesting batch: %s", err)
		return err
	}

	endedAt := time.Now()

	i.logger.WithFields(map[string]interface{}{
		"size":    len(batch),
		"endedAt": endedAt,
		"latency": endedAt.Sub(startingAt).String(),
	}).Debugf("Batch ingested")

	return nil
}

func NewDefaultIngester[STATE payment.ConnectorState](
	name string,
	db *mongo.Database,
	logger sharedlogging.Logger,
	publisher sharedpublish.Publisher,
) *defaultIngester[STATE] {
	return &defaultIngester[STATE]{
		name:      name,
		db:        db,
		logger:    logger,
		publisher: publisher,
	}
}
