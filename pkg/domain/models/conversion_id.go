package models

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/gibson042/canonicaljson-go"
)

type ConversionID struct {
	Reference   string
	ConnectorID ConnectorID
}

func (cid ConversionID) String() string {
	data, err := canonicaljson.Marshal(cid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func ConversionIDFromString(value string) (ConversionID, error) {
	ret := ConversionID{}
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

func MustConversionIDFromString(value string) *ConversionID {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		panic(err)
	}
	ret := ConversionID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		panic(err)
	}

	return &ret
}

func (cid ConversionID) Value() (driver.Value, error) {
	return cid.String(), nil
}

func (cid *ConversionID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("conversion id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {

		if v, ok := s.(string); ok {

			id, err := ConversionIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse conversion id %s: %v", v, err)
			}

			*cid = id
			return nil
		}
	}

	return fmt.Errorf("failed to scan conversion id: %v", value)
}
