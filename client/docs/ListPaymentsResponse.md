# ListPaymentsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Data** | **interface{}** |  | 
**Cursor** | Pointer to [**Cursor**](Cursor.md) |  | [optional] 

## Methods

### NewListPaymentsResponse

`func NewListPaymentsResponse(data interface{}, ) *ListPaymentsResponse`

NewListPaymentsResponse instantiates a new ListPaymentsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListPaymentsResponseWithDefaults

`func NewListPaymentsResponseWithDefaults() *ListPaymentsResponse`

NewListPaymentsResponseWithDefaults instantiates a new ListPaymentsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetData

`func (o *ListPaymentsResponse) GetData() interface{}`

GetData returns the Data field if non-nil, zero value otherwise.

### GetDataOk

`func (o *ListPaymentsResponse) GetDataOk() (*interface{}, bool)`

GetDataOk returns a tuple with the Data field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetData

`func (o *ListPaymentsResponse) SetData(v interface{})`

SetData sets Data field to given value.


### SetDataNil

`func (o *ListPaymentsResponse) SetDataNil(b bool)`

 SetDataNil sets the value for Data to be an explicit nil

### UnsetData
`func (o *ListPaymentsResponse) UnsetData()`

UnsetData ensures that no value is present for Data, not even an explicit nil
### GetCursor

`func (o *ListPaymentsResponse) GetCursor() Cursor`

GetCursor returns the Cursor field if non-nil, zero value otherwise.

### GetCursorOk

`func (o *ListPaymentsResponse) GetCursorOk() (*Cursor, bool)`

GetCursorOk returns a tuple with the Cursor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCursor

`func (o *ListPaymentsResponse) SetCursor(v Cursor)`

SetCursor sets Cursor field to given value.

### HasCursor

`func (o *ListPaymentsResponse) HasCursor() bool`

HasCursor returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


