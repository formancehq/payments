package client

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	atlar_client "github.com/get-momo/atlar-v1-go-client/client"
	"github.com/get-momo/atlar-v1-go-client/client/accounts"
	"github.com/get-momo/atlar-v1-go-client/client/counterparties"
	"github.com/get-momo/atlar-v1-go-client/client/credit_transfers"
	"github.com/get-momo/atlar-v1-go-client/client/external_accounts"
	"github.com/get-momo/atlar-v1-go-client/client/third_parties"
	"github.com/get-momo/atlar-v1-go-client/client/transactions"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetV1Accounts(ctx context.Context, token string, pageSize int64) (*accounts.GetV1AccountsOK, error)
	GetV1AccountsID(ctx context.Context, id string) (*accounts.GetV1AccountsIDOK, error)

	PostV1CounterParties(ctx context.Context, newExternalBankAccount models.BankAccount) (*counterparties.PostV1CounterpartiesCreated, error)
	GetV1CounterpartiesID(ctx context.Context, counterPartyID string) (*counterparties.GetV1CounterpartiesIDOK, error)

	GetV1ExternalAccounts(ctx context.Context, token string, pageSize int64) (*external_accounts.GetV1ExternalAccountsOK, error)
	GetV1ExternalAccountsID(ctx context.Context, externalAccountID string) (*external_accounts.GetV1ExternalAccountsIDOK, error)

	GetV1BetaThirdPartiesID(ctx context.Context, id string) (*third_parties.GetV1betaThirdPartiesIDOK, error)

	GetV1Transactions(ctx context.Context, token string, pageSize int64) (*transactions.GetV1TransactionsOK, error)
	GetV1TransactionsID(ctx context.Context, id string) (*transactions.GetV1TransactionsIDOK, error)

	PostV1CreditTransfers(ctx context.Context, req *atlar_models.CreatePaymentRequest) (*credit_transfers.PostV1CreditTransfersCreated, error)
	GetV1CreditTransfersGetByExternalIDExternalID(ctx context.Context, externalID string) (*credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDOK, error)
}

type client struct {
	client     *atlar_client.Rest
	httpClient *http.Client
}

func New(name string, baseURL *url.URL, accessKey, secret string) Client {
	return &client{
		client:     createAtlarClient(baseURL, accessKey, secret),
		httpClient: metrics.NewHTTPClient(name, models.DefaultConnectorClientTimeout),
	}
}

func createAtlarClient(baseURL *url.URL, accessKey, secret string) *atlar_client.Rest {
	transport := httptransport.New(
		baseURL.Host,
		baseURL.Path,
		[]string{baseURL.Scheme},
	)
	basicAuth := httptransport.BasicAuth(accessKey, secret)
	transport.DefaultAuthentication = basicAuth
	client := atlar_client.New(transport, strfmt.Default)
	return client
}

type ErrorCodeReader interface {
	Code() int
}

// wrap a public error for cases that we don't want to retry
// so that activities can classify this error for temporal
func wrapSDKErr(err error, atlarErr any) error {
	if err == nil {
		return nil
	}

	var code int
	switch {
	case atlarErr == nil:
		var atlarError *runtime.APIError
		if !errors.As(err, &atlarError) {
			return err
		}
		code = atlarError.Code

	default:
		if !errors.As(err, &atlarErr) {
			return err
		}

		reader, ok := atlarErr.(ErrorCodeReader)
		if !ok {
			return err
		}

		code = reader.Code()
	}

	if code == http.StatusTooManyRequests {
		return errorsutils.NewWrappedError(err, httpwrapper.ErrStatusCodeTooManyRequests)
	}

	if code >= http.StatusBadRequest && code < http.StatusInternalServerError {
		return errorsutils.NewWrappedError(err, httpwrapper.ErrStatusCodeClientError)
	} else if code >= http.StatusInternalServerError {
		return errorsutils.NewWrappedError(err, httpwrapper.ErrStatusCodeServerError)
	}

	return err
}
