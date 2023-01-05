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

// TaskDescriptorStripeDescriptor struct for TaskDescriptorStripeDescriptor
type TaskDescriptorStripeDescriptor struct {
	Name    interface{} `json:"name,omitempty"`
	Main    interface{} `json:"main,omitempty"`
	Account interface{} `json:"account,omitempty"`
}

// NewTaskDescriptorStripeDescriptor instantiates a new TaskDescriptorStripeDescriptor object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewTaskDescriptorStripeDescriptor() *TaskDescriptorStripeDescriptor {
	this := TaskDescriptorStripeDescriptor{}
	return &this
}

// NewTaskDescriptorStripeDescriptorWithDefaults instantiates a new TaskDescriptorStripeDescriptor object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewTaskDescriptorStripeDescriptorWithDefaults() *TaskDescriptorStripeDescriptor {
	this := TaskDescriptorStripeDescriptor{}
	return &this
}

// GetName returns the Name field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *TaskDescriptorStripeDescriptor) GetName() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *TaskDescriptorStripeDescriptor) GetNameOk() (*interface{}, bool) {
	if o == nil || isNil(o.Name) {
		return nil, false
	}
	return &o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *TaskDescriptorStripeDescriptor) HasName() bool {
	if o != nil && isNil(o.Name) {
		return true
	}

	return false
}

// SetName gets a reference to the given interface{} and assigns it to the Name field.
func (o *TaskDescriptorStripeDescriptor) SetName(v interface{}) {
	o.Name = v
}

// GetMain returns the Main field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *TaskDescriptorStripeDescriptor) GetMain() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.Main
}

// GetMainOk returns a tuple with the Main field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *TaskDescriptorStripeDescriptor) GetMainOk() (*interface{}, bool) {
	if o == nil || isNil(o.Main) {
		return nil, false
	}
	return &o.Main, true
}

// HasMain returns a boolean if a field has been set.
func (o *TaskDescriptorStripeDescriptor) HasMain() bool {
	if o != nil && isNil(o.Main) {
		return true
	}

	return false
}

// SetMain gets a reference to the given interface{} and assigns it to the Main field.
func (o *TaskDescriptorStripeDescriptor) SetMain(v interface{}) {
	o.Main = v
}

// GetAccount returns the Account field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *TaskDescriptorStripeDescriptor) GetAccount() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.Account
}

// GetAccountOk returns a tuple with the Account field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *TaskDescriptorStripeDescriptor) GetAccountOk() (*interface{}, bool) {
	if o == nil || isNil(o.Account) {
		return nil, false
	}
	return &o.Account, true
}

// HasAccount returns a boolean if a field has been set.
func (o *TaskDescriptorStripeDescriptor) HasAccount() bool {
	if o != nil && isNil(o.Account) {
		return true
	}

	return false
}

// SetAccount gets a reference to the given interface{} and assigns it to the Account field.
func (o *TaskDescriptorStripeDescriptor) SetAccount(v interface{}) {
	o.Account = v
}

func (o TaskDescriptorStripeDescriptor) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Name != nil {
		toSerialize["name"] = o.Name
	}
	if o.Main != nil {
		toSerialize["main"] = o.Main
	}
	if o.Account != nil {
		toSerialize["account"] = o.Account
	}
	return json.Marshal(toSerialize)
}

type NullableTaskDescriptorStripeDescriptor struct {
	value *TaskDescriptorStripeDescriptor
	isSet bool
}

func (v NullableTaskDescriptorStripeDescriptor) Get() *TaskDescriptorStripeDescriptor {
	return v.value
}

func (v *NullableTaskDescriptorStripeDescriptor) Set(val *TaskDescriptorStripeDescriptor) {
	v.value = val
	v.isSet = true
}

func (v NullableTaskDescriptorStripeDescriptor) IsSet() bool {
	return v.isSet
}

func (v *NullableTaskDescriptorStripeDescriptor) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableTaskDescriptorStripeDescriptor(val *TaskDescriptorStripeDescriptor) *NullableTaskDescriptorStripeDescriptor {
	return &NullableTaskDescriptorStripeDescriptor{value: val, isSet: true}
}

func (v NullableTaskDescriptorStripeDescriptor) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableTaskDescriptorStripeDescriptor) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
