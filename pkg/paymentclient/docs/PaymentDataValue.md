# PaymentDataValue

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Amount** | **int32** |  | 
**Asset** | **string** |  | 

## Methods

### NewPaymentDataValue

`func NewPaymentDataValue(amount int32, asset string, ) *PaymentDataValue`

NewPaymentDataValue instantiates a new PaymentDataValue object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPaymentDataValueWithDefaults

`func NewPaymentDataValueWithDefaults() *PaymentDataValue`

NewPaymentDataValueWithDefaults instantiates a new PaymentDataValue object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAmount

`func (o *PaymentDataValue) GetAmount() int32`

GetAmount returns the Amount field if non-nil, zero value otherwise.

### GetAmountOk

`func (o *PaymentDataValue) GetAmountOk() (*int32, bool)`

GetAmountOk returns a tuple with the Amount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAmount

`func (o *PaymentDataValue) SetAmount(v int32)`

SetAmount sets Amount field to given value.


### GetAsset

`func (o *PaymentDataValue) GetAsset() string`

GetAsset returns the Asset field if non-nil, zero value otherwise.

### GetAssetOk

`func (o *PaymentDataValue) GetAssetOk() (*string, bool)`

GetAssetOk returns a tuple with the Asset field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAsset

`func (o *PaymentDataValue) SetAsset(v string)`

SetAsset sets Asset field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


