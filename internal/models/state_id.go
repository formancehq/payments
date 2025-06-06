package models

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/gibson042/canonicaljson-go"
)

type StateID struct {
	Reference   string
	ConnectorID ConnectorID
}

func (aid *StateID) String() string {
	if aid == nil || aid.Reference == "" {
		return ""
	}

	data, err := canonicaljson.Marshal(aid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func StateIDFromString(value string) (*StateID, error) {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		return nil, err
	}
	ret := StateID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func MustStateIDFromString(value string) StateID {
	id, err := StateIDFromString(value)
	if err != nil {
		panic(err)
	}
	return *id
}

func (aid StateID) Value() (driver.Value, error) {
	return aid.String(), nil
}

func (aid *StateID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("account id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {

		if v, ok := s.(string); ok {

			id, err := StateIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse account id %s: %v", v, err)
			}

			*aid = *id
			return nil
		}
	}

	return fmt.Errorf("failed to scan account id: %v", value)
}
