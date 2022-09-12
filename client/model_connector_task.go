/*
Payments API

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 1.0.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

import (
	"encoding/json"
	"time"
	"fmt"
)

// ConnectorTask - struct for ConnectorTask
type ConnectorTask struct {
	StripeTask *StripeTask
}

// StripeTaskAsConnectorTask is a convenience function that returns StripeTask wrapped in ConnectorTask
func StripeTaskAsConnectorTask(v *StripeTask) ConnectorTask {
	return ConnectorTask{
		StripeTask: v,
	}
}


// Unmarshal JSON data into one of the pointers in the struct
func (dst *ConnectorTask) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into StripeTask
	err = newStrictDecoder(data).Decode(&dst.StripeTask)
	if err == nil {
		jsonStripeTask, _ := json.Marshal(dst.StripeTask)
		if string(jsonStripeTask) == "{}" { // empty struct
			dst.StripeTask = nil
		} else {
			match++
		}
	} else {
		dst.StripeTask = nil
	}

	if match > 1 { // more than 1 match
		// reset to nil
		dst.StripeTask = nil

		return fmt.Errorf("Data matches more than one schema in oneOf(ConnectorTask)")
	} else if match == 1 {
		return nil // exactly one match
	} else { // no match
		return fmt.Errorf("Data failed to match schemas in oneOf(ConnectorTask)")
	}
}

// Marshal data from the first non-nil pointers in the struct to JSON
func (src ConnectorTask) MarshalJSON() ([]byte, error) {
	if src.StripeTask != nil {
		return json.Marshal(&src.StripeTask)
	}

	return nil, nil // no data in oneOf schemas
}

// Get the actual instance
func (obj *ConnectorTask) GetActualInstance() (interface{}) {
	if obj == nil {
		return nil
	}
	if obj.StripeTask != nil {
		return obj.StripeTask
	}

	// all schemas are nil
	return nil
}

type NullableConnectorTask struct {
	value *ConnectorTask
	isSet bool
}

func (v NullableConnectorTask) Get() *ConnectorTask {
	return v.value
}

func (v *NullableConnectorTask) Set(val *ConnectorTask) {
	v.value = val
	v.isSet = true
}

func (v NullableConnectorTask) IsSet() bool {
	return v.isSet
}

func (v *NullableConnectorTask) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableConnectorTask(val *ConnectorTask) *NullableConnectorTask {
	return &NullableConnectorTask{value: val, isSet: true}
}

func (v NullableConnectorTask) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableConnectorTask) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


