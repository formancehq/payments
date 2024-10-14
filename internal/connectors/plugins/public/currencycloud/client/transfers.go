package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TransferRequest struct {
	SourceAccountID      string      `json:"source_account_id"`
	DestinationAccountID string      `json:"destination_account_id"`
	Currency             string      `json:"currency"`
	Amount               json.Number `json:"amount"`
	Reason               string      `json:"reason,omitempty"`
	UniqueRequestID      string      `json:"unique_request_id,omitempty"`
}

func (tr *TransferRequest) ToFormData() url.Values {
	form := url.Values{}
	form.Set("source_account_id", tr.SourceAccountID)
	form.Set("destination_account_id", tr.DestinationAccountID)
	form.Set("currency", tr.Currency)
	form.Set("amount", fmt.Sprintf("%v", tr.Amount))
	if tr.Reason != "" {
		form.Set("reason", tr.Reason)
	}
	if tr.UniqueRequestID != "" {
		form.Set("unique_request_id", tr.UniqueRequestID)
	}

	return form
}

type TransferResponse struct {
	ID                   string      `json:"id"`
	ShortReference       string      `json:"short_reference"`
	SourceAccountID      string      `json:"source_account_id"`
	DestinationAccountID string      `json:"destination_account_id"`
	Currency             string      `json:"currency"`
	Amount               json.Number `json:"amount"`
	Status               string      `json:"status"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
	CompletedAt          time.Time   `json:"completed_at"`
	CreatorAccountID     string      `json:"creator_account_id"`
	CreatorContactID     string      `json:"creator_contact_id"`
	Reason               string      `json:"reason"`
	UniqueRequestID      string      `json:"unique_request_id"`
}

func (c *client) InitiateTransfer(ctx context.Context, transferRequest *TransferRequest) (*TransferResponse, error) {
	// TODO(polo): metrics
	// f := connectors.ClientMetrics(ctx, "currencycloud", "initiate_transfer")
	// now := time.Now()
	// defer f(ctx, now)

	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}

	form := transferRequest.ToFormData()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.buildEndpoint("v2/transfers/create"), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var res TransferResponse
	var errRes currencyCloudError
	_, err = c.httpClient.Do(req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer: %w, %w", err, errRes.Error())
	}

	return &res, nil
}
