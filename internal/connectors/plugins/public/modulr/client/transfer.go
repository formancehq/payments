package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type DestinationType string

const (
	DestinationTypeAccount     DestinationType = "ACCOUNT"
	DestinationTypeBeneficiary DestinationType = "BENEFICIARY"
)

type Destination struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Details struct {
	SourceAccountID string      `json:"sourceAccountId"`
	Destination     Destination `json:"destination"`
	Currency        string      `json:"currency"`
	Amount          json.Number `json:"amount"`
}

type TransferRequest struct {
	IdempotencyKey    string      `json:"-"`
	SourceAccountID   string      `json:"sourceAccountId"`
	Destination       Destination `json:"destination"`
	Currency          string      `json:"currency"`
	Amount            json.Number `json:"amount"`
	Reference         string      `json:"reference"`
	ExternalReference string      `json:"externalReference"`
}

type getTransferResponse struct {
	Content []TransferResponse `json:"content"`
}

type TransferResponse struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	CreatedDate       string  `json:"createdDate"`
	ExternalReference string  `json:"externalReference"`
	ApprovalStatus    string  `json:"approvalStatus"`
	Message           string  `json:"message"`
	Details           Details `json:"details"`
}

func (c *client) InitiateTransfer(ctx context.Context, transferRequest *TransferRequest) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	body, err := json.Marshal(transferRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.buildEndpoint("payments"), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-mod-nonce", transferRequest.IdempotencyKey)

	var res TransferResponse
	var errRes modulrErrors
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to initiate transfer: %v", errRes.Error()),
			err,
		)
	}
	return &res, nil
}

func (c *client) GetTransfer(ctx context.Context, transferID string) (TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_transfer")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("payments?id=%s", transferID), nil)
	if err != nil {
		return TransferResponse{}, fmt.Errorf("failed to create get transfer request: %w", err)
	}

	var res getTransferResponse
	var errRes modulrErrors
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return TransferResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get transfer: %v", errRes.Error()),
			err,
		)
	}

	if len(res.Content) == 0 {
		return TransferResponse{}, fmt.Errorf("transfer not found")
	}
	return res.Content[0], nil
}
