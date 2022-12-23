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

// checks if the DummyPayConfig type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &DummyPayConfig{}

// DummyPayConfig struct for DummyPayConfig
type DummyPayConfig struct {
	// The frequency at which the connector will try to fetch new payment objects from the directory
	FilePollingPeriod interface{} `json:"filePollingPeriod,omitempty"`
	// The frequency at which the connector will create new payment objects in the directory
	FileGenerationPeriod interface{} `json:"fileGenerationPeriod,omitempty"`
	Directory            interface{} `json:"directory"`
}

// NewDummyPayConfig instantiates a new DummyPayConfig object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewDummyPayConfig(directory interface{}) *DummyPayConfig {
	this := DummyPayConfig{}
	this.Directory = directory
	return &this
}

// NewDummyPayConfigWithDefaults instantiates a new DummyPayConfig object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewDummyPayConfigWithDefaults() *DummyPayConfig {
	this := DummyPayConfig{}
	return &this
}

// GetFilePollingPeriod returns the FilePollingPeriod field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DummyPayConfig) GetFilePollingPeriod() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.FilePollingPeriod
}

// GetFilePollingPeriodOk returns a tuple with the FilePollingPeriod field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *DummyPayConfig) GetFilePollingPeriodOk() (*interface{}, bool) {
	if o == nil || isNil(o.FilePollingPeriod) {
		return nil, false
	}
	return &o.FilePollingPeriod, true
}

// HasFilePollingPeriod returns a boolean if a field has been set.
func (o *DummyPayConfig) HasFilePollingPeriod() bool {
	if o != nil && isNil(o.FilePollingPeriod) {
		return true
	}

	return false
}

// SetFilePollingPeriod gets a reference to the given interface{} and assigns it to the FilePollingPeriod field.
func (o *DummyPayConfig) SetFilePollingPeriod(v interface{}) {
	o.FilePollingPeriod = v
}

// GetFileGenerationPeriod returns the FileGenerationPeriod field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *DummyPayConfig) GetFileGenerationPeriod() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.FileGenerationPeriod
}

// GetFileGenerationPeriodOk returns a tuple with the FileGenerationPeriod field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *DummyPayConfig) GetFileGenerationPeriodOk() (*interface{}, bool) {
	if o == nil || isNil(o.FileGenerationPeriod) {
		return nil, false
	}
	return &o.FileGenerationPeriod, true
}

// HasFileGenerationPeriod returns a boolean if a field has been set.
func (o *DummyPayConfig) HasFileGenerationPeriod() bool {
	if o != nil && isNil(o.FileGenerationPeriod) {
		return true
	}

	return false
}

// SetFileGenerationPeriod gets a reference to the given interface{} and assigns it to the FileGenerationPeriod field.
func (o *DummyPayConfig) SetFileGenerationPeriod(v interface{}) {
	o.FileGenerationPeriod = v
}

// GetDirectory returns the Directory field value
// If the value is explicit nil, the zero value for interface{} will be returned
func (o *DummyPayConfig) GetDirectory() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}

	return o.Directory
}

// GetDirectoryOk returns a tuple with the Directory field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *DummyPayConfig) GetDirectoryOk() (*interface{}, bool) {
	if o == nil || isNil(o.Directory) {
		return nil, false
	}
	return &o.Directory, true
}

// SetDirectory sets field value
func (o *DummyPayConfig) SetDirectory(v interface{}) {
	o.Directory = v
}

func (o DummyPayConfig) MarshalJSON() ([]byte, error) {
	toSerialize, err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o DummyPayConfig) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if o.FilePollingPeriod != nil {
		toSerialize["filePollingPeriod"] = o.FilePollingPeriod
	}
	if o.FileGenerationPeriod != nil {
		toSerialize["fileGenerationPeriod"] = o.FileGenerationPeriod
	}
	if o.Directory != nil {
		toSerialize["directory"] = o.Directory
	}
	return toSerialize, nil
}

type NullableDummyPayConfig struct {
	value *DummyPayConfig
	isSet bool
}

func (v NullableDummyPayConfig) Get() *DummyPayConfig {
	return v.value
}

func (v *NullableDummyPayConfig) Set(val *DummyPayConfig) {
	v.value = val
	v.isSet = true
}

func (v NullableDummyPayConfig) IsSet() bool {
	return v.isSet
}

func (v *NullableDummyPayConfig) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableDummyPayConfig(val *DummyPayConfig) *NullableDummyPayConfig {
	return &NullableDummyPayConfig{value: val, isSet: true}
}

func (v NullableDummyPayConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableDummyPayConfig) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
