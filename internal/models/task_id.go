package models

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/gibson042/canonicaljson-go"
)

type TaskID struct {
	Reference   string
	ConnectorID ConnectorID
}

func TaskIDReference(prefix string, connectorID ConnectorID, objectID string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, connectorID.String(), objectID)
}

func (aid *TaskID) String() string {
	if aid == nil || aid.Reference == "" {
		return ""
	}

	data, err := canonicaljson.Marshal(aid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func TaskIDFromString(value string) (*TaskID, error) {
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		return nil, err
	}
	ret := TaskID{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func MustTaskIDFromString(value string) TaskID {
	id, err := TaskIDFromString(value)
	if err != nil {
		panic(err)
	}
	return *id
}

func (aid TaskID) Value() (driver.Value, error) {
	return aid.String(), nil
}

func (aid *TaskID) Scan(value interface{}) error {
	if value == nil {
		return errors.New("task id is nil")
	}

	if s, err := driver.String.ConvertValue(value); err == nil {

		if v, ok := s.(string); ok {

			id, err := TaskIDFromString(v)
			if err != nil {
				return fmt.Errorf("failed to parse task id %s: %v", v, err)
			}

			*aid = *id
			return nil
		}
	}

	return fmt.Errorf("failed to scan task id: %v", value)
}
