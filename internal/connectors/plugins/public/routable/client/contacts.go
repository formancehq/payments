package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type CreateContactRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	// actionable | read_only | none | self_managed
	DefaultForPayables string `json:"default_contact_for_payable_and_receivable"`
	ActingTeamMember   string `json:"acting_team_member,omitempty"`
}

type Contact struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

func (c *client) CreateContact(ctx context.Context, companyID string, reqBody *CreateContactRequest) (*Contact, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_contact")
	b, _ := json.Marshal(reqBody)
	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/companies/%s/contacts", companyID), bytesReader(b))
	if err != nil {
		return nil, err
	}
	var out Contact
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
