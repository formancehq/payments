package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type PaymentInitiationReversalAdjustmentStatus int

const (
	PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN = iota
	PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING
	PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED
	PAYMENT_INITIATION_REVERSAL_STATUS_FAILED
)

func (s PaymentInitiationReversalAdjustmentStatus) String() string {
	switch s {
	case PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING:
		return "PROCESSING"
	case PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED:
		return "PROCESSED"
	case PAYMENT_INITIATION_REVERSAL_STATUS_FAILED:
		return "FAILED"
	case PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN:
		return "UNKNOWN"
	}
	return "UNKNOWN"
}

func PaymentInitiationReversalStatusFromString(s string) (PaymentInitiationReversalAdjustmentStatus, error) {
	switch s {
	case "PROCESSING":
		return PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSING, nil
	case "PROCESSED":
		return PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED, nil
	case "FAILED":
		return PAYMENT_INITIATION_REVERSAL_STATUS_FAILED, nil
	}
	return PAYMENT_INITIATION_REVERSAL_STATUS_UNKNOWN, nil
}

func (t PaymentInitiationReversalAdjustmentStatus) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, t.String())), nil
}

func (t *PaymentInitiationReversalAdjustmentStatus) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	value, err := PaymentInitiationReversalStatusFromString(v)
	if err != nil {
		return err
	}

	*t = value

	return nil
}

func (t PaymentInitiationReversalAdjustmentStatus) Value() (driver.Value, error) {
	return t.String(), nil
}

func (t *PaymentInitiationReversalAdjustmentStatus) Scan(value interface{}) error {
	if value == nil {
		return errors.New("payment initiation reversal status status is nil")
	}

	s, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert payment initiation reversal status status")
	}

	v, ok := s.(string)
	if !ok {
		return fmt.Errorf("failed to cast payment initiation reversal status status")
	}

	res, err := PaymentInitiationReversalStatusFromString(v)
	if err != nil {
		return err
	}

	*t = res

	return nil
}
