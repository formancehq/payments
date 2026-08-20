package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
)

const (
	LinkTokenQueryParamID   = "link_token"
	PublicTokenQueryParamID = "public_token"
	StateQueryParamID       = "state"
)

type FormanceOpenBankingRedirectRequest struct {
	// RedirectURL is the caller-supplied destination to notify that the Link
	// session finished (the same URL passed as FormanceRedirectURL when the
	// link/update-link token was created). It's empty for a Link session
	// created by a version of this plugin that predates FormanceRedirectURL
	// - its registered webhook URL has no way to carry it, since Plaid fixes
	// that URL at link-token-creation time and a stale link/hosted-link
	// token can still complete after this plugin is redeployed. In that case
	// FormanceOpenBankingRedirect falls back to the connectorID/
	// STACK_PUBLIC_URL construction this package used before RedirectURL
	// existed, so those older sessions still resolve rather than being
	// dropped.
	RedirectURL string
	LinkToken   string
	PublicToken string
	AttemptID   uuid.UUID
}

func (c *client) FormanceOpenBankingRedirect(ctx context.Context, req FormanceOpenBankingRedirectRequest) error {
	redirectURL := req.RedirectURL
	if redirectURL == "" {
		var err error
		redirectURL, err = url.JoinPath(c.formanceStackEndpoint, "connectors", "open-banking", c.connectorID, "redirect")
		if err != nil {
			return err
		}
	}

	u, err := url.Parse(redirectURL)
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
