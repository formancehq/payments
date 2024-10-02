package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type PaymentInitiationAdjustmentStatus int

const (
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN = iota
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_ASK_RETRIED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_ASK_REVERSED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_PARTIALLY_REVERSED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED
)

func (s PaymentInitiationAdjustmentStatus) String() string {
	switch s {
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION:
		return "WAITING_FOR_VALIDATION"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING:
		return "PROCESSING"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED:
		return "PROCESSED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED:
		return "FAILED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED:
		return "REJECTED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_ASK_RETRIED:
		return "ASK_RETRIED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_ASK_REVERSED:
		return "ASK_REVERSED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING:
		return "REVERSE_PROCESSING"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED:
		return "REVERSE_FAILED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_PARTIALLY_REVERSED:
		return "PARTIALLY_REVERSED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED:
		return "REVERSED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN:
		return "UNKNOWN"
	}
	return "UNKNOWN"
}

func PaymentInitiationAdjustmentStatusFromString(s string) (PaymentInitiationAdjustmentStatus, error) {
	switch s {
	case "WAITING_FOR_VALIDATION":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION, nil
	case "PROCESSING":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, nil
	case "PROCESSED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, nil
	case "FAILED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED, nil
	case "REJECTED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED, nil
	case "ASK_RETRIED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_ASK_RETRIED, nil
	case "ASK_REVERSED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_ASK_REVERSED, nil
	case "REVERSE_PROCESSING":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING, nil
	case "REVERSE_FAILED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED, nil
	case "PARTIALLY_REVERSED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_PARTIALLY_REVERSED, nil
	case "REVERSED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED, nil
	case "UNKNOWN":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN, nil
	}

	return PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN, fmt.Errorf("unknown PaymentInitiationAdjustmentStatus: %s", s)
}

func (t PaymentInitiationAdjustmentStatus) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, t.String())), nil
}

func (t *PaymentInitiationAdjustmentStatus) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	value, err := PaymentInitiationAdjustmentStatusFromString(v)
	if err != nil {
		return err
	}

	*t = value

	return nil
}

func (t PaymentInitiationAdjustmentStatus) Value() (driver.Value, error) {
	return t.String(), nil
}

func (t *PaymentInitiationAdjustmentStatus) Scan(value interface{}) error {
	if value == nil {
		return errors.New("payment initiation adjusmtent status status is nil")
	}

	s, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert payment initiation adjusmtent status status")
	}

	v, ok := s.(string)
	if !ok {
		return fmt.Errorf("failed to cast payment initiation adjusmtent status status")
	}

	res, err := PaymentInitiationAdjustmentStatusFromString(v)
	if err != nil {
		return err
	}

	*t = res

	return nil
}
