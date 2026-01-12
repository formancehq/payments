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
	TargetAsset string

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

	// Time in force
	TimeInForce TimeInForce

	// Expiration time for GTD orders
	ExpiresAt *time.Time

	// Fee charged for the order (using integer representation)
	Fee *big.Int

	// Fee asset
	FeeAsset *string

	// Average fill price (using integer representation)
	AverageFillPrice *big.Int

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

	if !assets.IsValid(o.TargetAsset) {
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

	// Order creation date
	CreatedAt time.Time `json:"createdAt"`

	// Last update date
	UpdatedAt time.Time `json:"updatedAt"`

	// Order direction: BUY or SELL
	Direction OrderDirection `json:"direction"`

	// Source asset
	SourceAsset string `json:"sourceAsset"`

	// Target asset
	TargetAsset string `json:"targetAsset"`

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

	// Time in force
	TimeInForce TimeInForce `json:"timeInForce"`

	// Expiration time for GTD orders
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	// Fee charged for the order
	Fee *big.Int `json:"fee,omitempty"`

	// Fee asset
	FeeAsset *string `json:"feeAsset,omitempty"`

	// Average fill price
	AverageFillPrice *big.Int `json:"averageFillPrice,omitempty"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`

	// Related adjustments (status changes)
	Adjustments []OrderAdjustment `json:"adjustments"`
}

func (o *Order) IdempotencyKey() string {
	return IdempotencyKey(o.ID)
}

func (o Order) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                  string            `json:"id"`
		ConnectorID         string            `json:"connectorID"`
		Provider            string            `json:"provider"`
		Reference           string            `json:"reference"`
		CreatedAt           time.Time         `json:"createdAt"`
		UpdatedAt           time.Time         `json:"updatedAt"`
		Direction           OrderDirection    `json:"direction"`
		SourceAsset         string            `json:"sourceAsset"`
		TargetAsset         string            `json:"targetAsset"`
		Type                OrderType         `json:"type"`
		Status              OrderStatus       `json:"status"`
		BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
		LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
		TimeInForce         TimeInForce       `json:"timeInForce"`
		ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
		Fee                 *big.Int          `json:"fee,omitempty"`
		FeeAsset            *string           `json:"feeAsset,omitempty"`
		AverageFillPrice    *big.Int          `json:"averageFillPrice,omitempty"`
		Metadata            map[string]string `json:"metadata"`
		Adjustments         []OrderAdjustment `json:"adjustments"`
	}{
		ID:                  o.ID.String(),
		ConnectorID:         o.ConnectorID.String(),
		Provider:            ToV3Provider(o.ConnectorID.Provider),
		Reference:           o.Reference,
		CreatedAt:           o.CreatedAt,
		UpdatedAt:           o.UpdatedAt,
		Direction:           o.Direction,
		SourceAsset:         o.SourceAsset,
		TargetAsset:         o.TargetAsset,
		Type:                o.Type,
		Status:              o.Status,
		BaseQuantityOrdered: o.BaseQuantityOrdered,
		BaseQuantityFilled:  o.BaseQuantityFilled,
		LimitPrice:          o.LimitPrice,
		TimeInForce:         o.TimeInForce,
		ExpiresAt:           o.ExpiresAt,
		Fee:                 o.Fee,
		FeeAsset:            o.FeeAsset,
		AverageFillPrice:    o.AverageFillPrice,
		Metadata:            o.Metadata,
		Adjustments:         o.Adjustments,
	})
}

func (o *Order) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                  string            `json:"id"`
		ConnectorID         string            `json:"connectorID"`
		Reference           string            `json:"reference"`
		CreatedAt           time.Time         `json:"createdAt"`
		UpdatedAt           time.Time         `json:"updatedAt"`
		Direction           OrderDirection    `json:"direction"`
		SourceAsset         string            `json:"sourceAsset"`
		TargetAsset         string            `json:"targetAsset"`
		Type                OrderType         `json:"type"`
		Status              OrderStatus       `json:"status"`
		BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
		LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
		TimeInForce         TimeInForce       `json:"timeInForce"`
		ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
		Fee                 *big.Int          `json:"fee,omitempty"`
		FeeAsset            *string           `json:"feeAsset,omitempty"`
		AverageFillPrice    *big.Int          `json:"averageFillPrice,omitempty"`
		Metadata            map[string]string `json:"metadata"`
		Adjustments         []OrderAdjustment `json:"adjustments"`
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
	o.CreatedAt = aux.CreatedAt
	o.UpdatedAt = aux.UpdatedAt
	o.Direction = aux.Direction
	o.SourceAsset = aux.SourceAsset
	o.TargetAsset = aux.TargetAsset
	o.Type = aux.Type
	o.Status = aux.Status
	o.BaseQuantityOrdered = aux.BaseQuantityOrdered
	o.BaseQuantityFilled = aux.BaseQuantityFilled
	o.LimitPrice = aux.LimitPrice
	o.TimeInForce = aux.TimeInForce
	o.ExpiresAt = aux.ExpiresAt
	o.Fee = aux.Fee
	o.FeeAsset = aux.FeeAsset
	o.AverageFillPrice = aux.AverageFillPrice
	o.Metadata = aux.Metadata
	o.Adjustments = aux.Adjustments

	return nil
}

