package models

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/gibson042/canonicaljson-go"
)

type PaymentInitiationReversalAdjustmentID struct {
	PaymentInitiationReversalID PaymentInitiationReversalID
	CreatedAt                   time.Time
	Status                      PaymentInitiationReversalAdjustmentStatus
}

func (pid PaymentInitiationReversalAdjustmentID) String() string {
	data, err := canonicaljson.Marshal(pid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func PaymentInitiationReversalAdjustmentIDFromString(value string) (PaymentInitiationReversalAdjustmentID, error) {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		return PaymentInitiationReversalAdjustmentID{}, err
	}
	ret := PaymentInitiationReversalAdjustmentID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		return PaymentInitiationReversalAdjustmentID{}, err
	}

	return ret, nil
}

func MustPaymentInitiationReversalAdjustmentIDFromString(value string) *PaymentInitiationReversalAdjustmentID {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		panic(err)
	}
	ret := PaymentInitiationReversalAdjustmentID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		panic(err)
	}

	return &ret
}

func (pid PaymentInitiationReversalAdjustmentID) Value() (driver.Value, error) {
	return pid.String(), nil
}

func (pid *PaymentInitiationReversalAdjustmentID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("payment reversal adjustment id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {

		if v, ok := s.(string); ok {

			id, err := PaymentInitiationReversalAdjustmentIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse payment reversal adjustment id %s: %v", v, err)
			}

			*pid = id
			return nil
		}
	}

	return fmt.Errorf("failed to scan payment reversal adjustment id: %v", value)
}
