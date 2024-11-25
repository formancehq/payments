package models

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/gibson042/canonicaljson-go"
)

type PaymentInitiationReversalID struct {
	Reference   string
	ConnectorID ConnectorID
}

func (pid PaymentInitiationReversalID) String() string {
	data, err := canonicaljson.Marshal(pid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func PaymentInitiationReversalIDFromString(value string) (PaymentInitiationReversalID, error) {
	ret := PaymentInitiationReversalID{}
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		return ret, err
	}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

func MustPaymentInitiationReversalIDFromString(value string) *PaymentInitiationReversalID {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		panic(err)
	}
	ret := PaymentInitiationReversalID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		panic(err)
	}

	return &ret
}

func (pid PaymentInitiationReversalID) Value() (driver.Value, error) {
	return pid.String(), nil
}

func (pid *PaymentInitiationReversalID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("payment initiation reversal id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {

		if v, ok := s.(string); ok {

			id, err := PaymentInitiationReversalIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse payment initiation reversal id %s: %v", v, err)
			}

			*pid = id
			return nil
		}
	}

	return fmt.Errorf("failed to scan payment initiation reversal id: %v", value)
}
