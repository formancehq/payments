package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connector/httpwrapper"
	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/account"
	"github.com/stripe/stripe-go/v80/balance"
	"github.com/stripe/stripe-go/v80/balancetransaction"
	"github.com/stripe/stripe-go/v80/bankaccount"
	"github.com/stripe/stripe-go/v80/payout"
	"github.com/stripe/stripe-go/v80/transfer"
	"github.com/stripe/stripe-go/v80/transferreversal"
	"github.com/stripe/stripe-go/v80/webhookendpoint"
)

// https://github.com/stripe/stripe-go/blob/master/stripe.go#L1478
const StripeDefaultTimeout = 80 * time.Second

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetRootAccountID() string
	GetRootAccount() (*stripe.Account, error)
	GetAccounts(ctx context.Context, timeline Timeline, pageSize int64) ([]*stripe.Account, Timeline, bool, error)
	GetAccountBalances(ctx context.Context, accountID string) (*stripe.Balance, error)
	GetExternalAccounts(ctx context.Context, accountID string, timeline Timeline, pageSize int64) ([]*stripe.BankAccount, Timeline, bool, error)
	GetPayments(ctx context.Context, accountID string, timeline Timeline, pageSize int64) ([]*stripe.BalanceTransaction, Timeline, bool, error)
	CreatePayout(ctx context.Context, createPayoutRequest *CreatePayoutRequest) (*stripe.Payout, error)
	CreateTransfer(ctx context.Context, createTransferRequest *CreateTransferRequest) (*stripe.Transfer, error)
	ReverseTransfer(ctx context.Context, reverseTransferRequest ReverseTransferRequest) (*stripe.TransferReversal, error)
	CreateWebhookEndpoints(ctx context.Context, webhookBaseURL string) ([]*stripe.WebhookEndpoint, error)
	DeleteWebhookEndpoints([]connector.PSPWebhookConfig) error
}

type client struct {
	logger logging.Logger

	rootAccountID string

	accountClient            account.Client
	balanceClient            balance.Client
	transferClient           transfer.Client
	transferReversalClient   transferreversal.Client
	payoutClient             payout.Client
	bankAccountClient        bankaccount.Client
	balanceTransactionClient balancetransaction.Client
	webhookEndpointClient    webhookendpoint.Client
}

func New(
	name string,
	logger logging.Logger,
	backend stripe.Backend,
	apiKey string,
) (Client, error) {
	if backend == nil {
		backends := stripe.NewBackends(metrics.NewHTTPClient(name, StripeDefaultTimeout))
		backend = backends.API
	}
	accountClient := account.Client{B: backend, Key: apiKey}
	result, err := accountClient.Get()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", connector.ErrInvalidClientRequest, err)
	}

	return &client{
		logger:        logger,
		rootAccountID: result.ID,

		accountClient:            accountClient,
		balanceClient:            balance.Client{B: backend, Key: apiKey},
		transferClient:           transfer.Client{B: backend, Key: apiKey},
		payoutClient:             payout.Client{B: backend, Key: apiKey},
		bankAccountClient:        bankaccount.Client{B: backend, Key: apiKey},
		balanceTransactionClient: balancetransaction.Client{B: backend, Key: apiKey},
		webhookEndpointClient:    webhookendpoint.Client{B: backend, Key: apiKey},
	}, nil
}

func (c *client) GetRootAccountID() string {
	return c.rootAccountID
}

func limit(wanted int64, have int) *int64 {
	needed := wanted - int64(have)
	return &needed
}

// wrap a public error for cases that we don't want to retry
// so that activities can classify this error for temporal
func wrapSDKErr(err error) error {
	if err == nil {
		return nil
	}

	stripeErr, ok := err.(*stripe.Error)
	if !ok {
		return err
	}

	if stripeErr.Code == stripe.ErrorCodeRateLimit {
		return connector.NewWrappedError(
			err,
			httpwrapper.ErrStatusCodeTooManyRequests,
		)
	}

	switch stripeErr.Type {
	case stripe.ErrorTypeInvalidRequest, stripe.ErrorTypeIdempotency:
		return connector.NewWrappedError(
			err,
			httpwrapper.ErrStatusCodeClientError,
		)
	}
	return err
}
