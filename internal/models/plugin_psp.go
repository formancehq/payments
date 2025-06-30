package models

import (
	"context"
	"encoding/json"
)

type PSPPlugin interface {
	FetchNextAccounts(context.Context, FetchNextAccountsRequest) (FetchNextAccountsResponse, error)
	FetchNextPayments(context.Context, FetchNextPaymentsRequest) (FetchNextPaymentsResponse, error)
	FetchNextBalances(context.Context, FetchNextBalancesRequest) (FetchNextBalancesResponse, error)
	FetchNextExternalAccounts(context.Context, FetchNextExternalAccountsRequest) (FetchNextExternalAccountsResponse, error)
	FetchNextOthers(context.Context, FetchNextOthersRequest) (FetchNextOthersResponse, error)

	CreateBankAccount(context.Context, CreateBankAccountRequest) (CreateBankAccountResponse, error)
	CreateTransfer(context.Context, CreateTransferRequest) (CreateTransferResponse, error)
	ReverseTransfer(context.Context, ReverseTransferRequest) (ReverseTransferResponse, error)
	PollTransferStatus(context.Context, PollTransferStatusRequest) (PollTransferStatusResponse, error)
	CreatePayout(context.Context, CreatePayoutRequest) (CreatePayoutResponse, error)
	ReversePayout(context.Context, ReversePayoutRequest) (ReversePayoutResponse, error)
	PollPayoutStatus(context.Context, PollPayoutStatusRequest) (PollPayoutStatusResponse, error)
}

type FetchNextAccountsRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextAccountsResponse struct {
	Accounts []PSPAccount
	NewState json.RawMessage
	HasMore  bool
}

type FetchNextExternalAccountsRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextExternalAccountsResponse struct {
	ExternalAccounts []PSPAccount
	NewState         json.RawMessage
	HasMore          bool
}

type FetchNextPaymentsRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextPaymentsResponse struct {
	Payments         []PSPPayment
	PaymentsToDelete []PSPPayment
	NewState         json.RawMessage
	HasMore          bool
}

type FetchNextOthersRequest struct {
	Name        string
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextOthersResponse struct {
	Others   []PSPOther
	NewState json.RawMessage
	HasMore  bool
}

type FetchNextBalancesRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextBalancesResponse struct {
	Balances []PSPBalance
	NewState json.RawMessage
	HasMore  bool
}

type CreateBankAccountRequest struct {
	BankAccount BankAccount
}

type CreateBankAccountResponse struct {
	RelatedAccount PSPAccount
}

type CreateTransferRequest struct {
	PaymentInitiation PSPPaymentInitiation
}

type CreateTransferResponse struct {
	// If payment is immediately available, it will be return here and
	// the workflow will be terminated
	Payment *PSPPayment
	// Otherwise, the payment will be nil and the transfer ID will be returned
	// to be polled regularly until the payment is available
	PollingTransferID *string
}

type ReverseTransferRequest struct {
	PaymentInitiationReversal PSPPaymentInitiationReversal
}
type ReverseTransferResponse struct {
	Payment PSPPayment
}

type PollTransferStatusRequest struct {
	TransferID string
}

type PollTransferStatusResponse struct {
	// If nil, the payment is not yet available and the function will be called
	// again later
	// If not, the payment is available and the workflow will be terminated
	Payment *PSPPayment

	// If not nil, it means that the transfer failed, the payment initiation
	// will be marked as fail and the workflow will be terminated
	Error *string
}

type CreatePayoutRequest struct {
	PaymentInitiation PSPPaymentInitiation
}

type CreatePayoutResponse struct {
	// If payment is immediately available, it will be return here and
	// the workflow will be terminated
	Payment *PSPPayment
	// Otherwise, the payment will be nil and the payout ID will be returned
	// to be polled regularly until the payment is available
	PollingPayoutID *string
}

type ReversePayoutRequest struct {
	PaymentInitiationReversal PSPPaymentInitiationReversal
}
type ReversePayoutResponse struct {
	Payment PSPPayment
}

type PollPayoutStatusRequest struct {
	PayoutID string
}

type PollPayoutStatusResponse struct {
	// If nil, the payment is not yet available and the function will be called
	// again later
	// If not, the payment is available and the workflow will be terminated
	Payment *PSPPayment

	// If not nil, it means that the payout failed, the payment initiation
	// will be marked as fail and the workflow will be terminated
	Error *string
}
