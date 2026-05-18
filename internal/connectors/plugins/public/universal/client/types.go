package client

import "time"

// CapabilitiesResponse is the v1/capabilities envelope. Supported lists the
// primitives the counterparty implements, using the same string values the
// engine uses (FETCH_ACCOUNTS, CREATE_PAYOUT, ...). Features captures
// non-capability axes the plugin needs to know about at install time.
type CapabilitiesResponse struct {
	Supported []string `json:"supported"`
	Features  Features `json:"features"`
}

type Features struct {
	Pagination        string `json:"pagination"`        // "cursor" | "page" | "none"
	WebhookSignature  string `json:"webhookSignature"`  // "hmac-sha256" | "none"
	IdempotencyHeader string `json:"idempotencyHeader"` // override; empty means default "Idempotency-Key"

	// EventStream advertises a real-time push transport.
	// "" (default) → HTTP webhooks only.
	// "wss"        → counterparty also exposes /v1/stream as a
	//                signed WebSocket; client opts in at install.
	EventStream string `json:"eventStream"`

	// StreamEvents lists the event names the counterparty publishes
	// over the stream. Sentinel ["*"] means "every event I publish on
	// webhooks I also publish on the stream". Empty + EventStream=wss
	// fails install (no events to subscribe to is a misconfiguration).
	// Always intersected with the plugin's supportedWebhookNames so we
	// never subscribe to an event the engine can't route.
	StreamEvents []string `json:"streamEvents,omitempty"`
}

