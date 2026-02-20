package client

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/plaid/plaid-go/v34/plaid"
)

const (
	AttemptIDQueryParamID = "attemptID"
)

type CreateLinkTokenRequest struct {
	ApplicationName string
	UserID          string
	UserToken       string
	Language        string
	CountryCode     string
	RedirectURI     string
	WebhookBaseURL  string
	AttemptID       string
}

type CreateLinkTokenResponse struct {
	LinkToken     string
	Expiration    time.Time
	RequestID     string
	HostedLinkUrl string
}

func (c *client) CreateLinkToken(ctx context.Context, req CreateLinkTokenRequest) (CreateLinkTokenResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_link_token")

	countryCode := plaid.CountryCode(req.CountryCode)
	if !countryCode.IsValid() {
		return CreateLinkTokenResponse{}, fmt.Errorf("invalid plaid country code: %s: %w", req.CountryCode, connector.ErrInvalidRequest)
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
		return CreateLinkTokenResponse{}, fmt.Errorf("invalid webhook base URL: %w", err)
	}

	url = url.JoinPath("all")
	query := url.Query()
	query.Set(AttemptIDQueryParamID, req.AttemptID)
	url.RawQuery = query.Encode()

	webhookURL := url.String()

	request.SetUserToken(req.UserToken)
	request.SetEnableMultiItemLink(true)
	request.SetWebhook(webhookURL)
	request.Products = []plaid.Products{plaid.PRODUCTS_TRANSACTIONS}
	request.SetRedirectUri(req.RedirectURI)
	hostedLink := plaid.NewLinkTokenCreateHostedLink()
	hostedLink.SetCompletionRedirectUri(req.RedirectURI)
	request.SetHostedLink(*hostedLink)
	request.SetAccountFilters(plaid.LinkTokenAccountFilters{
		Depository: &plaid.DepositoryFilter{
			AccountSubtypes: []plaid.DepositoryAccountSubtype{plaid.DEPOSITORYACCOUNTSUBTYPE_CHECKING, plaid.DEPOSITORYACCOUNTSUBTYPE_SAVINGS},
		},
	})

	resp, _, err := c.client.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()
	if err != nil {
		return CreateLinkTokenResponse{}, wrapSDKError(err)
	}

	return CreateLinkTokenResponse{
		LinkToken:     resp.GetLinkToken(),
		Expiration:    resp.GetExpiration(),
		RequestID:     resp.GetRequestId(),
		HostedLinkUrl: resp.GetHostedLinkUrl(),
	}, nil
}
