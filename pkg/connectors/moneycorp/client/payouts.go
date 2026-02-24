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

type payoutRequest struct {
	Payout struct {
		Attributes *PayoutRequest `json:"attributes"`
	} `json:"data"`
}

type PayoutRequest struct {
	SourceAccountID  string      `json:"-"`
	IdempotencyKey   string      `json:"-"`
	RecipientID      string      `json:"recipientId"`
	PaymentDate      string      `json:"paymentDate"`
	PaymentAmount    json.Number `json:"paymentAmount"`
	PaymentCurrency  string      `json:"paymentCurrency"`
	PaymentMethod    string      `json:"paymentMethod"`
	PaymentReference string      `json:"paymentReference"`
	ClientReference  string      `json:"clientReference"`
	PaymentPurpose   string      `json:"paymentPurpose"`
}

type payoutResponse struct {
	Payout *PayoutResponse `json:"data"`
}

type RecipientDetails struct {
	RecipientID int64 `json:"recipientId"`
}

type PayoutAttributes struct {
	AccountID        int64            `json:"accountId"`
	PaymentAmount    json.Number      `json:"paymentAmount"`
	PaymentCurrency  string           `json:"paymentCurrency"`
	PaymentApproved  bool             `json:"paymentApproved"`
	PaymentStatus    string           `json:"paymentStatus"`
	PaymentMethod    string           `json:"paymentMethod"`
	PaymentDate      string           `json:"paymentDate"`
	PaymentValueDate string           `json:"paymentValueDate"`
	RecipientDetails RecipientDetails `json:"recipientDetails"`
	PaymentReference string           `json:"paymentReference"`
	ClientReference  string           `json:"clientReference"`
	CreatedAt        string           `json:"createdAt"`
	CreatedBy        string           `json:"createdBy"`
	UpdatedAt        string           `json:"updatedAt"`
	PaymentPurpose   string           `json:"paymentPurpose"`
}

type PayoutResponse struct {
	ID         string           `json:"id"`
	Attributes PayoutAttributes `json:"attributes"`
}

func (c *client) InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	endpoint := fmt.Sprintf("%s/accounts/%s/payments", c.endpoint, pr.SourceAccountID)

	reqBody := &payoutRequest{}
	reqBody.Payout.Attributes = pr
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payout request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", pr.IdempotencyKey)

	var res payoutResponse
	var errRes moneycorpErrors
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to initiate payout: %v", errRes.Error()),
			err,
		)
	}

	return res.Payout, nil
}
