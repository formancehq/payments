package ingestion

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/app/models"
	"github.com/formancehq/payments/internal/app/payments"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/go-libs/sharedpublish"
)

type Ingester interface {
	IngestPayments(ctx context.Context, batch PaymentBatch, commitState any) error
	IngestAccounts(ctx context.Context, batch AccountBatch) error
}

type DefaultIngester struct {
	repo       Repository
	logger     sharedlogging.Logger
	provider   models.ConnectorProvider
	descriptor payments.TaskDescriptor
	publisher  sharedpublish.Publisher
}

type Repository interface {
	UpsertAccounts(ctx context.Context, provider models.ConnectorProvider, accounts []models.Account) error
	UpsertPayments(ctx context.Context, provider models.ConnectorProvider, payments []*models.Payment) error
	UpdateTaskState(ctx context.Context, provider models.ConnectorProvider, descriptor json.RawMessage, state json.RawMessage) error
}

func NewDefaultIngester(
	provider models.ConnectorProvider,
	descriptor payments.TaskDescriptor,
	repo Repository,
	logger sharedlogging.Logger,
	publisher sharedpublish.Publisher,
) *DefaultIngester {
	return &DefaultIngester{
		provider:   provider,
		descriptor: descriptor,
		repo:       repo,
		logger:     logger,
		publisher:  publisher,
	}
}
