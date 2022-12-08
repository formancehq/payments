package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/app/payments"

	"github.com/formancehq/go-libs/sharedlogging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AccountBatchElement struct {
	Reference string
	Provider  string
	Type      payments.AccountType
}

type AccountBatch []AccountBatchElement

type AccountIngesterFn func(ctx context.Context, batch AccountBatch, commitState any) error

func (fn AccountIngesterFn) IngestAccounts(ctx context.Context, batch AccountBatch, commitState any) error {
	return fn(ctx, batch, commitState)
}

func (i *DefaultIngester) processAccountBatch(ctx context.Context, batch AccountBatch) ([]payments.Account, error) {
	allAccounts := make([]payments.Account, 0)

	for _, elem := range batch {
		logger := i.logger.WithFields(map[string]any{
			"id": elem.Reference,
		})

		update := bson.M{
			"$set": payments.Account{
				Reference: elem.Reference,
				Provider:  i.provider,
				Type:      elem.Type,
			},
		}

		data, err := json.Marshal(update)
		if err != nil {
			panic(err)
		}

		logger.WithFields(map[string]interface{}{
			"update": string(data),
		}).Debugf("Update account")

		ret := i.db.Collection(payments.AccountsCollection).FindOneAndUpdate(
			ctx,
			payments.Account{
				Reference: elem.Reference,
				Provider:  i.provider,
				Type:      elem.Type,
			},
			update,
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		)
		if ret.Err() != nil {
			logger.Errorf("Error updating account: %s", ret.Err())

			return nil, fmt.Errorf("error updating payment: %w", ret.Err())
		}

		account := payments.Account{}

		err = ret.Decode(&account)
		if err != nil {
			return nil, err
		}

		allAccounts = append(allAccounts, account)
	}

	return allAccounts, nil
}

func (i *DefaultIngester) IngestAccounts(ctx context.Context, batch AccountBatch) error {
	startingAt := time.Now()

	i.logger.WithFields(map[string]interface{}{
		"size":       len(batch),
		"startingAt": startingAt,
	}).Debugf("Ingest accounts batch")

	err := i.db.Client().UseSession(ctx, func(ctx mongo.SessionContext) error {
		var allAccounts []payments.Account

		_, err := ctx.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
			var err error

			allAccounts, err = i.processAccountBatch(ctx, batch)
			if err != nil {
				return nil, err
			}

			i.logger.Debugf("Update state")

			return nil, err
		})
		if err != nil {
			return err
		}

		if i.publisher != nil {
			for _, e := range allAccounts {
				i.publish(ctx, TopicPayments, NewEventSavedAccount(e))
			}
		}

		return err
	})
	if err != nil {
		sharedlogging.GetLogger(ctx).Errorf("Error ingesting accounts batch: %s", err)

		return err
	}

	endedAt := time.Now()

	i.logger.WithFields(map[string]interface{}{
		"size":    len(batch),
		"endedAt": endedAt,
		"latency": endedAt.Sub(startingAt).String(),
	}).Debugf("Accounts batch ingested")

	return nil
}
