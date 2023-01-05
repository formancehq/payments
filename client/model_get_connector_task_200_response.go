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

// GetConnectorTask200Response struct for GetConnectorTask200Response
type GetConnectorTask200Response struct {
	Data interface{} `json:"data,omitempty"`
}

// NewGetConnectorTask200Response instantiates a new GetConnectorTask200Response object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewGetConnectorTask200Response() *GetConnectorTask200Response {
	this := GetConnectorTask200Response{}
	return &this
}

// NewGetConnectorTask200ResponseWithDefaults instantiates a new GetConnectorTask200Response object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewGetConnectorTask200ResponseWithDefaults() *GetConnectorTask200Response {
	this := GetConnectorTask200Response{}
	return &this
}

// GetData returns the Data field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *GetConnectorTask200Response) GetData() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.Data
}

// GetDataOk returns a tuple with the Data field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *GetConnectorTask200Response) GetDataOk() (*interface{}, bool) {
	if o == nil || isNil(o.Data) {
		return nil, false
	}
	return &o.Data, true
}

// HasData returns a boolean if a field has been set.
func (o *GetConnectorTask200Response) HasData() bool {
	if o != nil && isNil(o.Data) {
		return true
	}

	return false
}

// SetData gets a reference to the given interface{} and assigns it to the Data field.
func (o *GetConnectorTask200Response) SetData(v interface{}) {
	o.Data = v
}

func (o GetConnectorTask200Response) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Data != nil {
		toSerialize["data"] = o.Data
	}
	return json.Marshal(toSerialize)
}

type NullableGetConnectorTask200Response struct {
	value *GetConnectorTask200Response
	isSet bool
}

func (v NullableGetConnectorTask200Response) Get() *GetConnectorTask200Response {
	return v.value
}

func (v *NullableGetConnectorTask200Response) Set(val *GetConnectorTask200Response) {
	v.value = val
	v.isSet = true
}

func (v NullableGetConnectorTask200Response) IsSet() bool {
	return v.isSet
}

func (v *NullableGetConnectorTask200Response) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableGetConnectorTask200Response(val *GetConnectorTask200Response) *NullableGetConnectorTask200Response {
	return &NullableGetConnectorTask200Response{value: val, isSet: true}
}

func (v NullableGetConnectorTask200Response) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableGetConnectorTask200Response) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
