package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type TimeInForce int

const (
	TIME_IN_FORCE_UNKNOWN TimeInForce = iota
	TIME_IN_FORCE_GOOD_UNTIL_CANCELLED
	TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME
	TIME_IN_FORCE_IMMEDIATE_OR_CANCEL
	TIME_IN_FORCE_FILL_OR_KILL
)

func (t TimeInForce) String() string {
	switch t {
	case TIME_IN_FORCE_GOOD_UNTIL_CANCELLED:
		return "GOOD_UNTIL_CANCELLED"
	case TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME:
		return "GOOD_UNTIL_DATE_TIME"
	case TIME_IN_FORCE_IMMEDIATE_OR_CANCEL:
		return "IMMEDIATE_OR_CANCEL"
	case TIME_IN_FORCE_FILL_OR_KILL:
		return "FILL_OR_KILL"
	default:
		return "UNKNOWN"
	}
}

func TimeInForceFromString(s string) (TimeInForce, error) {
	switch s {
	case "GOOD_UNTIL_CANCELLED", "GTC":
		return TIME_IN_FORCE_GOOD_UNTIL_CANCELLED, nil
	case "GOOD_UNTIL_DATE_TIME", "GTD":
		return TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME, nil
	case "IMMEDIATE_OR_CANCEL", "IOC":
		return TIME_IN_FORCE_IMMEDIATE_OR_CANCEL, nil
	case "FILL_OR_KILL", "FOK":
		return TIME_IN_FORCE_FILL_OR_KILL, nil
	default:
		return TIME_IN_FORCE_UNKNOWN, fmt.Errorf("unknown time in force: %s", s)
	}
}

func (t TimeInForce) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TimeInForce) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	var err error
	*t, err = TimeInForceFromString(s)
	return err
}

func (t TimeInForce) Value() (driver.Value, error) {
	return t.String(), nil
}

func (t *TimeInForce) Scan(value interface{}) error {
	if value == nil {
		return errors.New("time in force is nil")
	}

	s, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert time in force")
	}

	v, ok := s.(string)
	if !ok {
		return fmt.Errorf("failed to cast time in force")
	}

	*t, err = TimeInForceFromString(v)
	return err
}
