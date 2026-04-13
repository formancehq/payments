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
	ORDER_TYPE_STOP_LIMIT
	ORDER_TYPE_STOP
	ORDER_TYPE_TWAP
	ORDER_TYPE_VWAP
	ORDER_TYPE_PEG
	ORDER_TYPE_BLOCK
	ORDER_TYPE_RFQ
	ORDER_TYPE_TRAILING_STOP
	ORDER_TYPE_TRAILING_STOP_LIMIT
	ORDER_TYPE_TAKE_PROFIT
	ORDER_TYPE_TAKE_PROFIT_LIMIT
	ORDER_TYPE_LIMIT_MAKER
)

func (t OrderType) String() string {
	switch t {
	case ORDER_TYPE_MARKET:
		return "MARKET"
	case ORDER_TYPE_LIMIT:
		return "LIMIT"
	case ORDER_TYPE_STOP_LIMIT:
		return "STOP_LIMIT"
	case ORDER_TYPE_STOP:
		return "STOP"
	case ORDER_TYPE_TWAP:
		return "TWAP"
	case ORDER_TYPE_VWAP:
		return "VWAP"
	case ORDER_TYPE_PEG:
		return "PEG"
	case ORDER_TYPE_BLOCK:
		return "BLOCK"
	case ORDER_TYPE_RFQ:
		return "RFQ"
	case ORDER_TYPE_TRAILING_STOP:
		return "TRAILING_STOP"
	case ORDER_TYPE_TRAILING_STOP_LIMIT:
		return "TRAILING_STOP_LIMIT"
	case ORDER_TYPE_TAKE_PROFIT:
		return "TAKE_PROFIT"
	case ORDER_TYPE_TAKE_PROFIT_LIMIT:
		return "TAKE_PROFIT_LIMIT"
	case ORDER_TYPE_LIMIT_MAKER:
		return "LIMIT_MAKER"
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
	case "STOP_LIMIT":
		return ORDER_TYPE_STOP_LIMIT, nil
	case "STOP":
		return ORDER_TYPE_STOP, nil
	case "TWAP":
		return ORDER_TYPE_TWAP, nil
	case "VWAP":
		return ORDER_TYPE_VWAP, nil
	case "PEG":
		return ORDER_TYPE_PEG, nil
	case "BLOCK":
		return ORDER_TYPE_BLOCK, nil
	case "RFQ":
		return ORDER_TYPE_RFQ, nil
	case "TRAILING_STOP":
		return ORDER_TYPE_TRAILING_STOP, nil
	case "TRAILING_STOP_LIMIT":
		return ORDER_TYPE_TRAILING_STOP_LIMIT, nil
	case "TAKE_PROFIT":
		return ORDER_TYPE_TAKE_PROFIT, nil
	case "TAKE_PROFIT_LIMIT":
		return ORDER_TYPE_TAKE_PROFIT_LIMIT, nil
	case "LIMIT_MAKER":
		return ORDER_TYPE_LIMIT_MAKER, nil
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
	res := t.String()
	if res == "UNKNOWN" {
		return nil, fmt.Errorf("unknown order type")
	}
	return res, nil
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
