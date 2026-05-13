package main

import (
	"strconv"
	"time"
)

// The mock keeps its own wire types — intentionally decoupled from the
// plugin's `client/types.go` so the two can evolve in lockstep with the
// contract without import cycles. Field names and JSON tags MUST match the
// contract's universal-openapi.yaml exactly; the test suite cross-checks
// them by speaking the contract via the plugin.

type capabilities struct {
	Supported []string `json:"supported"`
	Features  features `json:"features"`
}

type features struct {
	Pagination        string `json:"pagination"`
	WebhookSignature  string `json:"webhookSignature"`
	IdempotencyHeader string `json:"idempotencyHeader,omitempty"`
}

type account struct {
	Reference    string            `json:"reference"`
	CreatedAt    time.Time         `json:"createdAt"`
	Name         *string           `json:"name,omitempty"`
	DefaultAsset *string           `json:"defaultAsset,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type accountsPage struct {
	Items      []account `json:"items"`
	NextCursor string    `json:"nextCursor,omitempty"`
	HasMore    bool      `json:"hasMore"`
}

type balance struct {
	AccountReference string    `json:"accountReference"`
	CreatedAt        time.Time `json:"createdAt"`
	Amount           string    `json:"amount"`
	Asset            string    `json:"asset"`
}

type balancesResponse struct {
	Items []balance `json:"items"`
}

type payment struct {
	Reference                   string            `json:"reference"`
	ParentReference             string            `json:"parentReference,omitempty"`
	CreatedAt                   time.Time         `json:"createdAt"`
	UpdatedAt                   time.Time         `json:"updatedAt"`
	Type                        string            `json:"type"`
	Status                      string            `json:"status"`
	Scheme                      string            `json:"scheme,omitempty"`
	Amount                      string            `json:"amount"`
	Asset                       string            `json:"asset"`
	SourceAccountReference      *string           `json:"sourceAccountReference,omitempty"`
	DestinationAccountReference *string           `json:"destinationAccountReference,omitempty"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

type paymentsPage struct {
	Items      []payment `json:"items"`
	NextCursor string    `json:"nextCursor,omitempty"`
	HasMore    bool      `json:"hasMore"`
}

type order struct {
	Reference                   string            `json:"reference"`
	ClientOrderID               string            `json:"clientOrderID,omitempty"`
	CreatedAt                   time.Time         `json:"createdAt"`
	UpdatedAt                   time.Time         `json:"updatedAt"`
	Direction                   string            `json:"direction"`
	Type                        string            `json:"type"`
	Status                      string            `json:"status"`
	SourceAsset                 string            `json:"sourceAsset"`
	DestinationAsset            string            `json:"destinationAsset"`
	BaseQuantityOrdered         string            `json:"baseQuantityOrdered"`
	BaseQuantityFilled          string            `json:"baseQuantityFilled,omitempty"`
	TimeInForce                 string            `json:"timeInForce,omitempty"`
	QuoteAmount                 string            `json:"quoteAmount,omitempty"`
	QuoteAsset                  string            `json:"quoteAsset,omitempty"`
	SourceAccountReference      *string           `json:"sourceAccountReference,omitempty"`
	DestinationAccountReference *string           `json:"destinationAccountReference,omitempty"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

type ordersPage struct {
	Items      []order `json:"items"`
	NextCursor string  `json:"nextCursor,omitempty"`
	HasMore    bool    `json:"hasMore"`
}

type conversion struct {
	Reference                   string            `json:"reference"`
	CreatedAt                   time.Time         `json:"createdAt"`
	Status                      string            `json:"status"`
	SourceAsset                 string            `json:"sourceAsset"`
	DestinationAsset            string            `json:"destinationAsset"`
	SourceAmount                string            `json:"sourceAmount"`
	DestinationAmount           string            `json:"destinationAmount,omitempty"`
	SourceAccountReference      *string           `json:"sourceAccountReference,omitempty"`
	DestinationAccountReference *string           `json:"destinationAccountReference,omitempty"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

type conversionsPage struct {
	Items      []conversion `json:"items"`
	NextCursor string       `json:"nextCursor,omitempty"`
	HasMore    bool         `json:"hasMore"`
}

type other struct {
	ID   string `json:"id"`
	Data any    `json:"data"`
}

type othersPage struct {
	Items      []other `json:"items"`
	NextCursor string  `json:"nextCursor,omitempty"`
	HasMore    bool    `json:"hasMore"`
}

type initiationRequest struct {
	Reference                   string            `json:"reference"`
	Description                 string            `json:"description,omitempty"`
	Amount                      string            `json:"amount"`
	Asset                       string            `json:"asset"`
	SourceAccountReference      string            `json:"sourceAccountReference"`
	DestinationAccountReference string            `json:"destinationAccountReference"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

type initiationResponse struct {
	Mode      string   `json:"mode"`
	PollingID string   `json:"pollingID,omitempty"`
	Payment   *payment `json:"payment,omitempty"`
	Error     string   `json:"error,omitempty"`
}

type bankAccountRequest struct {
	ID            string            `json:"id"`
	CreatedAt     time.Time         `json:"createdAt"`
	Name          string            `json:"name"`
	AccountNumber *string           `json:"accountNumber,omitempty"`
	IBAN          *string           `json:"iban,omitempty"`
	SwiftBicCode  *string           `json:"swiftBicCode,omitempty"`
	Country       *string           `json:"country,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type bankAccountResponse struct {
	RelatedAccount account `json:"relatedAccount"`
}

type webhookSubscriptionRequest struct {
	Name        string            `json:"name"`
	CallbackURL string            `json:"callbackUrl"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type webhookSubscriptionResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type errorResponse struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func mustParseInt(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}
