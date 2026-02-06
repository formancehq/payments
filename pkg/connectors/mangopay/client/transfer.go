package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
)

type Funds struct {
	Currency string      `json:"Currency"`
	Amount   json.Number `json:"Amount"`
}

type TransferRequest struct {
	Reference        string `json:"-"` // Needed for idempotency
	AuthorID         string `json:"AuthorId"`
	CreditedUserID   string `json:"CreditedUserId,omitempty"`
	DebitedFunds     Funds  `json:"DebitedFunds"`
	Fees             Funds  `json:"Fees"`
	DebitedWalletID  string `json:"DebitedWalletId"`
	CreditedWalletID string `json:"CreditedWalletId"`
}

type TransferResponse struct {
	ID               string `json:"Id"`
	CreationDate     int64  `json:"CreationDate"`
	AuthorID         string `json:"AuthorId"`
	CreditedUserID   string `json:"CreditedUserId"`
	DebitedFunds     Funds  `json:"DebitedFunds"`
	Fees             Funds  `json:"Fees"`
	CreditedFunds    Funds  `json:"CreditedFunds"`
	Status           string `json:"Status"`
	ResultCode       string `json:"ResultCode"`
	ResultMessage    string `json:"ResultMessage"`
	Type             string `json:"Type"`
	ExecutionDate    int64  `json:"ExecutionDate"`
	Nature           string `json:"Nature"`
	DebitedWalletID  string `json:"DebitedWalletId"`
	CreditedWalletID string `json:"CreditedWalletId"`
}

func (c *client) InitiateWalletTransfer(ctx context.Context, transferRequest *TransferRequest) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	endpoint := fmt.Sprintf("%s/v2.01/%s/transfers", c.endpoint, c.clientID)

	body, err := json.Marshal(transferRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", transferRequest.Reference)

	var transferResponse TransferResponse
	var errRes mangopayError
	_, err = c.httpClient.Do(ctx, req, &transferResponse, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to initiate transfer: %v", errRes.Error()),
			err,
		)
	}

	return &transferResponse, nil
}

func (c *client) GetWalletTransfer(ctx context.Context, transferID string) (TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_transfer")

	endpoint := fmt.Sprintf("%s/v2.01/%s/transfers/%s", c.endpoint, c.clientID, transferID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return TransferResponse{}, fmt.Errorf("failed to create login request: %w", err)
	}

	var transfer TransferResponse
	statusCode, err := c.httpClient.Do(ctx, req, &transfer, nil)
	if err != nil {
		return transfer, connector.NewWrappedError(
			fmt.Errorf("failed to get transfer: status code %d", statusCode),
			err,
		)
	}
	return transfer, nil
}
