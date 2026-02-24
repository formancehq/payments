package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type ConversionStatus int

const (
	CONVERSION_STATUS_UNKNOWN ConversionStatus = iota
	CONVERSION_STATUS_PENDING
	CONVERSION_STATUS_COMPLETED
	CONVERSION_STATUS_FAILED
)

func (s ConversionStatus) String() string {
	switch s {
	case CONVERSION_STATUS_PENDING:
		return "PENDING"
	case CONVERSION_STATUS_COMPLETED:
		return "COMPLETED"
	case CONVERSION_STATUS_FAILED:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

func ConversionStatusFromString(str string) (ConversionStatus, error) {
	switch str {
	case "PENDING":
		return CONVERSION_STATUS_PENDING, nil
	case "COMPLETED":
		return CONVERSION_STATUS_COMPLETED, nil
	case "FAILED":
		return CONVERSION_STATUS_FAILED, nil
	default:
		return CONVERSION_STATUS_UNKNOWN, fmt.Errorf("unknown conversion status: %s", str)
	}
}

func (s ConversionStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *ConversionStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	var err error
	*s, err = ConversionStatusFromString(str)
	return err
}

func (s ConversionStatus) Value() (driver.Value, error) {
	return s.String(), nil
}

func (s *ConversionStatus) Scan(value interface{}) error {
	if value == nil {
		return errors.New("conversion status is nil")
	}

	str, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert conversion status")
	}

	v, ok := str.(string)
	if !ok {
		return fmt.Errorf("failed to cast conversion status")
	}

	*s, err = ConversionStatusFromString(v)
	return err
}

// IsFinal returns true if the conversion status is a final state
func (s ConversionStatus) IsFinal() bool {
	switch s {
	case CONVERSION_STATUS_COMPLETED, CONVERSION_STATUS_FAILED:
		return true
	default:
		return false
	}
}
