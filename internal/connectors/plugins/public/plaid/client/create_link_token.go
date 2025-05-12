package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"github.com/plaid/plaid-go/v34/plaid"
)

type CreateLinkTokenRequest struct {
	UserName    string
	UserID      string
	Language    string
	CountryCode string
	RedirectURI string
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
		return CreateLinkTokenResponse{}, fmt.Errorf("invalid plaid country code: %s: %w", req.CountryCode, models.ErrInvalidRequest)
	}

	request := plaid.NewLinkTokenCreateRequest(
		req.UserName,
		req.Language,
		[]plaid.CountryCode{countryCode},
		plaid.LinkTokenCreateRequestUser{
			ClientUserId: req.UserID,
		},
	)

	request.Products = []plaid.Products{plaid.PRODUCTS_TRANSACTIONS, plaid.PRODUCTS_TRANSACTIONS_REFRESH}
	request.SetRedirectUri(req.RedirectURI)
	hostedLink := plaid.NewLinkTokenCreateHostedLink()
	hostedLink.SetCompletionRedirectUri(req.RedirectURI)
	request.SetHostedLink(*hostedLink)
	request.SetAccountFilters(plaid.LinkTokenAccountFilters{
		Depository: &plaid.DepositoryFilter{
			AccountSubtypes: []plaid.DepositoryAccountSubtype{plaid.DEPOSITORYACCOUNTSUBTYPE_ALL},
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
