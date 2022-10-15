package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	payments2 "github.com/numary/payments/internal/pkg/payments"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedpublish"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BatchElement struct {
	Referenced payments2.Referenced
	Payment    *payments2.Data
	Adjustment *payments2.Adjustment
	Forward    bool
}

type Batch []BatchElement

type Ingester interface {
	Ingest(ctx context.Context, batch Batch, commitState any) error
}
type IngesterFn func(ctx context.Context, batch Batch, commitState any) error

func (fn IngesterFn) Ingest(ctx context.Context, batch Batch, commitState any) error {
	return fn(ctx, batch, commitState)
}

func NoOpIngester() IngesterFn {
	return IngesterFn(func(ctx context.Context, batch Batch, commitState any) error {
		return nil
	})
}

type defaultIngester struct {
	db         *mongo.Database
	logger     sharedlogging.Logger
	provider   string
	descriptor payments2.TaskDescriptor
	publisher  sharedpublish.Publisher
}

type referenced payments2.Referenced

func (i *defaultIngester) processBatch(ctx context.Context, batch Batch) ([]payments2.Payment, error) {
	allPayments := make([]payments2.Payment, 0)
	for _, elem := range batch {
		logger := i.logger.WithFields(map[string]any{
			"id": referenced(elem.Referenced),
		})

		var update bson.M

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
				"$set": payments2.Payment{
					Identifier: payments2.Identifier{
						Referenced: elem.Referenced,
						Provider:   i.provider,
					},
					Data: *elem.Payment,
					Adjustments: []payments2.Adjustment{
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
						"$each": []any{payments2.Adjustment{
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

		data, err := json.Marshal(update)
		if err != nil {
			panic(err)
		}
		logger.WithFields(map[string]interface{}{
			"update": string(data),
		}).Debugf("Update payment")
		ret := i.db.Collection(payments2.Collection).FindOneAndUpdate(
			ctx,
			payments2.Identifier{
				Referenced: elem.Referenced,
				Provider:   i.provider,
			},
			update,
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		)
		if ret.Err() != nil {
			logger.Errorf("Error updating payment: %s", ret.Err())
			return nil, fmt.Errorf("error updating payment: %s", ret.Err())
		}
		p := payments2.Payment{}
		err = ret.Decode(&p)
		if err != nil {
			return nil, err
		}

		allPayments = append(allPayments, p)
	}
	return allPayments, nil
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

func (i *defaultIngester) Ingest(ctx context.Context, batch Batch, commitState any) error {
	startingAt := time.Now()
	i.logger.WithFields(map[string]interface{}{
		"size":       len(batch),
		"startingAt": startingAt,
	}).Debugf("Ingest batch")

	err := i.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) (err error) {
		var allPayments []payments2.Payment
		_, err = ctx.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			allPayments, err = i.processBatch(ctx, batch)
			if err != nil {
				return nil, err
			}

			i.logger.Debugf("Update state")

			_, err = i.db.Collection(payments2.TasksCollection).UpdateOne(ctx, map[string]any{
				"provider":   i.provider,
				"descriptor": i.descriptor,
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
		allPayments = Filter(allPayments, payments2.Payment.HasInitialValue)

		if i.publisher != nil {
			for _, e := range allPayments {
				i.publish(ctx, TopicPayments,
					NewEventSavedPayment(
						e.Computed()))
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

func NewDefaultIngester(
	provider string,
	descriptor payments2.TaskDescriptor,
	db *mongo.Database,
	logger sharedlogging.Logger,
	publisher sharedpublish.Publisher,
) *defaultIngester {
	return &defaultIngester{
		provider:   provider,
		descriptor: descriptor,
		db:         db,
		logger:     logger,
		publisher:  publisher,
	}
}
