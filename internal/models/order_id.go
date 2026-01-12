package models

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/gibson042/canonicaljson-go"
)

type OrderID struct {
	Reference   string
	ConnectorID ConnectorID
}

func (oid OrderID) String() string {
	data, err := canonicaljson.Marshal(oid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func OrderIDFromString(value string) (OrderID, error) {
	ret := OrderID{}
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

func MustOrderIDFromString(value string) *OrderID {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		panic(err)
	}
	ret := OrderID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		panic(err)
	}

	return &ret
}

func (oid OrderID) Value() (driver.Value, error) {
	return oid.String(), nil
}

func (oid *OrderID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("order id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {

		if v, ok := s.(string); ok {

			id, err := OrderIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse order id %s: %v", v, err)
			}

			*oid = id
			return nil
		}
	}

	return fmt.Errorf("failed to scan order id: %v", value)
}
