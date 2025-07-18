package services

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (s *Service) ConnectorsHandleWebhooks(
	ctx context.Context,
	url string,
	urlPath string,
	webhook models.Webhook,
) error {
	return handleEngineErrors(s.engine.HandleWebhook(ctx, url, urlPath, webhook))
}
