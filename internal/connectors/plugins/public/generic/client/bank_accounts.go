package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

type BankAccountRequest struct {
	Name          string            `json:"name"`
	AccountNumber *string           `json:"accountNumber,omitempty"`
	IBAN          *string           `json:"iban,omitempty"`
	SwiftBicCode  *string           `json:"swiftBicCode,omitempty"`
	Country       *string           `json:"country,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type BankAccountResponse struct {
	Id            string            `json:"id"`
	Name          string            `json:"name"`
	AccountNumber *string           `json:"accountNumber,omitempty"`
	IBAN          *string           `json:"iban,omitempty"`
	SwiftBicCode  *string           `json:"swiftBicCode,omitempty"`
	Country       *string           `json:"country,omitempty"`
	CreatedAt     string            `json:"createdAt"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

func (c *client) CreateBankAccount(ctx context.Context, request *BankAccountRequest) (*BankAccountResponse, error) {
	ctx = metrics.OperationContext(ctx, "create_bank_account")

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bank account request: %w", err)
	}

	baseURL := c.apiClient.GetConfig().Servers[0].URL
	url := fmt.Sprintf("%s/bank-accounts", baseURL)

	logging.FromContext(ctx).Debugf("Creating bank account: POST %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create bank account request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.apiClient.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bank account request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read bank account response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bank account request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var bankAccountResp BankAccountResponse
	if err := json.Unmarshal(respBody, &bankAccountResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bank account response: %w", err)
	}

	logging.FromContext(ctx).Debugf("Bank account created: %s", bankAccountResp.Id)

	return &bankAccountResp, nil
}
