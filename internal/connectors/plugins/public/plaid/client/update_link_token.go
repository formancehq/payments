package client

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"github.com/plaid/plaid-go/v34/plaid"
)

type UpdateLinkTokenRequest struct {
	ApplicationName string
	AttemptID       string
	UserID          string
	UserToken       string
	Language        string
	CountryCode     string
	RedirectURI     string
	AccessToken     string
	ItemID          string
	WebhookBaseURL  string
}

type UpdateLinkTokenResponse struct {
	LinkToken     string
	Expiration    time.Time
	RequestID     string
	HostedLinkUrl string
}

func (c *client) UpdateLinkToken(ctx context.Context, req UpdateLinkTokenRequest) (UpdateLinkTokenResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_link_token")

	countryCode := plaid.CountryCode(req.CountryCode)
	if !countryCode.IsValid() {
		return UpdateLinkTokenResponse{}, fmt.Errorf("invalid plaid country code: %s: %w", req.CountryCode, models.ErrInvalidRequest)
	}

	request := plaid.NewLinkTokenCreateRequest(
		req.ApplicationName,
		req.Language,
		[]plaid.CountryCode{countryCode},
		plaid.LinkTokenCreateRequestUser{
			ClientUserId: req.UserID,
		},
	)

	url, err := url.Parse(req.WebhookBaseURL)
	if err != nil {
		return UpdateLinkTokenResponse{}, fmt.Errorf("invalid webhook base URL: %w", err)
	}

	url = url.JoinPath("all")
	query := url.Query()
	query.Set(AttemptIDQueryParamID, req.AttemptID)
	url.RawQuery = query.Encode()

	webhookURL := url.String()

	update := plaid.NewLinkTokenCreateRequestUpdate()
	update.SetUser(false)
	// update.SetItemIds([]string{req.ItemID})
	request.SetUpdate(*update)

	request.SetWebhook(webhookURL)
	request.SetAccessToken(req.AccessToken)
	// request.SetUserToken(req.UserToken)
	request.SetRedirectUri(req.RedirectURI)
	hostedLink := plaid.NewLinkTokenCreateHostedLink()
	hostedLink.SetCompletionRedirectUri(req.RedirectURI)
	request.SetHostedLink(*hostedLink)

	resp, _, err := c.client.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()
	if err != nil {
		return UpdateLinkTokenResponse{}, wrapSDKError(err)
	}

	return UpdateLinkTokenResponse{
		LinkToken:     resp.GetLinkToken(),
		Expiration:    resp.GetExpiration(),
		RequestID:     resp.GetRequestId(),
		HostedLinkUrl: resp.GetHostedLinkUrl(),
	}, nil
}
