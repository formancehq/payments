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

// paymentStatusToAdjustmentStatus maps a payment status onto the payment
// initiation adjustment status it implies. The returned error is the one to
// record on the adjustment, not a failure of the mapping itself, and the
// boolean reports whether an adjustment should be produced at all.
func paymentStatusToAdjustmentStatus(status PaymentStatus) (PaymentInitiationAdjustmentStatus, error, bool) {
	switch status {
	case PAYMENT_STATUS_AMOUNT_ADJUSTMENT, PAYMENT_STATUS_UNKNOWN:
		// No need to add an adjustment for this payment initiation
		return 0, nil, false
	case PAYMENT_STATUS_PENDING, PAYMENT_STATUS_AUTHORISATION:
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, nil, true
	case PAYMENT_STATUS_SUCCEEDED,
		PAYMENT_STATUS_CAPTURE,
		PAYMENT_STATUS_REFUND_REVERSED,
		PAYMENT_STATUS_DISPUTE_WON:
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED, nil, true
	case PAYMENT_STATUS_CANCELLED,
		PAYMENT_STATUS_CAPTURE_FAILED,
		PAYMENT_STATUS_EXPIRED,
		PAYMENT_STATUS_FAILED,
		PAYMENT_STATUS_DISPUTE_LOST:
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED, errors.New("payment failed"), true
	case PAYMENT_STATUS_DISPUTE:
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN, nil, true
	case PAYMENT_STATUS_REFUNDED:
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED, nil, true
	case PAYMENT_STATUS_REFUNDED_FAILURE:
		return PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED, errors.New("payment refund failed"), true
	default:
		return 0, nil, false
	}
}

// FromPaymentDataToPaymentInitiationAdjustment builds the adjustment implied by
// a payment status change. Amount, asset and metadata are not carried by the
// status change itself, so they are taken from the payment initiation to keep
// the emitted adjustment event consistent with the ones produced by the
// create/reverse workflows.
func FromPaymentDataToPaymentInitiationAdjustment(status PaymentStatus, createdAt time.Time, pi PaymentInitiation) *PaymentInitiationAdjustment {
	piStatus, err, ok := paymentStatusToAdjustmentStatus(status)
	if !ok {
		return nil
	}

	asset := pi.Asset

	return &PaymentInitiationAdjustment{
		ID: PaymentInitiationAdjustmentID{
			PaymentInitiationID: pi.ID,
			CreatedAt:           createdAt,
			Status:              piStatus,
		},
		CreatedAt: createdAt,
		Status:    piStatus,
		Error:     err,
		Amount:    pi.Amount,
		Asset:     &asset,
		Metadata:  pi.Metadata,
	}
}

// FromPaymentDataToPaymentInitiationAdjustmentFromID builds the adjustment from
// a payment initiation ID alone, leaving amount, asset and metadata unset.
//
// Deprecated: only kept so in-flight 3.0 workflows keep replaying to the same
// result; use FromPaymentDataToPaymentInitiationAdjustment instead.
func FromPaymentDataToPaymentInitiationAdjustmentFromID(status PaymentStatus, createdAt time.Time, piID PaymentInitiationID) *PaymentInitiationAdjustment {
	piStatus, err, ok := paymentStatusToAdjustmentStatus(status)
	if !ok {
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
