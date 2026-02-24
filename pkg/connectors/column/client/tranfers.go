package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type TransferRequest struct {
	Amount                int64                  `json:"amount"`
	CurrencyCode          string                 `json:"currency_code"`
	SenderBankAccountId   string                 `json:"sender_bank_account_id,omitempty"`
	ReceiverBankAccountId string                 `json:"receiver_bank_account_id,omitempty"`
	AllowOverdraft        bool                   `json:"allow_overdraft,omitempty"`
	Hold                  bool                   `json:"hold,omitempty"`
	Details               TransferRequestDetails `json:"details,omitempty"`
}

type TransferRequestDetails struct {
	SenderName           string        `json:"sender_name"`
	MerchantName         string        `json:"merchant_name,omitempty"`
	MerchantCategoryCode string        `json:"merchant_category_code,omitempty"`
	AuthorizationMethod  string        `json:"authorization_method,omitempty"`
	InternalTransferType string        `json:"internal_transfer_type,omitempty"`
	Website              string        `json:"website,omitempty"`
	Address              ColumnAddress `json:"address,omitempty"`
}

type TransferResponse struct {
	ID                      string                 `json:"id"`
	CreatedAt               string                 `json:"created_at"`
	UpdatedAt               string                 `json:"updated_at"`
	IdempotencyKey          string                 `json:"idempotency_key,omitempty"`
	SenderBankAccountID     string                 `json:"sender_bank_account_id,omitempty"`
	SenderAccountNumberID   string                 `json:"sender_account_number_id,omitempty"`
	ReceiverBankAccountID   string                 `json:"receiver_bank_account_id,omitempty"`
	ReceiverAccountNumberID string                 `json:"receiver_account_number_id,omitempty"`
	Amount                  int64                  `json:"amount"`
	CurrencyCode            string                 `json:"currency_code"`
	Description             string                 `json:"description,omitempty"`
	Status                  string                 `json:"status"`
	AllowOverdraft          bool                   `json:"allow_overdraft,omitempty"`
	Details                 map[string]interface{} `json:"details,omitempty"`
}

func (c *client) InitiateTransfer(ctx context.Context, transferRequest *TransferRequest) (*TransferResponse, error) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_internal_transfer")

	body, err := json.Marshal(transferRequest)
	if err != nil {
		return &TransferResponse{}, fmt.Errorf("failed to marshal transfer request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, "transfers/book", bytes.NewBuffer(body))
	if err != nil {
		return &TransferResponse{}, fmt.Errorf("failed to create transfer request: %w", err)
	}

	var response TransferResponse
	var errRes columnError
	if _, err := c.httpClient.Do(ctx, req, &response, &errRes); err != nil {
		return &TransferResponse{}, fmt.Errorf("failed to create transfer: %w %w", err, errRes.Error())
	}

	return &response, nil
}
