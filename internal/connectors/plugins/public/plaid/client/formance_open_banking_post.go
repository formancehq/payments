package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

const (
	LinkTokenQueryParamID   = "link_token"
	PublicTokenQueryParamID = "public_token"
	StateQueryParamID       = "state"
)

type FormanceOpenBankingRedirectRequest struct {
	LinkToken   string
	PublicToken string
	AttemptID   uuid.UUID
}

func (c *client) FormanceOpenBankingRedirect(ctx context.Context, req FormanceOpenBankingRedirectRequest) error {
	endpoint, err := url.JoinPath(c.formanceStackEndpoint, "connectors", "open-banking", c.connectorID.String(), "redirect")
	if err != nil {
		return err
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set(models.NoRedirectQueryParamID, "true")
	q.Set(LinkTokenQueryParamID, req.LinkToken)
	q.Set(PublicTokenQueryParamID, req.PublicToken)
	q.Set(StateQueryParamID, models.CallbackState{
		AttemptID: req.AttemptID,
	}.String())
	u.RawQuery = q.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	_, err = c.formanceHTTPClient.Do(ctx, request, nil, nil)
	return err
}