// Account is the wire shape for both internal accounts (GET /v1/accounts) and
// external accounts (GET /v1/external-accounts). Reference is the raw PSP id
// — the engine namespaces by ConnectorID, so do not pre-namespace.
type Account struct {
	Reference    string            `json:"reference"`
	CreatedAt    time.Time         `json:"createdAt"`
	Name         *string           `json:"name,omitempty"`
	DefaultAsset *string           `json:"defaultAsset,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type AccountsPage struct {
	Items      []Account `json:"items"`
	NextCursor string    `json:"nextCursor,omitempty"`
	HasMore    bool      `json:"hasMore"`
}

// Balance is one entry in BalancesResponse. Asset uses UMN format ("USD/2",
// "BTC/8"); amounts are decimal-string minor units to preserve precision.
type Balance struct {
	AccountReference string    `json:"accountReference"`
	CreatedAt        time.Time `json:"createdAt"`
	Amount           string    `json:"amount"`
	Asset            string    `json:"asset"`
}

type BalancesResponse struct {
	Items []Balance `json:"items"`
}

// Payment mirrors models.PSPPayment. Status, Type and Scheme MUST use the
// canonical string values from the corresponding internal/models/*.go files
// (e.g. "SUCCEEDED", "PAYIN", "SEPA"). Unknown values map to OTHER on the
// plugin side.
type Payment struct {
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

type PaymentsPage struct {
	Items            []Payment        `json:"items"`
	PaymentsToDelete []ToDeleteRecord `json:"paymentsToDelete,omitempty"`
	NextCursor       string           `json:"nextCursor,omitempty"`
	HasMore          bool             `json:"hasMore"`
}

type ToDeleteRecord struct {
	Reference string `json:"reference"`
}

// Order mirrors models.PSPOrder. Quantities, prices and fees are decimal
// strings of minor units; QuoteAsset / FeeAsset / PriceAsset are UMN strings
// to give the engine precision context for analytics.
type Order struct {
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
	LimitPrice                  string            `json:"limitPrice,omitempty"`
	StopPrice                   string            `json:"stopPrice,omitempty"`
	TimeInForce                 string            `json:"timeInForce,omitempty"`
	ExpiresAt                   *time.Time        `json:"expiresAt,omitempty"`
	QuoteAmount                 string            `json:"quoteAmount,omitempty"`
	QuoteAsset                  string            `json:"quoteAsset,omitempty"`
	Fee                         string            `json:"fee,omitempty"`
	FeeAsset                    *string           `json:"feeAsset,omitempty"`
	AverageFillPrice            string            `json:"averageFillPrice,omitempty"`
	PriceAsset                  *string           `json:"priceAsset,omitempty"`
	SourceAccountReference      *string           `json:"sourceAccountReference,omitempty"`
	DestinationAccountReference *string           `json:"destinationAccountReference,omitempty"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

type OrdersPage struct {
	Items      []Order `json:"items"`
	NextCursor string  `json:"nextCursor,omitempty"`
	HasMore    bool    `json:"hasMore"`
}

// Conversion mirrors models.PSPConversion. Status uses canonical conversion
// status strings: PENDING, COMPLETED, FAILED.
type Conversion struct {
	Reference                   string            `json:"reference"`
	CreatedAt                   time.Time         `json:"createdAt"`
	Status                      string            `json:"status"`
	SourceAsset                 string            `json:"sourceAsset"`
	DestinationAsset            string            `json:"destinationAsset"`
	SourceAmount                string            `json:"sourceAmount"`
	DestinationAmount           string            `json:"destinationAmount,omitempty"`
	Fee                         string            `json:"fee,omitempty"`
	FeeAsset                    *string           `json:"feeAsset,omitempty"`
	SourceAccountReference      *string           `json:"sourceAccountReference,omitempty"`
	DestinationAccountReference *string           `json:"destinationAccountReference,omitempty"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

type ConversionsPage struct {
	Items      []Conversion `json:"items"`
	NextCursor string       `json:"nextCursor,omitempty"`
	HasMore    bool         `json:"hasMore"`
}

// Other is the catch-all opaque entity used by FETCH_OTHERS. The counterparty
// is free to put arbitrary JSON in `data`; we forward it untouched.
type Other struct {
	ID   string `json:"id"`
	Data any    `json:"data"`
}

type OthersPage struct {
	Items      []Other `json:"items"`
	NextCursor string  `json:"nextCursor,omitempty"`
	HasMore    bool    `json:"hasMore"`
}

// PayoutRequest / TransferRequest mirror PSPPaymentInitiation. Counterparties
// with restricted source/destination semantics validate beyond the schema and
// return a 4xx error envelope (which our Error type handles). Reference is the
// engine-supplied initiation reference; counterparty MUST dedup on
// IdempotencyKey, not on Reference.
type PayoutRequest struct {
	Reference                   string            `json:"reference"`
	Description                 string            `json:"description,omitempty"`
	Amount                      string            `json:"amount"`
	Asset                       string            `json:"asset"`
	SourceAccountReference      string            `json:"sourceAccountReference"`
	DestinationAccountReference string            `json:"destinationAccountReference"`
	Metadata                    map[string]string `json:"metadata,omitempty"`
}

type TransferRequest = PayoutRequest

// Mode is one of "terminal" (the response Payment is the final state) or
// "polling" (the engine should call GetPayout/GetTransfer until terminal or
// error). Counterparties pick which one fits their architecture per request.
type PayoutResponse struct {
	Mode      string   `json:"mode"`
	PollingID string   `json:"pollingID,omitempty"`
	Payment   *Payment `json:"payment,omitempty"`
	Error     string   `json:"error,omitempty"`
}

type TransferResponse = PayoutResponse

type ReverseRequest struct {
	Reference   string            `json:"reference"`
	Description string            `json:"description,omitempty"`
	Amount      string            `json:"amount"`
	Asset       string            `json:"asset"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// BankAccountRequest is the payload that backs CreateBankAccount. Fields
// follow the `BankAccount` model in internal/models/bank_accounts.go.
type BankAccountRequest struct {
	ID            string            `json:"id"`
	CreatedAt     time.Time         `json:"createdAt"`
	Name          string            `json:"name"`
	AccountNumber *string           `json:"accountNumber,omitempty"`
	IBAN          *string           `json:"iban,omitempty"`
	SwiftBicCode  *string           `json:"swiftBicCode,omitempty"`
	Country       *string           `json:"country,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type BankAccountResponse struct {
	RelatedAccount Account `json:"relatedAccount"`
}

// WebhookSubscriptionRequest tells the counterparty where to deliver events
// for a given event Name. CallbackURL is built by the engine from the
// publicly-routable webhook URL; the counterparty MUST sign POST bodies with
// the secret it shares OOB if it advertised webhookSignature == "hmac-sha256".
type WebhookSubscriptionRequest struct {
	Name        string            `json:"name"`
	CallbackURL string            `json:"callbackUrl"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type WebhookSubscriptionResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// WebhookEvent is the envelope every counterparty POST body must conform to.
// Type drives dispatch in TranslateWebhook (events catalog at
// contract/universal-events.md). Resource carries the typed inline payload.
type WebhookEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	CreatedAt time.Time       `json:"createdAt"`
	Resource  WebhookResource `json:"resource"`
}

type WebhookResource struct {
	Account         *Account    `json:"account,omitempty"`
	ExternalAccount *Account    `json:"externalAccount,omitempty"`
	Payment         *Payment    `json:"payment,omitempty"`
	Order           *Order      `json:"order,omitempty"`
	Conversion      *Conversion `json:"conversion,omitempty"`
	Balance         *Balance    `json:"balance,omitempty"`
	PaymentToDelete *string     `json:"paymentToDelete,omitempty"`
	PaymentToCancel *string     `json:"paymentToCancel,omitempty"`
}
