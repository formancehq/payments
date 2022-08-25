package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedpublish"
	"github.com/numary/payments/pkg/core"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	TopicPayments             = "payments"
	EventPaymentsSavedPayment = "SAVED_PAYMENT"
)

type EventPaymentsMessage struct {
	Date    time.Time            `json:"date"`
	Type    string               `json:"type"`
	Payload core.ComputedPayment `json:"payload"`
}

type BatchElement struct {
	Referenced core.Referenced
	Payment    *core.Data
	Adjustment *core.Adjustment
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
	descriptor core.TaskDescriptor
	publisher  sharedpublish.Publisher
}

type referenced core.Referenced

func (i *defaultIngester) processBatch(ctx context.Context, batch Batch) ([]core.Payment, error) {
	allPayments := make([]core.Payment, 0)
	for _, elem := range batch {
		logger := i.logger.WithFields(map[string]any{
			"id": referenced(elem.Referenced),
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
				"$set": core.Payment{
					Identifier: core.Identifier{
						Referenced: elem.Referenced,
						Provider:   i.provider,
					},
					Data: *elem.Payment,
					Adjustments: []core.Adjustment{
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
						"$each": []any{core.Adjustment{
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
		ret := i.db.Collection(core.Collection).FindOneAndUpdate(
			ctx,
			core.Identifier{
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
		p := core.Payment{}
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
		var allPayments []core.Payment
		_, err = ctx.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			allPayments, err = i.processBatch(ctx, batch)
			if err != nil {
				return nil, err
			}

			i.logger.Debugf("Update state")

			_, err = i.db.Collection(core.TasksCollection).UpdateOne(ctx, map[string]any{
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
		allPayments = Filter(allPayments, core.Payment.HasInitialValue)

		if i.publisher != nil {
			for _, e := range allPayments {
				err = i.publisher.Publish(ctx, TopicPayments, EventPaymentsMessage{
					Date:    time.Now().UTC(),
					Type:    EventPaymentsSavedPayment,
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

func NewDefaultIngester(
	provider string,
	descriptor core.TaskDescriptor,
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
