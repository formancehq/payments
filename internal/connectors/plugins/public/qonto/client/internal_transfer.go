package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"net/http"
)

type TransferRequest struct {
	SourceIBAN      string `json:"debit_iban"`
	DestinationIBAN string `json:"credit_iban"`
	Reference       string `json:"reference"`
	Currency        string `json:"currency"`
	Amount          string `json:"amount"`
}

type TransferResponse struct {
	Id          string      `json:"id"`
	Slug        string      `json:"slug"`
	Status      string      `json:"status"`
	Amount      json.Number `json:"amount"`
	AmountCents json.Number `json:"amount_cents"`
	Currency    string      `json:"currency"`
	Reference   string      `json:"reference"`
	CreatedDate string      `json:"created_at"`
}

func (c *client) CreateInternalTransfer(
	ctx context.Context,
	idempotencyKey string,
	request TransferRequest,
) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	type qontoRequest struct {
		InternalTransfer TransferRequest `json:"internal_transfer"`
	}
	body, err := json.Marshal(qontoRequest{
		InternalTransfer: request,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildEndpoint("v2/internal_transfers"), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Qonto-Idempotency-Key", idempotencyKey)

	errorResponse := qontoErrors{}

	type qontoResponse struct {
		InternalTransfer TransferResponse `json:"internal_transfer"`
	}
	successResponse := qontoResponse{}

	_, err = c.httpClient.Do(ctx, req, &successResponse, &errorResponse)

	if err != nil {
		return nil, errorsutils.NewWrappedError(
			err,
			fmt.Errorf("failed to create transfer: %w", errorResponse.Error()),
		)
	}
	return &successResponse.InternalTransfer, nil
}
