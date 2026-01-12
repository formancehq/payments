package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type OrderDirection int

const (
	ORDER_DIRECTION_UNKNOWN OrderDirection = iota
	ORDER_DIRECTION_BUY
	ORDER_DIRECTION_SELL
)

func (d OrderDirection) String() string {
	switch d {
	case ORDER_DIRECTION_BUY:
		return "BUY"
	case ORDER_DIRECTION_SELL:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

func OrderDirectionFromString(s string) (OrderDirection, error) {
	switch s {
	case "BUY":
		return ORDER_DIRECTION_BUY, nil
	case "SELL":
		return ORDER_DIRECTION_SELL, nil
	default:
		return ORDER_DIRECTION_UNKNOWN, fmt.Errorf("unknown order direction: %s", s)
	}
}

func (d OrderDirection) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *OrderDirection) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	var err error
	*d, err = OrderDirectionFromString(s)
	return err
}

func (d OrderDirection) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *OrderDirection) Scan(value interface{}) error {
	if value == nil {
		return errors.New("order direction is nil")
	}

	s, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert order direction")
	}

	v, ok := s.(string)
	if !ok {
		return fmt.Errorf("failed to cast order direction")
	}

	*d, err = OrderDirectionFromString(v)
	return err
}