// FromPSPOrderToOrder converts a PSPOrder to an Order
func FromPSPOrderToOrder(from PSPOrder, connectorID ConnectorID) (Order, error) {
	if err := from.Validate(); err != nil {
		return Order{}, err
	}

	now := time.Now().UTC()
	o := Order{
		ID: OrderID{
			Reference:   from.Reference,
			ConnectorID: connectorID,
		},
		ConnectorID:         connectorID,
		Reference:           from.Reference,
		CreatedAt:           from.CreatedAt,
		UpdatedAt:           now,
		Direction:           from.Direction,
		SourceAsset:         from.SourceAsset,
		TargetAsset:         from.TargetAsset,
		Type:                from.Type,
		Status:              from.Status,
		BaseQuantityOrdered: from.BaseQuantityOrdered,
		BaseQuantityFilled:  from.BaseQuantityFilled,
		LimitPrice:          from.LimitPrice,
		TimeInForce:         from.TimeInForce,
		ExpiresAt:           from.ExpiresAt,
		Fee:                 from.Fee,
		FeeAsset:            from.FeeAsset,
		AverageFillPrice:    from.AverageFillPrice,
		Metadata:            from.Metadata,
	}

	o.Adjustments = append(o.Adjustments, FromPSPOrderToOrderAdjustment(from, connectorID))

	return o, nil
}

// FromPSPOrders converts a slice of PSPOrders to Orders
func FromPSPOrders(from []PSPOrder, connectorID ConnectorID) ([]Order, error) {
	orders := make([]Order, 0, len(from))
	for _, o := range from {
		order, err := FromPSPOrderToOrder(o, connectorID)
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
	}{
		ID:                 oa.ID.String(),
		Reference:          oa.Reference,
		CreatedAt:          oa.CreatedAt,
		Status:             oa.Status,
		BaseQuantityFilled: oa.BaseQuantityFilled,
		Fee:                oa.Fee,
		FeeAsset:           oa.FeeAsset,
		Metadata:           oa.Metadata,
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

	return nil
}

// OrderAdjustmentID uniquely identifies an order adjustment
type OrderAdjustmentID struct {
	OrderID   OrderID
	Reference string
	CreatedAt time.Time
	Status    OrderStatus
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

// FromPSPOrderToOrderAdjustment creates an OrderAdjustment from a PSPOrder
func FromPSPOrderToOrderAdjustment(from PSPOrder, connectorID ConnectorID) OrderAdjustment {
	orderID := OrderID{
		Reference:   from.Reference,
		ConnectorID: connectorID,
	}

	return OrderAdjustment{
		ID: OrderAdjustmentID{
			OrderID:   orderID,
			Reference: from.Reference,
			CreatedAt: from.CreatedAt,
			Status:    from.Status,
		},
		Reference:          from.Reference,
		CreatedAt:          from.CreatedAt,
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
		CreatedAt           time.Time         `json:"createdAt"`
		UpdatedAt           time.Time         `json:"updatedAt"`
		Direction           OrderDirection    `json:"direction"`
		SourceAsset         string            `json:"sourceAsset"`
		TargetAsset         string            `json:"targetAsset"`
		Type                OrderType         `json:"type"`
		Status              string            `json:"status"`
		BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled"`
		LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
		TimeInForce         TimeInForce       `json:"timeInForce"`
		ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
		Fee                 *big.Int          `json:"fee,omitempty"`
		FeeAsset            *string           `json:"feeAsset,omitempty"`
		AverageFillPrice    *big.Int          `json:"averageFillPrice,omitempty"`
		Metadata            map[string]string `json:"metadata"`
		Error               *string           `json:"error,omitempty"`
	}{
		ID:                  oe.Order.ID.String(),
		ConnectorID:         oe.Order.ConnectorID.String(),
		Provider:            ToV3Provider(oe.Order.ConnectorID.Provider),
		Reference:           oe.Order.Reference,
		CreatedAt:           oe.Order.CreatedAt,
		UpdatedAt:           oe.Order.UpdatedAt,
		Direction:           oe.Order.Direction,
		SourceAsset:         oe.Order.SourceAsset,
		TargetAsset:         oe.Order.TargetAsset,
		Type:                oe.Order.Type,
		Status:              oe.Status.String(),
		BaseQuantityOrdered: oe.Order.BaseQuantityOrdered,
		BaseQuantityFilled:  oe.Order.BaseQuantityFilled,
		LimitPrice:          oe.Order.LimitPrice,
		TimeInForce:         oe.Order.TimeInForce,
		ExpiresAt:           oe.Order.ExpiresAt,
		Fee:                 oe.Order.Fee,
		FeeAsset:            oe.Order.FeeAsset,
		AverageFillPrice:    oe.Order.AverageFillPrice,
		Metadata:            oe.Order.Metadata,
		Error: func() *string {
			if oe.Error == nil {
				return nil
			}
			return pointer.For(oe.Error.Error())
		}(),
	})
}
