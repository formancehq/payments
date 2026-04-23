package models

import (
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/utils/assets"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/gibson042/canonicaljson-go"
)

// PSPOrder represents an order from a Payment Service Provider (exchange).
// This is the internal struct used by plugins to communicate order data.
type PSPOrder struct {
	// PSP order reference. Should be unique within the connector.
	Reference string

	// Client-assigned order ID used by the exchange for placement idempotency.
	// The caller provides this when submitting the order; the exchange deduplicates
	// retried submissions using it. Formance stores it for traceability only — our
	// storage dedup keys on Reference (the exchange-assigned ID), not this field.
	// Exchange-specific names: Coinbase client_order_id, Kraken cl_ord_id, Binance clientOrderId.
	ClientOrderID string

	// Order creation date
	CreatedAt time.Time

	// Order direction: BUY or SELL
	Direction OrderDirection

	// Source asset (what you're trading from)
	// For BUY orders: the quote currency (e.g., EUR in BTC/EUR)
	// For SELL orders: the base currency (e.g., BTC in BTC/EUR)
	SourceAsset string

	// Target asset (what you're trading to)
	// For BUY orders: the base currency (e.g., BTC in BTC/EUR)
	// For SELL orders: the quote currency (e.g., EUR in BTC/EUR)
	DestinationAsset string

	// Order type: MARKET or LIMIT
	Type OrderType

	// Order status
	Status OrderStatus

	// Base quantity ordered (in base asset units, using integer representation)
	BaseQuantityOrdered *big.Int

	// Base quantity filled (in base asset units, using integer representation)
	BaseQuantityFilled *big.Int

	// Limit price for LIMIT orders (optional, using integer representation)
	LimitPrice *big.Int

	// Stop price for STOP_LIMIT orders (optional, using integer representation)
	StopPrice *big.Int

	// Time in force
	TimeInForce TimeInForce

	// Expiration time for GTD orders
	ExpiresAt *time.Time

	// Quote amount filled (in quote asset units, e.g. USD cents).
	// For BUY: how much quote currency was spent. For SELL: how much was received.
	// This is the exact filled_value from the exchange, parsed at quote precision.
	QuoteAmount *big.Int

	// Quote asset with precision (e.g. "USD/2")
	QuoteAsset string

	// Fee charged for the order (using integer representation, in quote currency)
	Fee *big.Int

	// Fee asset with precision (e.g. "USD/2")
	FeeAsset *string

	// Average fill price (using integer representation, analytics only)
	AverageFillPrice *big.Int

	// Price asset with precision for interpreting price fields (analytics only)
	PriceAsset *string

	// Account references (wallet UUIDs) for source and destination.
	// BUY: source = quote wallet (USD), destination = base wallet (crypto)
	// SELL: source = base wallet (crypto), destination = quote wallet (USD)
	SourceAccountReference      *string
	DestinationAccountReference *string

	// Additional metadata
	Metadata map[string]string

	// PSP response in raw format
	Raw json.RawMessage
}

func (o *PSPOrder) Validate() error {
	if o.Reference == "" {
		return errorsutils.NewWrappedError(errors.New("missing order reference"), ErrValidation)
	}

	if o.CreatedAt.IsZero() {
		return errorsutils.NewWrappedError(errors.New("missing order createdAt"), ErrValidation)
	}

	if o.Direction == ORDER_DIRECTION_UNKNOWN {
		return errorsutils.NewWrappedError(errors.New("missing order direction"), ErrValidation)
	}

	if !assets.IsValid(o.SourceAsset) {
		return errorsutils.NewWrappedError(errors.New("invalid order source asset"), ErrValidation)
	}

	if !assets.IsValid(o.DestinationAsset) {
		return errorsutils.NewWrappedError(errors.New("invalid order target asset"), ErrValidation)
	}

	if o.Type == ORDER_TYPE_UNKNOWN {
		return errorsutils.NewWrappedError(errors.New("missing order type"), ErrValidation)
	}

	if o.Status == ORDER_STATUS_UNKNOWN {
		return errorsutils.NewWrappedError(errors.New("missing order status"), ErrValidation)
	}

	if o.BaseQuantityOrdered == nil {
		return errorsutils.NewWrappedError(errors.New("missing order base quantity ordered"), ErrValidation)
	}

	if o.Raw == nil {
		return errorsutils.NewWrappedError(errors.New("missing order raw"), ErrValidation)
	}

	return nil
}

