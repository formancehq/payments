package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"github.com/plaid/plaid-go/v34/plaid"
)

type UpdateLinkTokenRequest struct {
	UserName    string
	UserID      string
	UserToken   string
	Language    string
	CountryCode string
	RedirectURI string
	AccessToken string
	ItemID      string
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
		req.UserName,
		req.Language,
		[]plaid.CountryCode{countryCode},
		plaid.LinkTokenCreateRequestUser{
			ClientUserId: req.UserID,
		},
	)

	update := plaid.NewLinkTokenCreateRequestUpdate()
	update.SetUser(true)
	update.SetItemIds([]string{req.ItemID})
	request.SetUpdate(*update)

	request.SetAccessToken(req.AccessToken)
	request.SetUserToken(req.UserToken)
	request.SetEnableMultiItemLink(true)
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
		return UpdateLinkTokenResponse{}, wrapSDKError(err)
	}

	return UpdateLinkTokenResponse{
		LinkToken:     resp.GetLinkToken(),
		Expiration:    resp.GetExpiration(),
		RequestID:     resp.GetRequestId(),
		HostedLinkUrl: resp.GetHostedLinkUrl(),
	}, nil
}
