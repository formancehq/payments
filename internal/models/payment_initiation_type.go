package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type PaymentInitiationType int

const (
	PAYMENT_INITIATION_TYPE_UNKNOWN PaymentInitiationType = iota
	PAYMENT_INITIATION_TYPE_TRANSFER
	PAYMENT_INITIATION_TYPE_PAYOUT
)

func (t PaymentInitiationType) String() string {
	switch t {
	case PAYMENT_INITIATION_TYPE_UNKNOWN:
		return "UNKNOWN"
	case PAYMENT_INITIATION_TYPE_TRANSFER:
		return "TRANSFER"
	case PAYMENT_INITIATION_TYPE_PAYOUT:
		return "PAYOUT"
	default:
		return "UNKNOWN"
	}
}

func PaymentInitiationTypeFromString(value string) (PaymentInitiationType, error) {
	switch value {
	case "TRANSFER":
		return PAYMENT_INITIATION_TYPE_TRANSFER, nil
	case "PAYOUT":
		return PAYMENT_INITIATION_TYPE_PAYOUT, nil
	case "UNKNOWN":
		return PAYMENT_INITIATION_TYPE_UNKNOWN, nil
	default:
		return PAYMENT_INITIATION_TYPE_UNKNOWN, errors.New("invalid payment initiation type value")
	}
}

func MustPaymentInitiationTypeFromString(value string) PaymentInitiationType {
	ret, err := PaymentInitiationTypeFromString(value)
	if err != nil {
		panic(err)
	}
	return ret
}

func (t PaymentInitiationType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, t.String())), nil
}

func (t *PaymentInitiationType) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	value, err := PaymentInitiationTypeFromString(v)
	if err != nil {
		return err
	}

	*t = value

	return nil
}

func (t PaymentInitiationType) Value() (driver.Value, error) {
	return t.String(), nil
}

func (t *PaymentInitiationType) Scan(value interface{}) error {
	if value == nil {
		return errors.New("payment status is nil")
	}

	s, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert payment status")
	}

	v, ok := s.(string)
	if !ok {
		return fmt.Errorf("failed to cast payment status")
	}

	res, err := PaymentInitiationTypeFromString(v)
	if err != nil {
		return err
	}

	*t = res

	return nil
}
