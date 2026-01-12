package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type OrderType int

const (
	ORDER_TYPE_UNKNOWN OrderType = iota
	ORDER_TYPE_MARKET
	ORDER_TYPE_LIMIT
)

func (t OrderType) String() string {
	switch t {
	case ORDER_TYPE_MARKET:
		return "MARKET"
	case ORDER_TYPE_LIMIT:
		return "LIMIT"
	default:
		return "UNKNOWN"
	}
}

func OrderTypeFromString(s string) (OrderType, error) {
	switch s {
	case "MARKET":
		return ORDER_TYPE_MARKET, nil
	case "LIMIT":
		return ORDER_TYPE_LIMIT, nil
	default:
		return ORDER_TYPE_UNKNOWN, fmt.Errorf("unknown order type: %s", s)
	}
}

func (t OrderType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *OrderType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	var err error
	*t, err = OrderTypeFromString(s)
	return err
}

func (t OrderType) Value() (driver.Value, error) {
	return t.String(), nil
}

func (t *OrderType) Scan(value interface{}) error {
	if value == nil {
		return errors.New("order type is nil")
	}

	s, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert order type")
	}

	v, ok := s.(string)
	if !ok {
		return fmt.Errorf("failed to cast order type")
	}

	*t, err = OrderTypeFromString(v)
	return err
}
