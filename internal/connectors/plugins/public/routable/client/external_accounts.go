package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type ExternalAccount struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

type companiesList struct {
	Object  string            `json:"object"`
	Results []ExternalAccount `json:"results"`
}

func (c *client) GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_companies")

	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("page_size", strconv.Itoa(pageSize))
	}
	path := "/v1/companies"
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var out companiesList
	var perr problem
	if _, err := c.httpClient.Do(ctx, req, &out, &perr); err != nil {
		return nil, fmt.Errorf("%w: title=%s status=%d request_id=%s detail=%s errors=%v", err, perr.Title, perr.Status, perr.RequestID, perr.Detail, perr.Errors)
	}

	res := make([]*ExternalAccount, 0, len(out.Results))
	for i := range out.Results {
		e := out.Results[i]
		res = append(res, &e)
	}
	return res, nil
}

// Bank payment method creation

type CreateBankPaymentMethodRequest struct {
	Type        string                         `json:"type"` // "bank"
	TypeDetails CreateBankPaymentMethodDetails `json:"type_details"`
}

type CreateBankPaymentMethodDetails struct {
	AccountType   string `json:"account_type,omitempty"` // checking|savings
	AccountNumber string `json:"account_number,omitempty"`
	RoutingNumber string `json:"routing_number,omitempty"`
	Iban          string `json:"iban,omitempty"`
}

type PaymentMethod struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func (c *client) CreateBankPaymentMethod(ctx context.Context, companyID string, reqBody *CreateBankPaymentMethodRequest) (*PaymentMethod, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_payment_method")
	b, _ := json.Marshal(reqBody)
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/companies/%s/payment-methods", companyID), bytesReader(b))
	if err != nil {
		return nil, err
	}
	var out PaymentMethod
	var perr problem
	status, err := c.httpClient.Do(ctx, req, &out, &perr)
	if err != nil {
		return nil, fmt.Errorf("%w: title=%s status=%d request_id=%s detail=%s errors=%v", err, perr.Title, perr.Status, perr.RequestID, perr.Detail, perr.Errors)
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return nil, fmt.Errorf("title=%s status=%d request_id=%s detail=%s errors=%v", perr.Title, perr.Status, perr.RequestID, perr.Detail, perr.Errors)
	}
	return &out, nil
}
