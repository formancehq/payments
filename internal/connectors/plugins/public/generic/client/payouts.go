package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type PayoutRequest struct {
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"`
	SourceAccountId      string            `json:"sourceAccountId"`
	DestinationAccountId string            `json:"destinationAccountId"`
	Description          *string           `json:"description,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type PayoutResponse struct {
	Id                   string            `json:"id"`
	IdempotencyKey       string            `json:"idempotencyKey"`
	Amount               string            `json:"amount"`
	Currency             string            `json:"currency"`
	SourceAccountId      string            `json:"sourceAccountId"`
	DestinationAccountId string            `json:"destinationAccountId"`
	Description          *string           `json:"description,omitempty"`
	Status               string            `json:"status"`
	CreatedAt            string            `json:"createdAt"`
	UpdatedAt            *string           `json:"updatedAt,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

func (c *client) CreatePayout(ctx context.Context, request *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_payout")
	
	// TODO: Once the OpenAPI client is regenerated, replace this with actual API calls
	// For now, return a mock response to satisfy the interface
	return &PayoutResponse{
		Id:                   "payout_" + request.IdempotencyKey,
		IdempotencyKey:       request.IdempotencyKey,
		Amount:               request.Amount, // Pass through the amount as-is
		Currency:             request.Currency,
		SourceAccountId:      request.SourceAccountId,
		DestinationAccountId: request.DestinationAccountId,
		Description:          request.Description,
		Status:               "PENDING",
		CreatedAt:            "2024-01-01T00:00:00Z",
		UpdatedAt:            nil,
		Metadata:             request.Metadata,
	}, nil
}

func (c *client) GetPayoutStatus(ctx context.Context, payoutId string) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_payout_status")
	
	// TODO: Once the OpenAPI client is regenerated, replace this with actual API calls
	// For now, return a mock response to satisfy the interface
	return &PayoutResponse{
		Id:                   payoutId,
		IdempotencyKey:       payoutId + "_key",
		Amount:               "1000",
		Currency:             "USD",
		SourceAccountId:      "source_account",
		DestinationAccountId: "dest_account",
		Description:          nil,
		Status:               "SUCCEEDED",
		CreatedAt:            "2024-01-01T00:00:00Z",
		UpdatedAt:            nil,
		Metadata:             make(map[string]string),
	}, nil
}