// Order represents a trading order in Formance.
type Order struct {
	// Unique Order ID generated from order information
	ID OrderID `json:"id"`

	// Related Connector ID
	ConnectorID ConnectorID `json:"connectorID"`

	// PSP order reference
	Reference string `json:"reference"`

	// Client-assigned order ID for exchange-side placement idempotency (not used for storage dedup).
	ClientOrderID string `json:"clientOrderID,omitempty"`

	// Order creation date
	CreatedAt time.Time `json:"createdAt"`

	// Last update date
	UpdatedAt time.Time `json:"updatedAt"`

	// Order direction: BUY or SELL
	Direction OrderDirection `json:"direction"`

	// Source asset
	SourceAsset string `json:"sourceAsset"`

	// Target asset
	DestinationAsset string `json:"destinationAsset"`

	// Order type: MARKET or LIMIT
	Type OrderType `json:"type"`

	// Order status
	Status OrderStatus `json:"status"`

	// Base quantity ordered (using integer representation)
	BaseQuantityOrdered *big.Int `json:"baseQuantityOrdered"`

	// Base quantity filled (using integer representation)
	BaseQuantityFilled *big.Int `json:"baseQuantityFilled"`

	// Limit price for LIMIT orders (optional)
	LimitPrice *big.Int `json:"limitPrice,omitempty"`

	// Stop price for STOP_LIMIT orders (optional)
	StopPrice *big.Int `json:"stopPrice,omitempty"`

	// Time in force
	TimeInForce TimeInForce `json:"timeInForce"`

	// Expiration time for GTD orders
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	// Quote amount filled (in quote asset units, integer)
	QuoteAmount *big.Int `json:"quoteAmount,omitempty"`

	// Quote asset with precision (e.g. "USD/2")
	QuoteAsset string `json:"quoteAsset,omitempty"`

	// Fee charged (in quote currency, integer)
	Fee *big.Int `json:"fee,omitempty"`

	// Fee asset with precision (e.g. "USD/2")
	FeeAsset *string `json:"feeAsset,omitempty"`

	// Average fill price (analytics, integer at priceAsset precision)
	AverageFillPrice *big.Int `json:"averageFillPrice,omitempty"`

	// Price asset with precision for interpreting price fields (analytics)
	PriceAsset *string `json:"priceAsset,omitempty"`

	// Account references (Formance AccountIDs built from wallet UUID + connectorID)
	SourceAccountID      *AccountID `json:"sourceAccountID"`
	DestinationAccountID *AccountID `json:"destinationAccountID"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`

	// Related adjustments (status changes)
	Adjustments []OrderAdjustment `json:"adjustments"`
}

func (o Order) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                  string            `json:"id"`
		ConnectorID         string            `json:"connectorID"`
		Provider            string            `json:"provider"`
		Reference           string            `json:"reference"`
		ClientOrderID       string            `json:"clientOrderID,omitempty"`
		CreatedAt           time.Time         `json:"createdAt"`
		UpdatedAt           time.Time         `json:"updatedAt"`
		Direction           OrderDirection    `json:"direction"`
		SourceAsset         string            `json:"sourceAsset"`
		DestinationAsset         string            `json:"destinationAsset"`
		Type                OrderType         `json:"type"`
		Status              OrderStatus       `json:"status"`
		BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
		LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
		StopPrice           *big.Int          `json:"stopPrice,omitempty"`
		TimeInForce         TimeInForce       `json:"timeInForce"`
		ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
		QuoteAmount         *big.Int          `json:"quoteAmount,omitempty"`
		QuoteAsset          string            `json:"quoteAsset,omitempty"`
		Fee                 *big.Int          `json:"fee,omitempty"`
		FeeAsset            *string           `json:"feeAsset,omitempty"`
		AverageFillPrice    *big.Int          `json:"averageFillPrice,omitempty"`
		PriceAsset           *string           `json:"priceAsset,omitempty"`
		SourceAccountID      *string           `json:"sourceAccountID"`
		DestinationAccountID *string           `json:"destinationAccountID"`
		Metadata             map[string]string `json:"metadata"`
		Adjustments          []OrderAdjustment `json:"adjustments"`
	}{
		ID:                  o.ID.String(),
		ConnectorID:         o.ConnectorID.String(),
		Provider:            ToV3Provider(o.ConnectorID.Provider),
		Reference:           o.Reference,
		ClientOrderID:       o.ClientOrderID,
		CreatedAt:           o.CreatedAt,
		UpdatedAt:           o.UpdatedAt,
		Direction:           o.Direction,
		SourceAsset:         o.SourceAsset,
		DestinationAsset:    o.DestinationAsset,
		Type:                o.Type,
		Status:              o.Status,
		BaseQuantityOrdered: o.BaseQuantityOrdered,
		BaseQuantityFilled:  o.BaseQuantityFilled,
		LimitPrice:          o.LimitPrice,
		StopPrice:           o.StopPrice,
		TimeInForce:         o.TimeInForce,
		ExpiresAt:           o.ExpiresAt,
		QuoteAmount:         o.QuoteAmount,
		QuoteAsset:          o.QuoteAsset,
		Fee:                 o.Fee,
		FeeAsset:            o.FeeAsset,
		AverageFillPrice:    o.AverageFillPrice,
		PriceAsset:          o.PriceAsset,
		SourceAccountID:      o.SourceAccountID.StringPtr(),
		DestinationAccountID: o.DestinationAccountID.StringPtr(),
		Metadata:    o.Metadata,
		Adjustments: o.Adjustments,
	})
}

