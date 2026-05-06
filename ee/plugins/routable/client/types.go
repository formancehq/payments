package client

import "time"

// Domain models for the Routable API. Field names mirror the Routable JSON
// schema (https://developers.routable.com/reference). We only model the
// subset the dedicated payments plugin needs.

// Pagination links returned on every list endpoint. A non-empty Next signals
// that more pages exist.
type Links struct {
	Self string `json:"self"`
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

// HasMore returns true when the API reports a next page.
func (l Links) HasMore() bool { return l.Next != "" }

// AccountTypeDetails carries balance and bank-detail fields for a settings account.
type AccountTypeDetails struct {
	AvailableAmount string `json:"available_amount"`
	PendingAmount   string `json:"pending_amount"`
	IsUsable        bool   `json:"is_usable"`
	AccountNumber   string `json:"account_number,omitempty"`
	RoutingNumber   string `json:"routing_number,omitempty"`
	AccountType     string `json:"account_type,omitempty"`
	BankName        string `json:"bank_name,omitempty"`
}

// Account is a Routable settings account (funding source). API: GET /v1/settings/accounts.
type Account struct {
	Object       string             `json:"object"`
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Name         string             `json:"name"`
	CurrencyCode string             `json:"currency_code,omitempty"`
	TypeDetails  AccountTypeDetails `json:"type_details"`
	IsValid      bool               `json:"is_valid"`
	CreatedAt    time.Time          `json:"created_at"`
}

type ListAccountsResponse struct {
	Object  string    `json:"object"`
	Results []Account `json:"results"`
	Links   Links     `json:"links"`
}

// Address mirrors the Routable address shape (used by Company.RegisteredAddress).
type Address struct {
	Line1      string `json:"line_1,omitempty"`
	Line2      string `json:"line_2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

// Company is a Routable counterparty (vendor/customer). API: GET /v1/companies.
type Company struct {
	Object            string    `json:"object"`
	ID                string    `json:"id"`
	BusinessName      string    `json:"business_name"`
	DisplayName       string    `json:"display_name"`
	Type              string    `json:"type"`
	Status            string    `json:"status"`
	CountryCode       string    `json:"country_code"`
	ExternalID        string    `json:"external_id,omitempty"`
	IsVendor          bool      `json:"is_vendor"`
	IsCustomer        bool      `json:"is_customer"`
	IsArchived        bool      `json:"is_archived"`
	RegisteredAddress *Address  `json:"registered_address,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type ListCompaniesResponse struct {
	Object  string    `json:"object"`
	Results []Company `json:"results"`
	Links   Links     `json:"links"`
}

// PayableCompany is the embedded company on a Payable response.
type PayableCompany struct {
	Object      string `json:"object,omitempty"`
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
}

// PayableAccount is the embedded settings account on a Payable response.
type PayableAccount struct {
	Object string `json:"object,omitempty"`
	ID     string `json:"id"`
	Type   string `json:"type,omitempty"`
}

// Payable is a Routable outgoing payment. API: GET /v1/payables.
type Payable struct {
	Object              string          `json:"object"`
	ID                  string          `json:"id"`
	Type                string          `json:"type"`
	DeliveryMethod      string          `json:"delivery_method"`
	Status              string          `json:"status"`
	ExternalID          string          `json:"external_id,omitempty"`
	Amount              string          `json:"amount"`
	CurrencyCode        string          `json:"currency_code"`
	PayToCompany        *PayableCompany `json:"pay_to_company,omitempty"`
	WithdrawFromAccount *PayableAccount `json:"withdraw_from_account,omitempty"`
	Memo                string          `json:"memo,omitempty"`
	Reference           string          `json:"reference,omitempty"`
	StatusChangedAt     *time.Time      `json:"status_changed_at,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
}

type ListPayablesResponse struct {
	Object  string    `json:"object"`
	Results []Payable `json:"results"`
	Links   Links     `json:"links"`
}

// ReceivableCompany is the embedded company on a Receivable response.
type ReceivableCompany struct {
	Object      string `json:"object,omitempty"`
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
}

// ReceivableAccount is the embedded settings account on a Receivable response.
type ReceivableAccount struct {
	Object string `json:"object,omitempty"`
	ID     string `json:"id"`
	Type   string `json:"type,omitempty"`
}

// Receivable is a Routable incoming payment. API: GET /v1/receivables.
type Receivable struct {
	Object           string             `json:"object"`
	ID               string             `json:"id"`
	Type             string             `json:"type"`
	DeliveryMethod   string             `json:"delivery_method"`
	Status           string             `json:"status"`
	ExternalID       string             `json:"external_id,omitempty"`
	Amount           string             `json:"amount"`
	CurrencyCode     string             `json:"currency_code"`
	PayFromCompany   *ReceivableCompany `json:"pay_from_company,omitempty"`
	DepositToAccount *ReceivableAccount `json:"deposit_to_account,omitempty"`
	Memo             string             `json:"memo,omitempty"`
	Reference        string             `json:"reference,omitempty"`
	StatusChangedAt  *time.Time         `json:"status_changed_at,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
}

type ListReceivablesResponse struct {
	Object  string       `json:"object"`
	Results []Receivable `json:"results"`
	Links   Links        `json:"links"`
}

// PayableLineItem is required on POST /v1/payables. Routable rejects payables
// without at least one line item; we synthesise a single one mirroring the
// total amount when the caller does not supply richer detail.
type PayableLineItem struct {
	UnitPrice   string `json:"unit_price"`
	Amount      string `json:"amount"`
	Quantity    int    `json:"quantity,omitempty"`
	Description string `json:"description,omitempty"`
}

// CreatePayableRequest is the body for POST /v1/payables. IdempotencyKey is
// sent via the Idempotency-Key header (see Client.CreatePayable).
type CreatePayableRequest struct {
	Type                string            `json:"type"`
	DeliveryMethod      string            `json:"delivery_method"`
	PayToCompany        string            `json:"pay_to_company"`
	WithdrawFromAccount string            `json:"withdraw_from_account"`
	Amount              string            `json:"amount"`
	CurrencyCode        string            `json:"currency_code,omitempty"`
	LineItems           []PayableLineItem `json:"line_items"`
	ActingTeamMember    string            `json:"acting_team_member"`
	Reference           string            `json:"reference,omitempty"`
	ExternalID          string            `json:"external_id,omitempty"`
	Memo                string            `json:"memo,omitempty"`

	// IdempotencyKey is sent via the Idempotency-Key header, never the body.
	IdempotencyKey string `json:"-"`
}

// ErrorResponse is the JSON Routable returns on non-2xx responses.
type ErrorResponse struct {
	Object  string       `json:"object,omitempty"`
	Code    string       `json:"code,omitempty"`
	Message string       `json:"message,omitempty"`
	Errors  []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}
