package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type PayoutRequest struct {
	Type                string               `json:"type"`
	ActingTeamMember    string               `json:"acting_team_member,omitempty"`
	PayToCompany        string               `json:"pay_to_company"`
	WithdrawFromAccount string               `json:"withdraw_from_account"`
	CurrencyCode        string               `json:"currency_code"`
	Amount              string               `json:"amount"`
	SendOn              *string              `json:"send_on"`
	DeliveryMethod      string               `json:"delivery_method,omitempty"`
	PayToPaymentMethod  string               `json:"pay_to_payment_method,omitempty"`
	LineItems           []NewPayableLineItem `json:"line_items"`
	TypeDetails         map[string]any       `json:"type_details,omitempty"`
}

type NewPayableLineItem struct {
	UnitPrice   string `json:"unit_price"`
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	Amount      string `json:"amount"`
}

type PayoutResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type payablesCreateResponse struct {
	ID        string      `json:"id"`
	Status    interface{} `json:"status"`
	CreatedAt string      `json:"created_at"`
}

type problemError struct {
	Where  string `json:"where"`
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

type problem struct {
	Title     string         `json:"title"`
	Status    int            `json:"status"`
	RequestID string         `json:"request_id"`
	Detail    string         `json:"detail"`
	Errors    []problemError `json:"errors"`
}

func (p *problem) Error() string {
	return p.Detail
}

func (c *client) InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	body, _ := json.Marshal(pr)
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/payables", bytesReader(body))
	if err != nil {
		return nil, err
	}

	var out payablesCreateResponse
	var perr problem
	status, err := c.httpClient.Do(ctx, req, &out, &perr)
	if err != nil {
		if perr.Status != 0 || perr.Detail != "" || perr.Title != "" {
			_ = perr // ensure non-nil capture
			return nil, fmt.Errorf("%w: title=%s status=%d request_id=%s detail=%s errors=%v", err, perr.Title, perr.Status, perr.RequestID, perr.Detail, perr.Errors)
		}
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusAccepted {
		return nil, fmt.Errorf("title=%s status=%d request_id=%s detail=%s errors=%v", perr.Title, perr.Status, perr.RequestID, perr.Detail, perr.Errors)
	}

	var s string
	switch v := out.Status.(type) {
	case string:
		s = v
	case float64:
		s = strconv.Itoa(int(v))
	default:
		s = ""
	}

	var createdAt time.Time
	if out.CreatedAt != "" {
		createdAt, _ = time.Parse(time.RFC3339, out.CreatedAt)
	}
	return &PayoutResponse{ID: out.ID, Status: s, CreatedAt: createdAt}, nil
}

// bytesReader wraps a byte slice into an io.Reader
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
