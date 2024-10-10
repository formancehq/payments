package models

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/gibson042/canonicaljson-go"
)

type PaymentInitiationAdjustmentID struct {
	PaymentInitiationID PaymentInitiationID
	CreatedAt           time.Time
	Status              PaymentInitiationAdjustmentStatus
}

func (pid PaymentInitiationAdjustmentID) String() string {
	data, err := canonicaljson.Marshal(pid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func PaymentInitiationAdjustmentIDFromString(value string) (PaymentInitiationAdjustmentID, error) {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		return PaymentInitiationAdjustmentID{}, err
	}
	ret := PaymentInitiationAdjustmentID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		return PaymentInitiationAdjustmentID{}, err
	}

	return ret, nil
}

func MustPaymentInitiationAdjustmentIDFromString(value string) *PaymentInitiationAdjustmentID {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		panic(err)
	}
	ret := PaymentInitiationAdjustmentID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		panic(err)
	}

	return &ret
}

func (pid PaymentInitiationAdjustmentID) Value() (driver.Value, error) {
	return pid.String(), nil
}

func (pid *PaymentInitiationAdjustmentID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("payment adjustment id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {

		if v, ok := s.(string); ok {

			id, err := PaymentInitiationAdjustmentIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse payment adjustment id %s: %v", v, err)
			}

			*pid = id
			return nil
		}
	}

	return fmt.Errorf("failed to scan payment adjustement id: %v", value)
}
