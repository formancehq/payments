package client

import (
	"context"
	"fmt"
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

type FormanceBankBridgeRedirectRequest struct {
	LinkToken   string
	PublicToken string
	AttemptID   uuid.UUID
}

func (c *client) FormanceBankBridgeRedirect(ctx context.Context, req FormanceBankBridgeRedirectRequest) error {
	u, err := url.Parse(fmt.Sprintf("http://localhost:8080/v3/connectors/bank-bridges/%s/redirect", c.connectorID.String()))
	if err != nil {
		return err
	}

	q := u.Query()
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
