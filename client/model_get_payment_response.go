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

// checks if the GetPaymentResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &GetPaymentResponse{}

// GetPaymentResponse struct for GetPaymentResponse
type GetPaymentResponse struct {
	Data Payment `json:"data"`
}

// NewGetPaymentResponse instantiates a new GetPaymentResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewGetPaymentResponse(data Payment) *GetPaymentResponse {
	this := GetPaymentResponse{}
	this.Data = data
	return &this
}

// NewGetPaymentResponseWithDefaults instantiates a new GetPaymentResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewGetPaymentResponseWithDefaults() *GetPaymentResponse {
	this := GetPaymentResponse{}
	return &this
}

// GetData returns the Data field value
func (o *GetPaymentResponse) GetData() Payment {
	if o == nil {
		var ret Payment
		return ret
	}

	return o.Data
}

// GetDataOk returns a tuple with the Data field value
// and a boolean to check if the value has been set.
func (o *GetPaymentResponse) GetDataOk() (*Payment, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Data, true
}

// SetData sets field value
func (o *GetPaymentResponse) SetData(v Payment) {
	o.Data = v
}

func (o GetPaymentResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o GetPaymentResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	toSerialize["data"] = o.Data
	return toSerialize, nil
}

type NullableGetPaymentResponse struct {
	value *GetPaymentResponse
	isSet bool
}

func (v NullableGetPaymentResponse) Get() *GetPaymentResponse {
	return v.value
}

func (v *NullableGetPaymentResponse) Set(val *GetPaymentResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableGetPaymentResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableGetPaymentResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableGetPaymentResponse(val *GetPaymentResponse) *NullableGetPaymentResponse {
	return &NullableGetPaymentResponse{value: val, isSet: true}
}

func (v NullableGetPaymentResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableGetPaymentResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


