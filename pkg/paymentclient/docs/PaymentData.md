# PaymentData

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Provider** | **string** |  | 
**Reference** | Pointer to **string** |  | [optional] 
**Scheme** | Pointer to **string** |  | [optional] 
**Status** | **string** |  | 
**Value** | [**PaymentDataValue**](PaymentDataValue.md) |  | 
**Date** | **string** |  | 
**Raw** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewPaymentData

`func NewPaymentData(provider string, status string, value PaymentDataValue, date string, ) *PaymentData`

NewPaymentData instantiates a new PaymentData object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPaymentDataWithDefaults

`func NewPaymentDataWithDefaults() *PaymentData`

NewPaymentDataWithDefaults instantiates a new PaymentData object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetProvider

`func (o *PaymentData) GetProvider() string`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *PaymentData) GetProviderOk() (*string, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *PaymentData) SetProvider(v string)`

SetProvider sets Provider field to given value.


### GetReference

`func (o *PaymentData) GetReference() string`

GetReference returns the Reference field if non-nil, zero value otherwise.

### GetReferenceOk

`func (o *PaymentData) GetReferenceOk() (*string, bool)`

GetReferenceOk returns a tuple with the Reference field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReference

`func (o *PaymentData) SetReference(v string)`

SetReference sets Reference field to given value.

### HasReference

`func (o *PaymentData) HasReference() bool`

HasReference returns a boolean if a field has been set.

### GetScheme

`func (o *PaymentData) GetScheme() string`

GetScheme returns the Scheme field if non-nil, zero value otherwise.

### GetSchemeOk

`func (o *PaymentData) GetSchemeOk() (*string, bool)`

GetSchemeOk returns a tuple with the Scheme field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScheme

`func (o *PaymentData) SetScheme(v string)`

SetScheme sets Scheme field to given value.

### HasScheme

`func (o *PaymentData) HasScheme() bool`

HasScheme returns a boolean if a field has been set.

### GetStatus

`func (o *PaymentData) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *PaymentData) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *PaymentData) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetValue

`func (o *PaymentData) GetValue() PaymentDataValue`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *PaymentData) GetValueOk() (*PaymentDataValue, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *PaymentData) SetValue(v PaymentDataValue)`

SetValue sets Value field to given value.


### GetDate

`func (o *PaymentData) GetDate() string`

GetDate returns the Date field if non-nil, zero value otherwise.

### GetDateOk

`func (o *PaymentData) GetDateOk() (*string, bool)`

GetDateOk returns a tuple with the Date field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDate

`func (o *PaymentData) SetDate(v string)`

SetDate sets Date field to given value.


### GetRaw

`func (o *PaymentData) GetRaw() map[string]interface{}`

GetRaw returns the Raw field if non-nil, zero value otherwise.

### GetRawOk

`func (o *PaymentData) GetRawOk() (*map[string]interface{}, bool)`

GetRawOk returns a tuple with the Raw field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRaw

`func (o *PaymentData) SetRaw(v map[string]interface{})`

SetRaw sets Raw field to given value.

### HasRaw

`func (o *PaymentData) HasRaw() bool`

HasRaw returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


