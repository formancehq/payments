package workbench

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

// Response types with proper JSON tags for the workbench UI.
// These convert from internal models.PSP* types to API-friendly formats.

// AccountResponse is the JSON-serializable account for the UI.
type AccountResponse struct {
	Reference    string            `json:"reference"`
	CreatedAt    time.Time         `json:"created_at"`
	Name         *string           `json:"name,omitempty"`
	DefaultAsset *string           `json:"default_asset,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Raw          json.RawMessage   `json:"raw,omitempty"`
}

// PaymentResponse is the JSON-serializable payment for the UI.
type PaymentResponse struct {
	Reference                   string            `json:"reference"`
	ParentReference             string            `json:"parent_reference,omitempty"`
	CreatedAt                   time.Time         `json:"created_at"`
	Type                        string            `json:"type"`
	Amount                      string            `json:"amount"`
	Asset                       string            `json:"asset"`
	Scheme                      string            `json:"scheme,omitempty"`
	Status                      string            `json:"status"`
	SourceAccountReference      *string           `json:"source_account_reference,omitempty"`
	DestinationAccountReference *string           `json:"destination_account_reference,omitempty"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
	Raw                         json.RawMessage   `json:"raw,omitempty"`
}

// BalanceResponse is the JSON-serializable balance for the UI.
type BalanceResponse struct {
	AccountReference string    `json:"account_reference"`
	CreatedAt        time.Time `json:"created_at"`
	Amount           string    `json:"amount"`
	Asset            string    `json:"asset"`
}

// ToAccountResponse converts a PSPAccount to an AccountResponse.
func ToAccountResponse(acc models.PSPAccount) AccountResponse {
	return AccountResponse{
		Reference:    acc.Reference,
		CreatedAt:    acc.CreatedAt,
		Name:         acc.Name,
		DefaultAsset: acc.DefaultAsset,
		Metadata:     acc.Metadata,
		Raw:          acc.Raw,
	}
}

// ToAccountResponses converts a slice of PSPAccounts to AccountResponses.
func ToAccountResponses(accounts []models.PSPAccount) []AccountResponse {
	result := make([]AccountResponse, len(accounts))
	for i, acc := range accounts {
		result[i] = ToAccountResponse(acc)
	}
	return result
}

// ToPaymentResponse converts a PSPPayment to a PaymentResponse.
func ToPaymentResponse(pay models.PSPPayment) PaymentResponse {
	amount := "0"
	if pay.Amount != nil {
		amount = pay.Amount.String()
	}

	return PaymentResponse{
		Reference:                   pay.Reference,
		ParentReference:             pay.ParentReference,
		CreatedAt:                   pay.CreatedAt,
		Type:                        pay.Type.String(),
		Amount:                      amount,
		Asset:                       pay.Asset,
		Scheme:                      pay.Scheme.String(),
		Status:                      pay.Status.String(),
		SourceAccountReference:      pay.SourceAccountReference,
		DestinationAccountReference: pay.DestinationAccountReference,
		Metadata:                    pay.Metadata,
		Raw:                         pay.Raw,
	}
}

// ToPaymentResponses converts a slice of PSPPayments to PaymentResponses.
func ToPaymentResponses(payments []models.PSPPayment) []PaymentResponse {
	result := make([]PaymentResponse, len(payments))
	for i, pay := range payments {
		result[i] = ToPaymentResponse(pay)
	}
	return result
}

// ToBalanceResponse converts a PSPBalance to a BalanceResponse.
func ToBalanceResponse(bal models.PSPBalance) BalanceResponse {
	amount := "0"
	if bal.Amount != nil {
		amount = bal.Amount.String()
	}

	return BalanceResponse{
		AccountReference: bal.AccountReference,
		CreatedAt:        bal.CreatedAt,
		Amount:           amount,
		Asset:            bal.Asset,
	}
}

// ToBalanceResponses converts a slice of PSPBalances to BalanceResponses.
func ToBalanceResponses(balances []models.PSPBalance) []BalanceResponse {
	result := make([]BalanceResponse, len(balances))
	for i, bal := range balances {
		result[i] = ToBalanceResponse(bal)
	}
	return result
}

