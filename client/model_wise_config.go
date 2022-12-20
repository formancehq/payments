/*
Payments API

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 1.0.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package client

import (
	"encoding/json"
)

// WiseConfig struct for WiseConfig
type WiseConfig struct {
	ApiKey interface{} `json:"apiKey"`
}

// NewWiseConfig instantiates a new WiseConfig object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewWiseConfig(apiKey interface{}) *WiseConfig {
	this := WiseConfig{}
	this.ApiKey = apiKey
	return &this
}

// NewWiseConfigWithDefaults instantiates a new WiseConfig object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewWiseConfigWithDefaults() *WiseConfig {
	this := WiseConfig{}
	return &this
}

// GetApiKey returns the ApiKey field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *WiseConfig) GetApiKey() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *WiseConfig) GetApiKeyOk() (*interface{}, bool) {
	if o == nil || isNil(o.ApiKey) {
		return nil, false
	}
	return &o.ApiKey, true
}

// SetApiKey sets field value
func (o *WiseConfig) SetApiKey(v interface{}) {
	o.ApiKey = v
}

func (o WiseConfig) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.ApiKey != nil {
		toSerialize["apiKey"] = o.ApiKey
	}
	return json.Marshal(toSerialize)
}

type NullableWiseConfig struct {
	value *WiseConfig
	isSet bool
}

func (v NullableWiseConfig) Get() *WiseConfig {
	return v.value
}

func (v *NullableWiseConfig) Set(val *WiseConfig) {
	v.value = val
	v.isSet = true
}

func (v NullableWiseConfig) IsSet() bool {
	return v.isSet
}

func (v *NullableWiseConfig) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableWiseConfig(val *WiseConfig) *NullableWiseConfig {
	return &NullableWiseConfig{value: val, isSet: true}
}

func (v NullableWiseConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableWiseConfig) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}