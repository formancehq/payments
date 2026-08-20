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
	// link/update-link token was created) - this package builds no part of
	// it itself, so it stays usable outside the payments connector engine.
	RedirectURL string
	LinkToken   string
	PublicToken string
	AttemptID   uuid.UUID
}

func (c *client) FormanceOpenBankingRedirect(ctx context.Context, req FormanceOpenBankingRedirectRequest) error {
	u, err := url.Parse(req.RedirectURL)
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
