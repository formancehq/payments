package workbench

import (
	"encoding/json"
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
