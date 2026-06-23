package models

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/gibson042/canonicaljson-go"
)

type EventID struct {
	EventIdempotencyKey string
	ConnectorID         *ConnectorID
}

func (aid *EventID) String() string {
	if aid == nil || aid.EventIdempotencyKey == "" {
		return ""
	}

	data, err := canonicaljson.Marshal(aid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func EventIDFromString(value string) (EventID, error) {
	ret := EventID{}

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

func (aid EventID) Value() (driver.Value, error) {
	return aid.String(), nil
}

func (aid *EventID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("event id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {

		if v, ok := s.(string); ok {

			id, err := EventIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse event id %s: %v", v, err)
			}

			*aid = id
			return nil
		}
	}

	return fmt.Errorf("failed to scan event id: %v", value)
}
