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
	FetchNextOrders(context.Context, FetchNextOrdersRequest) (FetchNextOrdersResponse, error)
	FetchNextConversions(context.Context, FetchNextConversionsRequest) (FetchNextConversionsResponse, error)

	CreateBankAccount(context.Context, CreateBankAccountRequest) (CreateBankAccountResponse, error)
	CreateTransfer(context.Context, CreateTransferRequest) (CreateTransferResponse, error)
	ReverseTransfer(context.Context, ReverseTransferRequest) (ReverseTransferResponse, error)
	PollTransferStatus(context.Context, PollTransferStatusRequest) (PollTransferStatusResponse, error)
	CreatePayout(context.Context, CreatePayoutRequest) (CreatePayoutResponse, error)
	ReversePayout(context.Context, ReversePayoutRequest) (ReversePayoutResponse, error)
	PollPayoutStatus(context.Context, PollPayoutStatusRequest) (PollPayoutStatusResponse, error)
	CreateOrder(context.Context, CreateOrderRequest) (CreateOrderResponse, error)
	CancelOrder(context.Context, CancelOrderRequest) (CancelOrderResponse, error)
	PollOrderStatus(context.Context, PollOrderStatusRequest) (PollOrderStatusResponse, error)
	CreateConversion(context.Context, CreateConversionRequest) (CreateConversionResponse, error)

	// Market data methods
	GetOrderBook(context.Context, GetOrderBookRequest) (GetOrderBookResponse, error)
	GetQuote(context.Context, GetQuoteRequest) (GetQuoteResponse, error)
	GetTradableAssets(context.Context, GetTradableAssetsRequest) (GetTradableAssetsResponse, error)
	GetTicker(context.Context, GetTickerRequest) (GetTickerResponse, error)
	GetOHLC(context.Context, GetOHLCRequest) (GetOHLCResponse, error)
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
	PaymentsToDelete []PSPPaymentsToDelete
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

// Order-related request/response types

type FetchNextOrdersRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextOrdersResponse struct {
	Orders   []PSPOrder
	NewState json.RawMessage
	HasMore  bool
}

type CreateOrderRequest struct {
	Order PSPOrder
}

type CreateOrderResponse struct {
	// If order is immediately created/filled, return it here
	Order *PSPOrder
	// Otherwise, return the order ID to be polled
	PollingOrderID *string
}

type CancelOrderRequest struct {
	OrderID string
}

type CancelOrderResponse struct {
	// Updated order after cancellation
	Order PSPOrder
}

type PollOrderStatusRequest struct {
	OrderID string
}

type PollOrderStatusResponse struct {
	// If nil, the order is not yet in a final state and the function will be
	// called again later
	// If not nil, the order is in a final state and the workflow will be terminated
	Order *PSPOrder

	// If not nil, it means that the order failed, the order will be marked
	// as failed and the workflow will be terminated
	Error *string
}

// Conversion-related request/response types

type FetchNextConversionsRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextConversionsResponse struct {
	Conversions []PSPConversion
	NewState    json.RawMessage
	HasMore     bool
}

type CreateConversionRequest struct {
	Conversion PSPConversion
}

type CreateConversionResponse struct {
	// If conversion is immediately completed, return it here
	Conversion *PSPConversion
	// Otherwise, return the conversion ID to be polled
	PollingConversionID *string
}

// WebSocket-related types for real-time order updates

// OrderUpdateHandler is a callback function that receives order updates from WebSocket
type OrderUpdateHandler func(order PSPOrder)

// WebSocketConfig contains configuration for WebSocket connections
type WebSocketConfig struct {
	// Whether to auto-reconnect on disconnection
	AutoReconnect bool
	// Maximum number of reconnect attempts (0 = unlimited)
	MaxReconnectAttempts int
	// Reconnect delay multiplier for exponential backoff
	ReconnectBackoffMultiplier float64
	// Initial reconnect delay
	InitialReconnectDelay int64 // milliseconds
	// Maximum reconnect delay
	MaxReconnectDelay int64 // milliseconds
}

// DefaultWebSocketConfig returns sensible defaults for WebSocket configuration
func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		AutoReconnect:              true,
		MaxReconnectAttempts:       0, // unlimited
		ReconnectBackoffMultiplier: 2.0,
		InitialReconnectDelay:      1000,  // 1 second
		MaxReconnectDelay:          60000, // 60 seconds
	}
}

// StartOrderWebSocketRequest contains parameters for starting a WebSocket order stream
type StartOrderWebSocketRequest struct {
	Config  WebSocketConfig
	Handler OrderUpdateHandler
}

// StartOrderWebSocketResponse contains the result of starting a WebSocket stream
type StartOrderWebSocketResponse struct {
	// A function to stop the WebSocket connection
	StopFunc func()
}

// WebSocketPlugin interface for connectors that support WebSocket functionality
type WebSocketPlugin interface {
	// StartOrderWebSocket starts a WebSocket connection for real-time order updates
	// Returns a stop function that can be called to close the connection
	StartOrderWebSocket(ctx context.Context, req StartOrderWebSocketRequest) (StartOrderWebSocketResponse, error)
}
