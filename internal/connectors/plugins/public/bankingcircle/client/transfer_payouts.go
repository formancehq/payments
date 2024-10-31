package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
)

type PaymentAccount struct {
	Account              string `json:"account"`
	FinancialInstitution string `json:"financialInstitution"`
	Country              string `json:"country"`
}

type PaymentRequest struct {
	IdempotencyKey         string          `json:"idempotencyKey"`
	RequestedExecutionDate time.Time       `json:"requestedExecutionDate"`
	DebtorAccount          PaymentAccount  `json:"debtorAccount"`
	DebtorReference        string          `json:"debtorReference"`
	CurrencyOfTransfer     string          `json:"currencyOfTransfer"`
	Amount                 Amount          `json:"amount"`
	ChargeBearer           string          `json:"chargeBearer"`
	CreditorAccount        *PaymentAccount `json:"creditorAccount"`
}

type PaymentResponse struct {
	PaymentID string `json:"paymentId"`
	Status    string `json:"status"`
}

func (c *client) InitiateTransferOrPayouts(ctx context.Context, transferRequest *PaymentRequest) (*PaymentResponse, error) {
	if err := c.ensureAccessTokenIsValid(ctx); err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "create_transfers_payouts")

	body, err := json.Marshal(transferRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/v1/payments/singles", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create payments request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	var res PaymentResponse
	statusCode, err := c.httpClient.Do(ctx, req, &res, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make payout, status code %d: %w", statusCode, err)
	}
	return &res, nil
}