// OrderResponse is the JSON-serializable order for the UI.
type OrderResponse struct {
	Reference           string            `json:"reference"`
	CreatedAt           time.Time         `json:"created_at"`
	Direction           string            `json:"direction"`
	SourceAsset         string            `json:"source_asset"`
	DestinationAsset         string            `json:"destination_asset"`
	Type                string            `json:"type"`
	Status              string            `json:"status"`
	BaseQuantityOrdered string            `json:"base_quantity_ordered"`
	BaseQuantityFilled  string            `json:"base_quantity_filled,omitempty"`
	LimitPrice          string            `json:"limit_price,omitempty"`
	StopPrice           string            `json:"stop_price,omitempty"`
	TimeInForce         string            `json:"time_in_force,omitempty"`
	ExpiresAt           *time.Time        `json:"expires_at,omitempty"`
	Fee                 string            `json:"fee,omitempty"`
	FeeAsset            *string           `json:"fee_asset,omitempty"`
	AverageFillPrice    string            `json:"average_fill_price,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	Raw                 json.RawMessage   `json:"raw,omitempty"`
}

// ToOrderResponse converts a PSPOrder to an OrderResponse.
func ToOrderResponse(o models.PSPOrder) OrderResponse {
	resp := OrderResponse{
		Reference:           o.Reference,
		CreatedAt:           o.CreatedAt,
		Direction:           o.Direction.String(),
		SourceAsset:         o.SourceAsset,
		DestinationAsset:         o.DestinationAsset,
		Type:                o.Type.String(),
		Status:              o.Status.String(),
		BaseQuantityOrdered: bigIntToString(o.BaseQuantityOrdered),
		BaseQuantityFilled:  bigIntToString(o.BaseQuantityFilled),
		LimitPrice:          bigIntToString(o.LimitPrice),
		StopPrice:           bigIntToString(o.StopPrice),
		TimeInForce:         o.TimeInForce.String(),
		ExpiresAt:           o.ExpiresAt,
		Fee:                 bigIntToString(o.Fee),
		FeeAsset:            o.FeeAsset,
		AverageFillPrice:    bigIntToString(o.AverageFillPrice),
		Metadata:            o.Metadata,
		Raw:                 o.Raw,
	}
	return resp
}

// ToOrderResponses converts a slice of PSPOrders to OrderResponses.
func ToOrderResponses(orders []models.PSPOrder) []OrderResponse {
	result := make([]OrderResponse, len(orders))
	for i, o := range orders {
		result[i] = ToOrderResponse(o)
	}
	return result
}

// ConversionResponse is the JSON-serializable conversion for the UI.
type ConversionResponse struct {
	Reference    string            `json:"reference"`
	CreatedAt    time.Time         `json:"created_at"`
	SourceAsset  string            `json:"source_asset"`
	DestinationAsset  string            `json:"destination_asset"`
	SourceAmount string            `json:"source_amount"`
	DestinationAmount string            `json:"target_amount,omitempty"`
	Status               string            `json:"status"`
	SourceAccountID      string            `json:"source_account_id,omitempty"`
	DestinationAccountID string            `json:"destination_account_id,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	Raw          json.RawMessage   `json:"raw,omitempty"`
}

// ToConversionResponse converts a PSPConversion to a ConversionResponse.
func ToConversionResponse(c models.PSPConversion) ConversionResponse {
	return ConversionResponse{
		Reference:    c.Reference,
		CreatedAt:    c.CreatedAt,
		SourceAsset:  c.SourceAsset,
		DestinationAsset:  c.DestinationAsset,
		SourceAmount: bigIntToString(c.SourceAmount),
		DestinationAmount: bigIntToString(c.DestinationAmount),
		Status:               c.Status.String(),
		SourceAccountID:      ptrToString(c.SourceAccountReference),
		DestinationAccountID: ptrToString(c.DestinationAccountReference),
		Metadata:     c.Metadata,
		Raw:          c.Raw,
	}
}

// ToConversionResponses converts a slice of PSPConversions to ConversionResponses.
func ToConversionResponses(conversions []models.PSPConversion) []ConversionResponse {
	result := make([]ConversionResponse, len(conversions))
	for i, c := range conversions {
		result[i] = ToConversionResponse(c)
	}
	return result
}

func ptrToString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func bigIntToString(v *big.Int) string {
	if v == nil {
		return ""
	}
	return v.String()
}
