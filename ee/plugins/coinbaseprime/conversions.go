package coinbaseprime

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	// Coinbase Prime conversions are typically tracked via transactions.
	// For now, return empty as there is no dedicated list endpoint.
	// This would need to be implemented by polling transactions and filtering for conversions.
	return models.FetchNextConversionsResponse{
		Conversions: []models.PSPConversion{},
		NewState:    nil,
		HasMore:     false,
	}, nil
}
