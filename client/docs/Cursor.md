# Cursor

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**HasMore** | Pointer to **interface{}** | Indicates if there are more items to fetch | [optional] 
**Next** | Pointer to **interface{}** | The cursor to use to fetch the next page of results | [optional] 
**Previous** | Pointer to **interface{}** | The cursor to use to fetch the previous page of results | [optional] 
**PageSize** | Pointer to **interface{}** | The number of items per page | [optional] 

## Methods

### NewCursor

`func NewCursor() *Cursor`

NewCursor instantiates a new Cursor object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCursorWithDefaults

`func NewCursorWithDefaults() *Cursor`

NewCursorWithDefaults instantiates a new Cursor object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetHasMore

`func (o *Cursor) GetHasMore() interface{}`

GetHasMore returns the HasMore field if non-nil, zero value otherwise.

### GetHasMoreOk

`func (o *Cursor) GetHasMoreOk() (*interface{}, bool)`

GetHasMoreOk returns a tuple with the HasMore field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHasMore

`func (o *Cursor) SetHasMore(v interface{})`

SetHasMore sets HasMore field to given value.

### HasHasMore

`func (o *Cursor) HasHasMore() bool`

HasHasMore returns a boolean if a field has been set.

### SetHasMoreNil

`func (o *Cursor) SetHasMoreNil(b bool)`

 SetHasMoreNil sets the value for HasMore to be an explicit nil

### UnsetHasMore
`func (o *Cursor) UnsetHasMore()`

UnsetHasMore ensures that no value is present for HasMore, not even an explicit nil
### GetNext

`func (o *Cursor) GetNext() interface{}`

GetNext returns the Next field if non-nil, zero value otherwise.

### GetNextOk

`func (o *Cursor) GetNextOk() (*interface{}, bool)`

GetNextOk returns a tuple with the Next field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNext

`func (o *Cursor) SetNext(v interface{})`

SetNext sets Next field to given value.

### HasNext

`func (o *Cursor) HasNext() bool`

HasNext returns a boolean if a field has been set.

### SetNextNil

`func (o *Cursor) SetNextNil(b bool)`

 SetNextNil sets the value for Next to be an explicit nil

### UnsetNext
`func (o *Cursor) UnsetNext()`

UnsetNext ensures that no value is present for Next, not even an explicit nil
### GetPrevious

`func (o *Cursor) GetPrevious() interface{}`

GetPrevious returns the Previous field if non-nil, zero value otherwise.

### GetPreviousOk

`func (o *Cursor) GetPreviousOk() (*interface{}, bool)`

GetPreviousOk returns a tuple with the Previous field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPrevious

`func (o *Cursor) SetPrevious(v interface{})`

SetPrevious sets Previous field to given value.

### HasPrevious

`func (o *Cursor) HasPrevious() bool`

HasPrevious returns a boolean if a field has been set.

### SetPreviousNil

`func (o *Cursor) SetPreviousNil(b bool)`

 SetPreviousNil sets the value for Previous to be an explicit nil

### UnsetPrevious
`func (o *Cursor) UnsetPrevious()`

UnsetPrevious ensures that no value is present for Previous, not even an explicit nil
### GetPageSize

`func (o *Cursor) GetPageSize() interface{}`

GetPageSize returns the PageSize field if non-nil, zero value otherwise.

### GetPageSizeOk

`func (o *Cursor) GetPageSizeOk() (*interface{}, bool)`

GetPageSizeOk returns a tuple with the PageSize field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPageSize

`func (o *Cursor) SetPageSize(v interface{})`

SetPageSize sets PageSize field to given value.

### HasPageSize

`func (o *Cursor) HasPageSize() bool`

HasPageSize returns a boolean if a field has been set.

### SetPageSizeNil

`func (o *Cursor) SetPageSizeNil(b bool)`

 SetPageSizeNil sets the value for PageSize to be an explicit nil

### UnsetPageSize
`func (o *Cursor) UnsetPageSize()`

UnsetPageSize ensures that no value is present for PageSize, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


