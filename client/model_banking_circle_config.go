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

// BankingCircleConfig struct for BankingCircleConfig
type BankingCircleConfig struct {
	Username              interface{} `json:"username"`
	Password              interface{} `json:"password"`
	Endpoint              interface{} `json:"endpoint"`
	AuthorizationEndpoint interface{} `json:"authorizationEndpoint"`
}

// NewBankingCircleConfig instantiates a new BankingCircleConfig object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewBankingCircleConfig(username interface{}, password interface{}, endpoint interface{}, authorizationEndpoint interface{}) *BankingCircleConfig {
	this := BankingCircleConfig{}
	this.Username = username
	this.Password = password
	this.Endpoint = endpoint
	this.AuthorizationEndpoint = authorizationEndpoint
	return &this
}

// NewBankingCircleConfigWithDefaults instantiates a new BankingCircleConfig object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewBankingCircleConfigWithDefaults() *BankingCircleConfig {
	this := BankingCircleConfig{}
	return &this
}

// GetUsername returns the Username field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *BankingCircleConfig) GetUsername() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.Username
}

// GetUsernameOk returns a tuple with the Username field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *BankingCircleConfig) GetUsernameOk() (*interface{}, bool) {
	if o == nil || isNil(o.Username) {
		return nil, false
	}
	return &o.Username, true
}

// SetUsername sets field value
func (o *BankingCircleConfig) SetUsername(v interface{}) {
	o.Username = v
}

// GetPassword returns the Password field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *BankingCircleConfig) GetPassword() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.Password
}

// GetPasswordOk returns a tuple with the Password field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *BankingCircleConfig) GetPasswordOk() (*interface{}, bool) {
	if o == nil || isNil(o.Password) {
		return nil, false
	}
	return &o.Password, true
}

// SetPassword sets field value
func (o *BankingCircleConfig) SetPassword(v interface{}) {
	o.Password = v
}

// GetEndpoint returns the Endpoint field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *BankingCircleConfig) GetEndpoint() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.Endpoint
}

// GetEndpointOk returns a tuple with the Endpoint field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *BankingCircleConfig) GetEndpointOk() (*interface{}, bool) {
	if o == nil || isNil(o.Endpoint) {
		return nil, false
	}
	return &o.Endpoint, true
}

// SetEndpoint sets field value
func (o *BankingCircleConfig) SetEndpoint(v interface{}) {
	o.Endpoint = v
}

// GetAuthorizationEndpoint returns the AuthorizationEndpoint field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *BankingCircleConfig) GetAuthorizationEndpoint() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.AuthorizationEndpoint
}

// GetAuthorizationEndpointOk returns a tuple with the AuthorizationEndpoint field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *BankingCircleConfig) GetAuthorizationEndpointOk() (*interface{}, bool) {
	if o == nil || isNil(o.AuthorizationEndpoint) {
		return nil, false
	}
	return &o.AuthorizationEndpoint, true
}

// SetAuthorizationEndpoint sets field value
func (o *BankingCircleConfig) SetAuthorizationEndpoint(v interface{}) {
	o.AuthorizationEndpoint = v
}

func (o BankingCircleConfig) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Username != nil {
		toSerialize["username"] = o.Username
	}
	if o.Password != nil {
		toSerialize["password"] = o.Password
	}
	if o.Endpoint != nil {
		toSerialize["endpoint"] = o.Endpoint
	}
	if o.AuthorizationEndpoint != nil {
		toSerialize["authorizationEndpoint"] = o.AuthorizationEndpoint
	}
	return json.Marshal(toSerialize)
}

type NullableBankingCircleConfig struct {
	value *BankingCircleConfig
	isSet bool
}

func (v NullableBankingCircleConfig) Get() *BankingCircleConfig {
	return v.value
}

func (v *NullableBankingCircleConfig) Set(val *BankingCircleConfig) {
	v.value = val
	v.isSet = true
}

func (v NullableBankingCircleConfig) IsSet() bool {
	return v.isSet
}

func (v *NullableBankingCircleConfig) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableBankingCircleConfig(val *BankingCircleConfig) *NullableBankingCircleConfig {
	return &NullableBankingCircleConfig{value: val, isSet: true}
}

func (v NullableBankingCircleConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableBankingCircleConfig) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}