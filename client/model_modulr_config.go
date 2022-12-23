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

// checks if the ModulrConfig type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &ModulrConfig{}

// ModulrConfig struct for ModulrConfig
type ModulrConfig struct {
	ApiKey interface{} `json:"apiKey"`
	ApiSecret interface{} `json:"apiSecret"`
	Endpoint interface{} `json:"endpoint,omitempty"`
}

// NewModulrConfig instantiates a new ModulrConfig object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewModulrConfig(apiKey interface{}, apiSecret interface{}) *ModulrConfig {
	this := ModulrConfig{}
	this.ApiKey = apiKey
	this.ApiSecret = apiSecret
	return &this
}

// NewModulrConfigWithDefaults instantiates a new ModulrConfig object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewModulrConfigWithDefaults() *ModulrConfig {
	this := ModulrConfig{}
	return &this
}

// GetApiKey returns the ApiKey field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *ModulrConfig) GetApiKey() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.ApiKey
}

// GetApiKeyOk returns a tuple with the ApiKey field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ModulrConfig) GetApiKeyOk() (*interface{}, bool) {
	if o == nil || isNil(o.ApiKey) {
		return nil, false
	}
	return &o.ApiKey, true
}

// SetApiKey sets field value
func (o *ModulrConfig) SetApiKey(v interface{}) {
	o.ApiKey = v
}

// GetApiSecret returns the ApiSecret field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *ModulrConfig) GetApiSecret() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.ApiSecret
}

// GetApiSecretOk returns a tuple with the ApiSecret field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ModulrConfig) GetApiSecretOk() (*interface{}, bool) {
	if o == nil || isNil(o.ApiSecret) {
		return nil, false
	}
	return &o.ApiSecret, true
}

// SetApiSecret sets field value
func (o *ModulrConfig) SetApiSecret(v interface{}) {
	o.ApiSecret = v
}

// GetEndpoint returns the Endpoint field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *ModulrConfig) GetEndpoint() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.Endpoint
}

// GetEndpointOk returns a tuple with the Endpoint field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *ModulrConfig) GetEndpointOk() (*interface{}, bool) {
	if o == nil || isNil(o.Endpoint) {
		return nil, false
	}
	return &o.Endpoint, true
}

// HasEndpoint returns a boolean if a field has been set.
func (o *ModulrConfig) HasEndpoint() bool {
	if o != nil && isNil(o.Endpoint) {
		return true
	}

	return false
}

// SetEndpoint gets a reference to the given interface{} and assigns it to the Endpoint field.
func (o *ModulrConfig) SetEndpoint(v interface{}) {
	o.Endpoint = v
}

func (o ModulrConfig) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o ModulrConfig) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if o.ApiKey != nil {
		toSerialize["apiKey"] = o.ApiKey
	}
	if o.ApiSecret != nil {
		toSerialize["apiSecret"] = o.ApiSecret
	}
	if o.Endpoint != nil {
		toSerialize["endpoint"] = o.Endpoint
	}
	return toSerialize, nil
}

type NullableModulrConfig struct {
	value *ModulrConfig
	isSet bool
}

func (v NullableModulrConfig) Get() *ModulrConfig {
	return v.value
}

func (v *NullableModulrConfig) Set(val *ModulrConfig) {
	v.value = val
	v.isSet = true
}

func (v NullableModulrConfig) IsSet() bool {
	return v.isSet
}

func (v *NullableModulrConfig) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableModulrConfig(val *ModulrConfig) *NullableModulrConfig {
	return &NullableModulrConfig{value: val, isSet: true}
}

func (v NullableModulrConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableModulrConfig) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