func (o *Order) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                  string            `json:"id"`
		ConnectorID         string            `json:"connectorID"`
		Reference           string            `json:"reference"`
		ClientOrderID       string            `json:"clientOrderID,omitempty"`
		CreatedAt           time.Time         `json:"createdAt"`
		UpdatedAt           time.Time         `json:"updatedAt"`
		Direction           OrderDirection    `json:"direction"`
		SourceAsset         string            `json:"sourceAsset"`
		DestinationAsset         string            `json:"destinationAsset"`
		Type                OrderType         `json:"type"`
		Status              OrderStatus       `json:"status"`
		BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
		LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
		StopPrice           *big.Int          `json:"stopPrice,omitempty"`
		TimeInForce         TimeInForce       `json:"timeInForce"`
		ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
		QuoteAmount         *big.Int          `json:"quoteAmount,omitempty"`
		QuoteAsset          string            `json:"quoteAsset,omitempty"`
		Fee                 *big.Int          `json:"fee,omitempty"`
		FeeAsset            *string           `json:"feeAsset,omitempty"`
		AverageFillPrice    *big.Int          `json:"averageFillPrice,omitempty"`
		PriceAsset           *string           `json:"priceAsset,omitempty"`
		SourceAccountID      *string           `json:"sourceAccountID,omitempty"`
		DestinationAccountID *string           `json:"destinationAccountID,omitempty"`
		Metadata             map[string]string `json:"metadata"`
		Adjustments          []OrderAdjustment `json:"adjustments"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := OrderIDFromString(aux.ID)
	if err != nil {
		return err
	}

	connectorID, err := ConnectorIDFromString(aux.ConnectorID)
	if err != nil {
		return err
	}

	o.ID = id
	o.ConnectorID = connectorID
	o.Reference = aux.Reference
	o.ClientOrderID = aux.ClientOrderID
	o.CreatedAt = aux.CreatedAt
	o.UpdatedAt = aux.UpdatedAt
	o.Direction = aux.Direction
	o.SourceAsset = aux.SourceAsset
	o.DestinationAsset = aux.DestinationAsset
	o.Type = aux.Type
	o.Status = aux.Status
	o.BaseQuantityOrdered = aux.BaseQuantityOrdered
	o.BaseQuantityFilled = aux.BaseQuantityFilled
	o.LimitPrice = aux.LimitPrice
	o.StopPrice = aux.StopPrice
	o.TimeInForce = aux.TimeInForce
	o.ExpiresAt = aux.ExpiresAt
	o.QuoteAmount = aux.QuoteAmount
	o.QuoteAsset = aux.QuoteAsset
	o.Fee = aux.Fee
	o.FeeAsset = aux.FeeAsset
	o.AverageFillPrice = aux.AverageFillPrice
	o.PriceAsset = aux.PriceAsset
	if aux.SourceAccountID != nil {
		id, err := AccountIDFromString(*aux.SourceAccountID)
		if err != nil {
			return err
		}
		o.SourceAccountID = &id
	} else {
		o.SourceAccountID = nil
	}
	if aux.DestinationAccountID != nil {
		id, err := AccountIDFromString(*aux.DestinationAccountID)
		if err != nil {
			return err
		}
		o.DestinationAccountID = &id
	} else {
		o.DestinationAccountID = nil
	}
	o.Metadata = aux.Metadata
	o.Adjustments = aux.Adjustments

	return nil
}

// ToPSPOrder converts an Order to a PSPOrder for sending to the plugin
func ToPSPOrder(order *Order) PSPOrder {
	return PSPOrder{
		Reference:           order.Reference,
		CreatedAt:           order.CreatedAt,
		Direction:           order.Direction,
		SourceAsset:         order.SourceAsset,
		DestinationAsset:         order.DestinationAsset,
		Type:                order.Type,
		Status:              order.Status,
		BaseQuantityOrdered: order.BaseQuantityOrdered,
		BaseQuantityFilled:  order.BaseQuantityFilled,
		LimitPrice:          order.LimitPrice,
		StopPrice:           order.StopPrice,
		TimeInForce:         order.TimeInForce,
		ExpiresAt:           order.ExpiresAt,
		QuoteAmount:         order.QuoteAmount,
		QuoteAsset:          order.QuoteAsset,
		Fee:                 order.Fee,
		FeeAsset:            order.FeeAsset,
		AverageFillPrice:    order.AverageFillPrice,
		PriceAsset:                  order.PriceAsset,
		SourceAccountReference:      order.SourceAccountID.Ref(),
		DestinationAccountReference: order.DestinationAccountID.Ref(),
		Metadata:                    order.Metadata,
		// Order itself does not carry the PSP raw payload — it lives on
		// the adjustments list (each adjustment snapshots the PSP response
		// at that moment). For round-trip purposes (Order → PSPOrder →
		// Validate()), use the latest adjustment's Raw: it represents the
		// order's current state.
		Raw: latestAdjustmentRaw(order.Adjustments),
	}
}

// latestAdjustmentRaw returns the Raw payload of the most recent
// adjustment by CreatedAt. FromPSPOrderToOrder appends at least one
// adjustment on every PSP observation, so a well-formed Order has a
// non-empty list.
func latestAdjustmentRaw(adjustments []OrderAdjustment) json.RawMessage {
	if len(adjustments) == 0 {
		return nil
	}
	latest := adjustments[0]
	for _, a := range adjustments[1:] {
		if a.CreatedAt.After(latest.CreatedAt) {
			latest = a
		}
	}
	return latest.Raw
}

// FromPSPOrderToOrder converts a PSPOrder to an Order.
// observedAt should be workflow.Now() in production (Temporal deterministic clock).
func FromPSPOrderToOrder(from PSPOrder, connectorID ConnectorID, observedAt time.Time) (Order, error) {
	if err := from.Validate(); err != nil {
		return Order{}, err
	}

	o := Order{
		ID: OrderID{
			Reference:   from.Reference,
			ConnectorID: connectorID,
		},
		ConnectorID:          connectorID,
		Reference:            from.Reference,
		ClientOrderID:        from.ClientOrderID,
		CreatedAt:            from.CreatedAt,
		UpdatedAt:            observedAt,
		Direction:            from.Direction,
		SourceAsset:          from.SourceAsset,
		DestinationAsset:     from.DestinationAsset,
		Type:                 from.Type,
		Status:               from.Status,
		BaseQuantityOrdered:  from.BaseQuantityOrdered,
		BaseQuantityFilled:   from.BaseQuantityFilled,
		LimitPrice:           from.LimitPrice,
		StopPrice:            from.StopPrice,
		TimeInForce:          from.TimeInForce,
		ExpiresAt:            from.ExpiresAt,
		QuoteAmount:          from.QuoteAmount,
		QuoteAsset:           from.QuoteAsset,
		Fee:                  from.Fee,
		FeeAsset:             from.FeeAsset,
		AverageFillPrice:     from.AverageFillPrice,
		PriceAsset:           from.PriceAsset,
		SourceAccountID:      NewAccountID(from.SourceAccountReference, connectorID),
		DestinationAccountID: NewAccountID(from.DestinationAccountReference, connectorID),
		Metadata:             from.Metadata,
	}

	o.Adjustments = append(o.Adjustments, FromPSPOrderToOrderAdjustment(from, connectorID, observedAt))

	return o, nil
}

// FromPSPOrders converts a slice of PSPOrders to Orders.
// observedAt should be workflow.Now() in production (Temporal deterministic clock).
func FromPSPOrders(from []PSPOrder, connectorID ConnectorID, observedAt time.Time) ([]Order, error) {
	orders := make([]Order, 0, len(from))
	for _, o := range from {
		order, err := FromPSPOrderToOrder(o, connectorID, observedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

// OrderAdjustment represents a status change or update to an order
type OrderAdjustment struct {
	// Unique adjustment ID
	ID OrderAdjustmentID `json:"id"`

	// Reference from PSP
	Reference string `json:"reference"`

	// Adjustment creation time
	CreatedAt time.Time `json:"createdAt"`

	// Status at this adjustment
	Status OrderStatus `json:"status"`

	// Base quantity filled at this point
	BaseQuantityFilled *big.Int `json:"baseQuantityFilled,omitempty"`

	// Fee at this point
	Fee *big.Int `json:"fee,omitempty"`

	// Fee asset
	FeeAsset *string `json:"feeAsset,omitempty"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`

	// Raw PSP response
	Raw json.RawMessage `json:"raw"`
}

func (oa *OrderAdjustment) IdempotencyKey() string {
	return IdempotencyKey(oa.ID)
}

func (oa OrderAdjustment) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                 string            `json:"id"`
		Reference          string            `json:"reference"`
		CreatedAt          time.Time         `json:"createdAt"`
		Status             OrderStatus       `json:"status"`
		BaseQuantityFilled *big.Int          `json:"baseQuantityFilled,omitempty"`
		Fee                *big.Int          `json:"fee,omitempty"`
		FeeAsset           *string           `json:"feeAsset,omitempty"`
		Metadata           map[string]string `json:"metadata"`
		Raw                json.RawMessage   `json:"raw"`
	}{
		ID:                 oa.ID.String(),
		Reference:          oa.Reference,
		CreatedAt:          oa.CreatedAt,
		Status:             oa.Status,
		BaseQuantityFilled: oa.BaseQuantityFilled,
		Fee:                oa.Fee,
		FeeAsset:           oa.FeeAsset,
		Metadata:           oa.Metadata,
		Raw:                oa.Raw,
	})
}

func (oa *OrderAdjustment) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                 string            `json:"id"`
		Reference          string            `json:"reference"`
		CreatedAt          time.Time         `json:"createdAt"`
		Status             OrderStatus       `json:"status"`
		BaseQuantityFilled *big.Int          `json:"baseQuantityFilled,omitempty"`
		Fee                *big.Int          `json:"fee,omitempty"`
		FeeAsset           *string           `json:"feeAsset,omitempty"`
		Metadata           map[string]string `json:"metadata"`
		Raw                json.RawMessage   `json:"raw"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := OrderAdjustmentIDFromString(aux.ID)
	if err != nil {
		return err
	}

	oa.ID = id
	oa.Reference = aux.Reference
	oa.CreatedAt = aux.CreatedAt
	oa.Status = aux.Status
	oa.BaseQuantityFilled = aux.BaseQuantityFilled
	oa.Fee = aux.Fee
	oa.FeeAsset = aux.FeeAsset
	oa.Metadata = aux.Metadata
	oa.Raw = aux.Raw

	return nil
}

