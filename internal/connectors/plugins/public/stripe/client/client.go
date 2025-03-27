package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/account"
	"github.com/stripe/stripe-go/v79/balance"
	"github.com/stripe/stripe-go/v79/balancetransaction"
	"github.com/stripe/stripe-go/v79/bankaccount"
	"github.com/stripe/stripe-go/v79/payout"
	"github.com/stripe/stripe-go/v79/transfer"
	"github.com/stripe/stripe-go/v79/transferreversal"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, timeline Timeline, pageSize int64) ([]*stripe.Account, Timeline, bool, error)
	GetAccountBalances(ctx context.Context, accountID string) (*stripe.Balance, error)
	GetExternalAccounts(ctx context.Context, accountID string, timeline Timeline, pageSize int64) ([]*stripe.BankAccount, Timeline, bool, error)
	GetPayments(ctx context.Context, accountID string, timeline Timeline, pageSize int64) ([]*stripe.BalanceTransaction, Timeline, bool, error)
	CreatePayout(ctx context.Context, createPayoutRequest *CreatePayoutRequest) (*stripe.Payout, error)
	CreateTransfer(ctx context.Context, createTransferRequest *CreateTransferRequest) (*stripe.Transfer, error)
	ReverseTransfer(ctx context.Context, reverseTransferRequest ReverseTransferRequest) (*stripe.TransferReversal, error)
}

type client struct {
	accountClient            account.Client
	balanceClient            balance.Client
	transferClient           transfer.Client
	transferReversalClient   transferreversal.Client
	payoutClient             payout.Client
	bankAccountClient        bankaccount.Client
	balanceTransactionClient balancetransaction.Client
}

func New(name string, backend stripe.Backend, apiKey string) Client {
	if backend == nil {
		backends := stripe.NewBackends(metrics.NewHTTPClient(name, models.DefaultConnectorClientTimeout))
		backend = backends.API
	}

	return &client{
		accountClient:            account.Client{B: backend, Key: apiKey},
		balanceClient:            balance.Client{B: backend, Key: apiKey},
		transferClient:           transfer.Client{B: backend, Key: apiKey},
		payoutClient:             payout.Client{B: backend, Key: apiKey},
		bankAccountClient:        bankaccount.Client{B: backend, Key: apiKey},
		balanceTransactionClient: balancetransaction.Client{B: backend, Key: apiKey},
	}
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
		return errorsutils.NewWrappedError(
			err,
			httpwrapper.ErrStatusCodeTooManyRequests,
		)
	}

	switch stripeErr.Type {
	case stripe.ErrorTypeInvalidRequest, stripe.ErrorTypeIdempotency:
		return errorsutils.NewWrappedError(
			err,
			httpwrapper.ErrStatusCodeClientError,
		)
	}
	return err
}
