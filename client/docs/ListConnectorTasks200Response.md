# ListConnectorTasks200Response

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Cursor** | Pointer to [**Cursor**](Cursor.md) |  | [optional] 
**Data** | Pointer to **interface{}** |  | [optional] 

## Methods

### NewListConnectorTasks200Response

`func NewListConnectorTasks200Response() *ListConnectorTasks200Response`

NewListConnectorTasks200Response instantiates a new ListConnectorTasks200Response object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewListConnectorTasks200ResponseWithDefaults

`func NewListConnectorTasks200ResponseWithDefaults() *ListConnectorTasks200Response`

NewListConnectorTasks200ResponseWithDefaults instantiates a new ListConnectorTasks200Response object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCursor

`func (o *ListConnectorTasks200Response) GetCursor() Cursor`

GetCursor returns the Cursor field if non-nil, zero value otherwise.

### GetCursorOk

`func (o *ListConnectorTasks200Response) GetCursorOk() (*Cursor, bool)`

GetCursorOk returns a tuple with the Cursor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCursor

`func (o *ListConnectorTasks200Response) SetCursor(v Cursor)`

SetCursor sets Cursor field to given value.

### HasCursor

`func (o *ListConnectorTasks200Response) HasCursor() bool`

HasCursor returns a boolean if a field has been set.

### GetData

`func (o *ListConnectorTasks200Response) GetData() interface{}`

GetData returns the Data field if non-nil, zero value otherwise.

### GetDataOk

`func (o *ListConnectorTasks200Response) GetDataOk() (*interface{}, bool)`

GetDataOk returns a tuple with the Data field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetData

`func (o *ListConnectorTasks200Response) SetData(v interface{})`

SetData sets Data field to given value.

### HasData

`func (o *ListConnectorTasks200Response) HasData() bool`

HasData returns a boolean if a field has been set.

### SetDataNil

`func (o *ListConnectorTasks200Response) SetDataNil(b bool)`

 SetDataNil sets the value for Data to be an explicit nil

### UnsetData
`func (o *ListConnectorTasks200Response) UnsetData()`

UnsetData ensures that no value is present for Data, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


