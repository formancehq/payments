package client

import (
	"context"
	"crypto/tls"
	"net/http"

	cbclient "github.com/coinbase-samples/prime-sdk-go/client"
	cbcreds "github.com/coinbase-samples/prime-sdk-go/credentials"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context, accountRef string) ([]*Balance, error)
	GetWalletBalance(ctx context.Context, walletId string) ([]*Balance, error)
	GetPortfolioTransactions(ctx context.Context, portfolioId string, page, pageSize int) ([]*Transaction, error)
	GetWalletTransactions(ctx context.Context, portfolioId string, walletId string, page, pageSize int) ([]*Transaction, error)
	GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error)
}

type Config struct {
	Credentials string
	// Dev-only: allow disabling TLS verification when testing locally.
	InsecureSkipVerify bool
}

type client struct {
	httpClient httpwrapper.Client

	// TODO: fill config parameters
	// You may need fields here for authentication purpose
	credentials string

	// sdkHTTP is the instrumented HTTP client passed to the Coinbase SDK.
	sdkHTTP *http.Client
	// sdk is the Coinbase Prime REST client used by services.
	sdk cbclient.RestClient
}

func New(cfg Config) *client {
	config := &httpwrapper.Config{
		// TODO: you can set an underlying http transport in metrics.TransportOpts for authentication for example
		Transport: metrics.NewTransport("coinbaseprime", metrics.TransportOpts{}),
		// TODO: if the PSP requires special http status code handling, you can override the default handling by setting a
		// custom HttpErrorCheckerFn like below
		// HttpErrorCheckerFn: func(statusCode int) error {
		// 	if statusCode == http.StatusNotFound {
		// 		return nil
		// 	} else if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		// 		return httpwrapper.ErrStatusCodeClientError
		// 	} else if statusCode >= http.StatusInternalServerError {
		// 		return httpwrapper.ErrStatusCodeServerError
		// 	}
		// 	return nil
		// },
	}

	// Prepare an instrumented HTTP client for the Coinbase SDK
	baseHTTP, err := cbclient.DefaultHttpClient()
	if err != nil {
		baseHTTP = http.Client{}
	}
	parent := baseHTTP.Transport
	if parent == nil {
		parent = http.DefaultTransport
	}
	// Optional: allow insecure TLS for local dev only
	if cfg.InsecureSkipVerify {
		if tp, ok := parent.(*http.Transport); ok {
			clone := tp.Clone()
			if clone.TLSClientConfig == nil {
				clone.TLSClientConfig = &tls.Config{}
			}
			clone.TLSClientConfig.InsecureSkipVerify = true
			parent = clone
		} else {
			parent = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		}
	}
	baseHTTP.Transport = metrics.NewTransport("coinbaseprime", metrics.TransportOpts{Transport: parent})

	// Parse credentials blob for the SDK
	creds, _ := cbcreds.UnmarshalCredentials([]byte(cfg.Credentials))
	// Build SDK REST client
	sdk := cbclient.NewRestClient(creds, baseHTTP)

	return &client{
		httpClient:  httpwrapper.NewClient(config),
		credentials: cfg.Credentials,
		sdkHTTP:     &baseHTTP,
		sdk:         sdk,
	}
}