// OrderAdjustmentID uniquely identifies an order adjustment.
// CreatedAt is intentionally excluded — it represents observation time (variable),
// not identity. Mutable fields (BaseQuantityFilled, Fee, FeeAsset) are included
// so that same-status changes (e.g., PARTIALLY_FILLED at different fill levels)
// produce distinct adjustments.
type OrderAdjustmentID struct {
	OrderID            OrderID
	Reference          string
	Status             OrderStatus
	BaseQuantityFilled *big.Int
	Fee                *big.Int
	FeeAsset           *string
}

func (oaid OrderAdjustmentID) String() string {
	data, err := canonicaljson.Marshal(oaid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func OrderAdjustmentIDFromString(value string) (OrderAdjustmentID, error) {
	ret := OrderAdjustmentID{}
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		return ret, err
	}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

func (oaid OrderAdjustmentID) Value() (driver.Value, error) {
	return oaid.String(), nil
}

func (oaid *OrderAdjustmentID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("order adjustment id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {
		if v, ok := s.(string); ok {
			id, err := OrderAdjustmentIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse order adjustment id %s: %v", v, err)
			}
			*oaid = id
			return nil
		}
	}

	return fmt.Errorf("failed to scan order adjustment id: %v", value)
}

// FromPSPOrderToOrderAdjustment creates an OrderAdjustment from a PSPOrder.
// observedAt should be workflow.Now() in production (Temporal deterministic clock).
func FromPSPOrderToOrderAdjustment(from PSPOrder, connectorID ConnectorID, observedAt time.Time) OrderAdjustment {
	orderID := OrderID{
		Reference:   from.Reference,
		ConnectorID: connectorID,
	}

	return OrderAdjustment{
		ID: OrderAdjustmentID{
			OrderID:            orderID,
			Reference:          from.Reference,
			Status:             from.Status,
			BaseQuantityFilled: from.BaseQuantityFilled,
			Fee:                from.Fee,
			FeeAsset:           from.FeeAsset,
		},
		Reference:          from.Reference,
		CreatedAt:          observedAt,
		Status:             from.Status,
		BaseQuantityFilled: from.BaseQuantityFilled,
		Fee:                from.Fee,
		FeeAsset:           from.FeeAsset,
		Metadata:           from.Metadata,
		Raw:                from.Raw,
	}
}

// OrderExpanded includes order with its current status
type OrderExpanded struct {
	Order  Order
	Status OrderStatus
	Error  error
}

func (oe OrderExpanded) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                  string            `json:"id"`
		ConnectorID         string            `json:"connectorID"`
		Provider            string            `json:"provider"`
		Reference           string            `json:"reference"`
		ClientOrderID       string            `json:"clientOrderID,omitempty"`
		CreatedAt           time.Time         `json:"createdAt"`
		UpdatedAt           time.Time         `json:"updatedAt"`
		Direction           OrderDirection    `json:"direction"`
		SourceAsset         string            `json:"sourceAsset"`
		DestinationAsset         string            `json:"destinationAsset"`
		Type                OrderType         `json:"type"`
		Status              string            `json:"status"`
		BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
		LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
		StopPrice           *big.Int          `json:"stopPrice,omitempty"`
		TimeInForce         TimeInForce       `json:"timeInForce"`
		ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
		QuoteAmount         *big.Int          `json:"quoteAmount,omitempty"`
		QuoteAsset          string            `json:"quoteAsset,omitempty"`
		Fee                 *big.Int          `json:"fee,omitempty"`
		FeeAsset            *string           `json:"feeAsset,omitempty"`
		AverageFillPrice    *big.Int          `json:"averageFillPrice,omitempty"`
		PriceAsset           *string           `json:"priceAsset,omitempty"`
		SourceAccountID      *string           `json:"sourceAccountID"`
		DestinationAccountID *string           `json:"destinationAccountID"`
		Metadata             map[string]string `json:"metadata"`
		Adjustments          []OrderAdjustment `json:"adjustments"`
		Error                *string           `json:"error,omitempty"`
	}{
		ID:                   oe.Order.ID.String(),
		ConnectorID:          oe.Order.ConnectorID.String(),
		Provider:             ToV3Provider(oe.Order.ConnectorID.Provider),
		Reference:            oe.Order.Reference,
		ClientOrderID:        oe.Order.ClientOrderID,
		CreatedAt:            oe.Order.CreatedAt,
		UpdatedAt:            oe.Order.UpdatedAt,
		Direction:            oe.Order.Direction,
		SourceAsset:          oe.Order.SourceAsset,
		DestinationAsset:     oe.Order.DestinationAsset,
		Type:                oe.Order.Type,
		Status:              oe.Status.String(),
		BaseQuantityOrdered: oe.Order.BaseQuantityOrdered,
		BaseQuantityFilled:  oe.Order.BaseQuantityFilled,
		LimitPrice:          oe.Order.LimitPrice,
		StopPrice:           oe.Order.StopPrice,
		TimeInForce:         oe.Order.TimeInForce,
		ExpiresAt:           oe.Order.ExpiresAt,
		QuoteAmount:         oe.Order.QuoteAmount,
		QuoteAsset:          oe.Order.QuoteAsset,
		Fee:                 oe.Order.Fee,
		FeeAsset:            oe.Order.FeeAsset,
		AverageFillPrice:    oe.Order.AverageFillPrice,
		PriceAsset:           oe.Order.PriceAsset,
		SourceAccountID:      oe.Order.SourceAccountID.StringPtr(),
		DestinationAccountID: oe.Order.DestinationAccountID.StringPtr(),
		Metadata:             oe.Order.Metadata,
		Adjustments:          oe.Order.Adjustments,
		Error: func() *string {
			if oe.Error == nil {
				return nil
			}
			return pointer.For(oe.Error.Error())
		}(),
	})
}
