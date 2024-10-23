package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type PaymentType string

const (
	PAYIN  PaymentType = "PAYIN"
	PAYOUT PaymentType = "PAYOUT"
)

type Payment struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	CreatedDate       string `json:"createdDate"`
	ExternalReference string `json:"externalReference"`
	ApprovalStatus    string `json:"approvalStatus"`
	CreatedBy         string `json:"createdBy"`
	Type              string `json:"type"`
	Details           struct {
		AccountNumber   string `json:"accountNumber"`
		SourceAccountID string `json:"sourceAccountId"`
		Destination     struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"destination"`
		Amount   json.Number `json:"amount"`
		Currency string      `json:"currency"`
	} `json:"details"`
}

func (c *client) GetPayments(ctx context.Context, paymentType PaymentType, page, pageSize int, modifiedSince time.Time) ([]Payment, error) {
	// TODO(polo): add metrics
	// f := connectors.ClientMetrics(ctx, "modulr", "list_payments")
	// now := time.Now()
	// defer f(ctx, now)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("payments"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create accounts request: %w", err)
	}

	q := req.URL.Query()
	q.Add("page", strconv.Itoa(page))
	q.Add("size", strconv.Itoa(pageSize))
	q.Add("type", string(paymentType))
	q.Add("sortOrder", "asc")
	if !modifiedSince.IsZero() {
		q.Add("modifiedSince", modifiedSince.Format("2006-01-02T15:04:05-0700"))
	}
	req.URL.RawQuery = q.Encode()

	var res responseWrapper[[]Payment]
	var errRes modulrError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get payments: %w %w", err, errRes.Error())
	}
	return res.Content, nil
}
