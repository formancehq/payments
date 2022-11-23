package ingestion

import (
	"context"

	"github.com/formancehq/payments/internal/pkg/payments"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/go-libs/sharedpublish"
	"go.mongodb.org/mongo-driver/mongo"
)

type Ingester interface {
	IngestPayments(ctx context.Context, batch PaymentBatch, commitState any) error
	IngestAccounts(ctx context.Context, batch AccountBatch) error
}

type DefaultIngester struct {
	db         *mongo.Database
	logger     sharedlogging.Logger
	provider   string
	descriptor payments.TaskDescriptor
	publisher  sharedpublish.Publisher
}

func NewDefaultIngester(
	provider string,
	descriptor payments.TaskDescriptor,
	db *mongo.Database,
	logger sharedlogging.Logger,
	publisher sharedpublish.Publisher,
) *DefaultIngester {
	return &DefaultIngester{
		provider:   provider,
		descriptor: descriptor,
		db:         db,
		logger:     logger,
		publisher:  publisher,
	}
}
