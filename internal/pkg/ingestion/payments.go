package ingestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/pkg/payments"

	"github.com/formancehq/go-libs/sharedlogging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PaymentBatchElement struct {
	Referenced payments.Referenced
	Payment    *payments.Data
	Adjustment *payments.Adjustment
	Metadata   payments.Metadata
	Forward    bool
}

type PaymentBatch []PaymentBatchElement

type IngesterFn func(ctx context.Context, batch PaymentBatch, commitState any) error

func (fn IngesterFn) IngestPayments(ctx context.Context, batch PaymentBatch, commitState any) error {
	return fn(ctx, batch, commitState)
}

type referenced payments.Referenced

func (i *DefaultIngester) processPaymentBatch(ctx context.Context, batch PaymentBatch) ([]payments.Payment, error) {
	allPayments := make([]payments.Payment, 0)

	for _, elem := range batch {
		logger := i.logger.WithFields(map[string]any{
			"id": referenced(elem.Referenced),
		})

		var update bson.M

		if elem.Adjustment == nil && elem.Payment == nil {
			return nil, errors.New("either adjustment or payment must be provided")
		}

		var metadataChanges payments.MetadataChanges

		if elem.Payment != nil {
			ret := i.db.Collection(payments.Collection).FindOne(
				ctx,
				payments.Identifier{
					Referenced: elem.Referenced,
					Provider:   i.provider,
				})
			if ret.Err() != nil && !errors.Is(ret.Err(), mongo.ErrNoDocuments) {
				logger.Errorf("Error retrieving payment: %s", ret.Err())

				return nil, fmt.Errorf("error retrieving payment: %w", ret.Err())
			}

			if ret != nil && ret.Err() == nil {
				payment := payments.Payment{}

				err := ret.Decode(&payment)
				if err != nil {
					return nil, err
				}

				metadataChanges = payment.MergeMetadata(elem.Metadata)

				elem.Metadata = metadataChanges.After
			}
		}

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
					"date":   elem.Adjustment.Date,
				},
			}
		case elem.Forward && elem.Payment != nil:
			update = bson.M{
				"$set": payments.Payment{
					Identifier: payments.Identifier{
						Referenced: elem.Referenced,
						Provider:   i.provider,
					},
					Data: *elem.Payment,
					Adjustments: []payments.Adjustment{
						{
							Status: elem.Payment.Status,
							Amount: elem.Payment.InitialAmount,
							Date:   elem.Payment.CreatedAt,
							Raw:    elem.Payment.Raw,
						},
					},
					Metadata: elem.Metadata,
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
						"$each": []any{payments.Adjustment{
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

		ret := i.db.Collection(payments.Collection).FindOneAndUpdate(
			ctx,
			payments.Identifier{
				Referenced: elem.Referenced,
				Provider:   i.provider,
			},
			update,
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		)
		if ret.Err() != nil {
			logger.Errorf("Error updating payment: %s", ret.Err())

			return nil, fmt.Errorf("error updating payment: %w", ret.Err())
		}

		payment := payments.Payment{}

		err = ret.Decode(&payment)
		if err != nil {
			return nil, err
		}

		if metadataChanges.HasChanged() {
			logger.WithFields(map[string]interface{}{
				"metadata": payment.Metadata,
			}).Debugf("Metadata changed")

			_, err = i.db.Collection(payments.MetadataChangelogCollection).InsertOne(ctx, metadataChanges)
			if err != nil {
				return nil, err
			}
		}

		allPayments = append(allPayments, payment)
	}

	return allPayments, nil
}

func filter[T any](objects []T, compareFn func(t T) bool) []T {
	ret := make([]T, 0)

	for _, o := range objects {
		if compareFn(o) {
			ret = append(ret, o)
		}
	}

	return ret
}

func (i *DefaultIngester) IngestPayments(ctx context.Context, batch PaymentBatch, commitState any) error {
	startingAt := time.Now()

	i.logger.WithFields(map[string]interface{}{
		"size":       len(batch),
		"startingAt": startingAt,
	}).Debugf("Ingest batch")

	err := i.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) error {
		var allPayments []payments.Payment
		_, err := ctx.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			var err error

			allPayments, err = i.processPaymentBatch(ctx, batch)
			if err != nil {
				return nil, err
			}

			i.logger.Debugf("Update state")

			_, err = i.db.Collection(payments.TasksCollection).UpdateOne(ctx, map[string]any{
				"provider":   i.provider,
				"descriptor": i.descriptor,
			}, map[string]any{
				"$set": map[string]any{
					"state": commitState,
				},
			}, options.Update().SetUpsert(true))

			return nil, err
		})
		allPayments = filter(allPayments, payments.Payment.HasInitialValue)

		if i.publisher != nil {
			for _, e := range allPayments {
				i.publish(ctx, TopicPayments, NewEventSavedPayment(e.Computed()))
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
