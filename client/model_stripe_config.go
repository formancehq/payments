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

// StripeConfig struct for StripeConfig
type StripeConfig struct {
	// The frequency at which the connector will try to fetch new BalanceTransaction objects from Stripe api
	PollingPeriod interface{} `json:"pollingPeriod,omitempty"`
	ApiKey        interface{} `json:"apiKey"`
	// Number of BalanceTransaction to fetch at each polling interval.
	PageSize interface{} `json:"pageSize,omitempty"`
}

// NewStripeConfig instantiates a new StripeConfig object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewStripeConfig(apiKey interface{}) *StripeConfig {
	this := StripeConfig{}
	this.ApiKey = apiKey
	return &this
}

// NewStripeConfigWithDefaults instantiates a new StripeConfig object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewStripeConfigWithDefaults() *StripeConfig {
	this := StripeConfig{}
	return &this
}

// GetPollingPeriod returns the PollingPeriod field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *StripeConfig) GetPollingPeriod() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.PollingPeriod
}

// GetPollingPeriodOk returns a tuple with the PollingPeriod field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *StripeConfig) GetPollingPeriodOk() (*interface{}, bool) {
	if o == nil || isNil(o.PollingPeriod) {
		return nil, false
	}
	return &o.PollingPeriod, true
}

// HasPollingPeriod returns a boolean if a field has been set.
func (o *StripeConfig) HasPollingPeriod() bool {
	if o != nil && isNil(o.PollingPeriod) {
		return true
	}

	return false
}

// SetPollingPeriod gets a reference to the given interface{} and assigns it to the PollingPeriod field.
func (o *StripeConfig) SetPollingPeriod(v interface{}) {
	o.PollingPeriod = v
}

// GetApiKey returns the ApiKey field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *StripeConfig) GetApiKey() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *StripeConfig) GetApiKeyOk() (*interface{}, bool) {
	if o == nil || isNil(o.ApiKey) {
		return nil, false
	}
	return &o.ApiKey, true
}

// SetApiKey sets field value
func (o *StripeConfig) SetApiKey(v interface{}) {
	o.ApiKey = v
}

// GetPageSize returns the PageSize field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *StripeConfig) GetPageSize() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.PageSize
}

// GetPageSizeOk returns a tuple with the PageSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *StripeConfig) GetPageSizeOk() (*interface{}, bool) {
	if o == nil || isNil(o.PageSize) {
		return nil, false
	}
	return &o.PageSize, true
}

// HasPageSize returns a boolean if a field has been set.
func (o *StripeConfig) HasPageSize() bool {
	if o != nil && isNil(o.PageSize) {
		return true
	}

	return false
}

// SetPageSize gets a reference to the given interface{} and assigns it to the PageSize field.
func (o *StripeConfig) SetPageSize(v interface{}) {
	o.PageSize = v
}

func (o StripeConfig) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.PollingPeriod != nil {
		toSerialize["pollingPeriod"] = o.PollingPeriod
	}
	if o.ApiKey != nil {
		toSerialize["apiKey"] = o.ApiKey
	}
	if o.PageSize != nil {
		toSerialize["pageSize"] = o.PageSize
	}
	return json.Marshal(toSerialize)
}

type NullableStripeConfig struct {
	value *StripeConfig
	isSet bool
}

func (v NullableStripeConfig) Get() *StripeConfig {
	return v.value
}

func (v *NullableStripeConfig) Set(val *StripeConfig) {
	v.value = val
	v.isSet = true
}

func (v NullableStripeConfig) IsSet() bool {
	return v.isSet
}

func (v *NullableStripeConfig) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableStripeConfig(val *StripeConfig) *NullableStripeConfig {
	return &NullableStripeConfig{value: val, isSet: true}
}

func (v NullableStripeConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableStripeConfig) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
