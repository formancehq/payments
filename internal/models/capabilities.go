package models

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type Capability int

const (
	CAPABILITY_FETCH_UNKNOWN Capability = iota

	// CAPABILITY_FETCH_X indicates that the connector can fetch the object X
	CAPABILITY_FETCH_ACCOUNTS
	CAPABILITY_FETCH_BALANCES
	CAPABILITY_FETCH_EXTERNAL_ACCOUNTS
	CAPABILITY_FETCH_PAYMENTS
	CAPABILITY_FETCH_OTHERS

	// Webhooks capabilities indicates that the connector can create, manage and
	// receive webhooks from the connector
	CAPABILITY_CREATE_WEBHOOKS
	CAPABILITY_TRANSLATE_WEBHOOKS

	// Creation capabilities indicates that the connector supports the creation
	// of the object
	CAPABILITY_CREATE_BANK_ACCOUNT
	CAPABILITY_CREATE_TRANSFER
	CAPABILITY_CREATE_PAYOUT

	// Thanks to the formance API, we can create formance object of an account
	// and a payment without sending anything to the connector.
	// It can be useful for testing, but also for the generic connector if the
	// user don't want to use our platform to connect directly to their PSP, but
	// still want us to record the accounts and payments.
	CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION
	CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION
)

func (t Capability) String() string {
	switch t {
	case CAPABILITY_FETCH_ACCOUNTS:
		return "FETCH_ACCOUNTS"
	case CAPABILITY_FETCH_BALANCES:
		return "FETCH_BALANCES"
	case CAPABILITY_FETCH_EXTERNAL_ACCOUNTS:
		return "FETCH_EXTERNAL_ACCOUNTS"
	case CAPABILITY_FETCH_PAYMENTS:
		return "FETCH_PAYMENTS"
	case CAPABILITY_FETCH_OTHERS:
		return "FETCH_OTHERS"

	case CAPABILITY_CREATE_WEBHOOKS:
		return "CREATE_WEBHOOKS"
	case CAPABILITY_TRANSLATE_WEBHOOKS:
		return "TRANSLATE_WEBHOOKS"

	case CAPABILITY_CREATE_BANK_ACCOUNT:
		return "CREATE_BANK_ACCOUNT"
	case CAPABILITY_CREATE_TRANSFER:
		return "CREATE_TRANSFER"
	case CAPABILITY_CREATE_PAYOUT:
		return "CREATE_PAYOUT"

	case CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION:
		return "ALLOW_FORMANCE_ACCOUNT_CREATION"
	case CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION:
		return "ALLOW_FORMANCE_PAYMENT_CREATION"

	default:
		return "UNKNOWN"
	}
}

func (t Capability) Value() (driver.Value, error) {
	res := t.String()
	if res == "UNKNOWN" {
		return nil, fmt.Errorf("unknown capability")
	}
	return res, nil
}

func (t *Capability) Scan(value interface{}) error {
	if value == nil {
		return errors.New("capability is nil")
	}

	s, err := driver.String.ConvertValue(value)
	if err != nil {
		return fmt.Errorf("failed to convert capability")
	}

	v, ok := s.(string)
	if !ok {
		return fmt.Errorf("failed to cast capability")
	}

	switch v {
	case "FETCH_ACCOUNTS":
		*t = CAPABILITY_FETCH_ACCOUNTS
	case "FETCH_BALANCES":
		*t = CAPABILITY_FETCH_BALANCES
	case "FETCH_EXTERNAL_ACCOUNTS":
		*t = CAPABILITY_FETCH_EXTERNAL_ACCOUNTS
	case "FETCH_PAYMENTS":
		*t = CAPABILITY_FETCH_PAYMENTS
	case "FETCH_OTHERS":
		*t = CAPABILITY_FETCH_OTHERS

	case "CREATE_WEBHOOKS":
		*t = CAPABILITY_CREATE_WEBHOOKS
	case "TRANSLATE_WEBHOOKS":
		*t = CAPABILITY_TRANSLATE_WEBHOOKS

	case "CREATE_BANK_ACCOUNT":
		*t = CAPABILITY_CREATE_BANK_ACCOUNT
	case "CREATE_TRANSFER":
		*t = CAPABILITY_CREATE_TRANSFER
	case "CREATE_PAYOUT":
		*t = CAPABILITY_CREATE_PAYOUT

	case "ALLOW_FORMANCE_ACCOUNT_CREATION":
		*t = CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION
	case "ALLOW_FORMANCE_PAYMENT_CREATION":
		*t = CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION

	default:
		return fmt.Errorf("unknown capability")
	}

	return nil
}
