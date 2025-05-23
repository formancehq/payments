package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

const (
	LinkTokenQueryParamID   = "link_token"
	PublicTokenQueryParamID = "public_token"
)

type FormanceBankBridgeRedirectRequest struct {
	LinkToken   string
	PublicToken string
}

func (c *client) FormanceBankBridgeRedirect(ctx context.Context, req FormanceBankBridgeRedirectRequest) error {
	u, err := url.Parse(fmt.Sprintf("http://localhost:8080/v3/connectors/bank-bridges/%s/redirect", c.connectorID.String()))
	if err != nil {
		return err
	}

	q := u.Query()
	q.Set(LinkTokenQueryParamID, req.LinkToken)
	q.Set(PublicTokenQueryParamID, req.PublicToken)
	u.RawQuery = q.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	_, err = c.formanceHTTPClient.Do(ctx, request, nil, nil)
	return err
}
