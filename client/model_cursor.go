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

// Cursor struct for Cursor
type Cursor struct {
	// Indicates if there are more items to fetch
	HasMore interface{} `json:"hasMore,omitempty"`
	// The cursor to use to fetch the next page of results
	Next interface{} `json:"next,omitempty"`
	// The cursor to use to fetch the previous page of results
	Previous interface{} `json:"previous,omitempty"`
	// The number of items per page
	PageSize interface{} `json:"pageSize,omitempty"`
}

// NewCursor instantiates a new Cursor object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewCursor() *Cursor {
	this := Cursor{}
	return &this
}

// NewCursorWithDefaults instantiates a new Cursor object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewCursorWithDefaults() *Cursor {
	this := Cursor{}
	return &this
}

// GetHasMore returns the HasMore field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Cursor) GetHasMore() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.HasMore
}

// GetHasMoreOk returns a tuple with the HasMore field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *Cursor) GetHasMoreOk() (*interface{}, bool) {
	if o == nil || isNil(o.HasMore) {
		return nil, false
	}
	return &o.HasMore, true
}

// HasHasMore returns a boolean if a field has been set.
func (o *Cursor) HasHasMore() bool {
	if o != nil && isNil(o.HasMore) {
		return true
	}

	return false
}

// SetHasMore gets a reference to the given interface{} and assigns it to the HasMore field.
func (o *Cursor) SetHasMore(v interface{}) {
	o.HasMore = v
}

// GetNext returns the Next field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Cursor) GetNext() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.Next
}

// GetNextOk returns a tuple with the Next field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *Cursor) GetNextOk() (*interface{}, bool) {
	if o == nil || isNil(o.Next) {
		return nil, false
	}
	return &o.Next, true
}

// HasNext returns a boolean if a field has been set.
func (o *Cursor) HasNext() bool {
	if o != nil && isNil(o.Next) {
		return true
	}

	return false
}

// SetNext gets a reference to the given interface{} and assigns it to the Next field.
func (o *Cursor) SetNext(v interface{}) {
	o.Next = v
}

// GetPrevious returns the Previous field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Cursor) GetPrevious() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.Previous
}

// GetPreviousOk returns a tuple with the Previous field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *Cursor) GetPreviousOk() (*interface{}, bool) {
	if o == nil || isNil(o.Previous) {
		return nil, false
	}
	return &o.Previous, true
}

// HasPrevious returns a boolean if a field has been set.
func (o *Cursor) HasPrevious() bool {
	if o != nil && isNil(o.Previous) {
		return true
	}

	return false
}

// SetPrevious gets a reference to the given interface{} and assigns it to the Previous field.
func (o *Cursor) SetPrevious(v interface{}) {
	o.Previous = v
}

// GetPageSize returns the PageSize field value if set, zero value otherwise (both if not set or set to explicit null).
func (o *Cursor) GetPageSize() interface{} {
	if o == nil {
		var ret interface{}
		return ret
	}
	return o.PageSize
}

// GetPageSizeOk returns a tuple with the PageSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *Cursor) GetPageSizeOk() (*interface{}, bool) {
	if o == nil || isNil(o.PageSize) {
		return nil, false
	}
	return &o.PageSize, true
}

// HasPageSize returns a boolean if a field has been set.
func (o *Cursor) HasPageSize() bool {
	if o != nil && isNil(o.PageSize) {
		return true
	}

	return false
}

// SetPageSize gets a reference to the given interface{} and assigns it to the PageSize field.
func (o *Cursor) SetPageSize(v interface{}) {
	o.PageSize = v
}

func (o Cursor) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.HasMore != nil {
		toSerialize["hasMore"] = o.HasMore
	}
	if o.Next != nil {
		toSerialize["next"] = o.Next
	}
	if o.Previous != nil {
		toSerialize["previous"] = o.Previous
	}
	if o.PageSize != nil {
		toSerialize["pageSize"] = o.PageSize
	}
	return json.Marshal(toSerialize)
}

type NullableCursor struct {
	value *Cursor
	isSet bool
}

func (v NullableCursor) Get() *Cursor {
	return v.value
}

func (v *NullableCursor) Set(val *Cursor) {
	v.value = val
	v.isSet = true
}

func (v NullableCursor) IsSet() bool {
	return v.isSet
}

func (v *NullableCursor) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableCursor(val *Cursor) *NullableCursor {
	return &NullableCursor{value: val, isSet: true}
}

func (v NullableCursor) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableCursor) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
