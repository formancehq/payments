package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type OrderStatus int

const (
	ORDER_STATUS_UNKNOWN OrderStatus = iota
	ORDER_STATUS_PENDING
	ORDER_STATUS_OPEN
	ORDER_STATUS_PARTIALLY_FILLED
	ORDER_STATUS_FILLED
	ORDER_STATUS_CANCELLED
	ORDER_STATUS_FAILED
	ORDER_STATUS_EXPIRED
)

func (s OrderStatus) String() string {
	switch s {
	case ORDER_STATUS_PENDING:
		return "PENDING"
	case ORDER_STATUS_OPEN:
		return "OPEN"
	case ORDER_STATUS_PARTIALLY_FILLED:
		return "PARTIALLY_FILLED"
	case ORDER_STATUS_FILLED:
		return "FILLED"
	case ORDER_STATUS_CANCELLED:
		return "CANCELLED"
	case ORDER_STATUS_FAILED:
		return "FAILED"
	case ORDER_STATUS_EXPIRED:
		return "EXPIRED"
	default:
		return "UNKNOWN"
	}
}

func OrderStatusFromString(str string) (OrderStatus, error) {
	switch str {
	case "PENDING":
		return ORDER_STATUS_PENDING, nil
	case "OPEN":
		return ORDER_STATUS_OPEN, nil
	case "PARTIALLY_FILLED":
		return ORDER_STATUS_PARTIALLY_FILLED, nil
	case "FILLED":
		return ORDER_STATUS_FILLED, nil
	case "CANCELLED":
		return ORDER_STATUS_CANCELLED, nil
	case "FAILED":
		return ORDER_STATUS_FAILED, nil
	case "EXPIRED":
		return ORDER_STATUS_EXPIRED, nil
	default:
		return ORDER_STATUS_UNKNOWN, fmt.Errorf("unknown order status: %s", str)
	}
}

func (s OrderStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *OrderStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	var err error
	*s, err = OrderStatusFromString(str)
	return err
}

func (s OrderStatus) Value() (driver.Value, error) {
	return s.String(), nil
}

func (s *OrderStatus) Scan(value interface{}) error {
	if value == nil {
		return errors.New("order status is nil")
	}

	str, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert order status")
	}

	v, ok := str.(string)
	if !ok {
		return fmt.Errorf("failed to cast order status")
	}

	*s, err = OrderStatusFromString(v)
	return err
}

// IsFinal returns true if the order status is a final state
func (s OrderStatus) IsFinal() bool {
	switch s {
	case ORDER_STATUS_FILLED, ORDER_STATUS_CANCELLED, ORDER_STATUS_FAILED, ORDER_STATUS_EXPIRED:
		return true
	default:
		return false
	}
}

// CanCancel returns true if an order in this status can be cancelled
func (s OrderStatus) CanCancel() bool {
	switch s {
	case ORDER_STATUS_PENDING, ORDER_STATUS_OPEN, ORDER_STATUS_PARTIALLY_FILLED:
		return true
	default:
		return false
	}
}
