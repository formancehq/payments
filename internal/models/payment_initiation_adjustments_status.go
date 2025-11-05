package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type PaymentInitiationAdjustmentStatus int

const (
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN PaymentInitiationAdjustmentStatus = iota
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED
	PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING
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
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING:
		return "REVERSE_PROCESSING"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED:
		return "REVERSE_FAILED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED:
		return "REVERSED"
	case PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING:
		return "SCHEDULED_FOR_PROCESSING"
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
	case "REVERSE_PROCESSING":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING, nil
	case "REVERSE_FAILED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED, nil
	case "REVERSED":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED, nil
	case "SCHEDULED_FOR_PROCESSING":
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING, nil
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
		return errors.New("payment initiation adjustment status is nil")
	}

	s, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert payment initiation adjustment status")
	}

	v, ok := s.(string)
	if !ok {
		return fmt.Errorf("failed to cast payment initiation adjustment status")
	}

	res, err := PaymentInitiationAdjustmentStatusFromString(v)
	if err != nil {
		return err
	}

	*t = res

	return nil
}

func FromPaymentToPaymentInitiationAdjustment(from *Payment, piID PaymentInitiationID) *PaymentInitiationAdjustment {
	var status PaymentInitiationAdjustmentStatus
	var err error

	switch from.Status {
	case PAYMENT_STATUS_AMOUNT_ADJUSTMENT, PAYMENT_STATUS_UNKNOWN:
		// No need to add an adjustment for this payment initiation
		return nil
	case PAYMENT_STATUS_PENDING, PAYMENT_STATUS_AUTHORISATION:
		status = PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING
	case PAYMENT_STATUS_SUCCEEDED,
		PAYMENT_STATUS_CAPTURE,
		PAYMENT_STATUS_REFUND_REVERSED,
		PAYMENT_STATUS_DISPUTE_WON:
		status = PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED
	case PAYMENT_STATUS_CANCELLED,
		PAYMENT_STATUS_CAPTURE_FAILED,
		PAYMENT_STATUS_EXPIRED,
		PAYMENT_STATUS_FAILED,
		PAYMENT_STATUS_DISPUTE_LOST:
		status = PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED
		err = errors.New("payment failed")
	case PAYMENT_STATUS_DISPUTE:
		status = PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN
	case PAYMENT_STATUS_REFUNDED:
		status = PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED
	case PAYMENT_STATUS_REFUNDED_FAILURE:
		status = PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED
		err = errors.New("payment refund failed")
	default:
		return nil
	}

	return &PaymentInitiationAdjustment{
		ID: PaymentInitiationAdjustmentID{
			PaymentInitiationID: piID,
			CreatedAt:           from.CreatedAt,
			Status:              status,
		},
		CreatedAt: from.CreatedAt,
		Status:    status,
		Error:     err,
	}
}

func FromPaymentDataToPaymentInitiationAdjustment(status PaymentStatus, createdAt time.Time, piID PaymentInitiationID) *PaymentInitiationAdjustment {
	var piStatus PaymentInitiationAdjustmentStatus
	var err error

	switch status {
	case PAYMENT_STATUS_AMOUNT_ADJUSTMENT, PAYMENT_STATUS_UNKNOWN:
		// No need to add an adjustment for this payment initiation
		return nil
	case PAYMENT_STATUS_PENDING, PAYMENT_STATUS_AUTHORISATION:
		piStatus = PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING
	case PAYMENT_STATUS_SUCCEEDED,
		PAYMENT_STATUS_CAPTURE,
		PAYMENT_STATUS_REFUND_REVERSED,
		PAYMENT_STATUS_DISPUTE_WON:
		piStatus = PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED
	case PAYMENT_STATUS_CANCELLED,
		PAYMENT_STATUS_CAPTURE_FAILED,
		PAYMENT_STATUS_EXPIRED,
		PAYMENT_STATUS_FAILED,
		PAYMENT_STATUS_DISPUTE_LOST:
		piStatus = PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED
		err = errors.New("payment failed")
	case PAYMENT_STATUS_DISPUTE:
		piStatus = PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN
	case PAYMENT_STATUS_REFUNDED:
		piStatus = PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED
	case PAYMENT_STATUS_REFUNDED_FAILURE:
		piStatus = PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED
		err = errors.New("payment refund failed")
	default:
		return nil
	}

	return &PaymentInitiationAdjustment{
		ID: PaymentInitiationAdjustmentID{
			PaymentInitiationID: piID,
			CreatedAt:           createdAt,
			Status:              piStatus,
		},
		CreatedAt: createdAt,
		Status:    piStatus,
		Error:     err,
	}
}
