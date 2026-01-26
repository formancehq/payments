package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type CreateCompanyRequest struct {
	Type             string  `json:"type"` // "business" or "personal"
	Name             *string `json:"name,omitempty"`
	BusinessName     *string `json:"business_name,omitempty"`
	ActingTeamMember string  `json:"acting_team_member,omitempty"`
	IsCustomer       bool    `json:"is_customer"`
	IsVendor         bool    `json:"is_vendor"`
}

type Company struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

func (c *client) CreateCompany(ctx context.Context, reqBody *CreateCompanyRequest) (*Company, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_company")
	b, _ := json.Marshal(reqBody)
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/companies", bytesReader(b))
	if err != nil {
		return nil, err
	}
	var out Company
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
