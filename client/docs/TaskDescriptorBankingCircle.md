# TaskDescriptorBankingCircle

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Provider** | Pointer to **interface{}** | The connector code | [optional] 
**CreatedAt** | Pointer to **interface{}** | The date when the task was created | [optional] 
**Status** | Pointer to **interface{}** | The task status | [optional] 
**Error** | Pointer to **interface{}** | The error message if the task failed | [optional] 
**State** | Pointer to **interface{}** | The task state | [optional] 
**Descriptor** | Pointer to [**TaskDescriptorBankingCircleDescriptor**](TaskDescriptorBankingCircleDescriptor.md) |  | [optional] 

## Methods

### NewTaskDescriptorBankingCircle

`func NewTaskDescriptorBankingCircle() *TaskDescriptorBankingCircle`

NewTaskDescriptorBankingCircle instantiates a new TaskDescriptorBankingCircle object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTaskDescriptorBankingCircleWithDefaults

`func NewTaskDescriptorBankingCircleWithDefaults() *TaskDescriptorBankingCircle`

NewTaskDescriptorBankingCircleWithDefaults instantiates a new TaskDescriptorBankingCircle object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetProvider

`func (o *TaskDescriptorBankingCircle) GetProvider() interface{}`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *TaskDescriptorBankingCircle) GetProviderOk() (*interface{}, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *TaskDescriptorBankingCircle) SetProvider(v interface{})`

SetProvider sets Provider field to given value.

### HasProvider

`func (o *TaskDescriptorBankingCircle) HasProvider() bool`

HasProvider returns a boolean if a field has been set.

### SetProviderNil

`func (o *TaskDescriptorBankingCircle) SetProviderNil(b bool)`

 SetProviderNil sets the value for Provider to be an explicit nil

### UnsetProvider
`func (o *TaskDescriptorBankingCircle) UnsetProvider()`

UnsetProvider ensures that no value is present for Provider, not even an explicit nil
### GetCreatedAt

`func (o *TaskDescriptorBankingCircle) GetCreatedAt() interface{}`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *TaskDescriptorBankingCircle) GetCreatedAtOk() (*interface{}, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *TaskDescriptorBankingCircle) SetCreatedAt(v interface{})`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *TaskDescriptorBankingCircle) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### SetCreatedAtNil

`func (o *TaskDescriptorBankingCircle) SetCreatedAtNil(b bool)`

 SetCreatedAtNil sets the value for CreatedAt to be an explicit nil

### UnsetCreatedAt
`func (o *TaskDescriptorBankingCircle) UnsetCreatedAt()`

UnsetCreatedAt ensures that no value is present for CreatedAt, not even an explicit nil
### GetStatus

`func (o *TaskDescriptorBankingCircle) GetStatus() interface{}`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *TaskDescriptorBankingCircle) GetStatusOk() (*interface{}, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *TaskDescriptorBankingCircle) SetStatus(v interface{})`

SetStatus sets Status field to given value.

### HasStatus

`func (o *TaskDescriptorBankingCircle) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### SetStatusNil

`func (o *TaskDescriptorBankingCircle) SetStatusNil(b bool)`

 SetStatusNil sets the value for Status to be an explicit nil

### UnsetStatus
`func (o *TaskDescriptorBankingCircle) UnsetStatus()`

UnsetStatus ensures that no value is present for Status, not even an explicit nil
### GetError

`func (o *TaskDescriptorBankingCircle) GetError() interface{}`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *TaskDescriptorBankingCircle) GetErrorOk() (*interface{}, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *TaskDescriptorBankingCircle) SetError(v interface{})`

SetError sets Error field to given value.

### HasError

`func (o *TaskDescriptorBankingCircle) HasError() bool`

HasError returns a boolean if a field has been set.

### SetErrorNil

`func (o *TaskDescriptorBankingCircle) SetErrorNil(b bool)`

 SetErrorNil sets the value for Error to be an explicit nil

### UnsetError
`func (o *TaskDescriptorBankingCircle) UnsetError()`

UnsetError ensures that no value is present for Error, not even an explicit nil
### GetState

`func (o *TaskDescriptorBankingCircle) GetState() interface{}`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *TaskDescriptorBankingCircle) GetStateOk() (*interface{}, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *TaskDescriptorBankingCircle) SetState(v interface{})`

SetState sets State field to given value.

### HasState

`func (o *TaskDescriptorBankingCircle) HasState() bool`

HasState returns a boolean if a field has been set.

### SetStateNil

`func (o *TaskDescriptorBankingCircle) SetStateNil(b bool)`

 SetStateNil sets the value for State to be an explicit nil

### UnsetState
`func (o *TaskDescriptorBankingCircle) UnsetState()`

UnsetState ensures that no value is present for State, not even an explicit nil
### GetDescriptor

`func (o *TaskDescriptorBankingCircle) GetDescriptor() TaskDescriptorBankingCircleDescriptor`

GetDescriptor returns the Descriptor field if non-nil, zero value otherwise.

### GetDescriptorOk

`func (o *TaskDescriptorBankingCircle) GetDescriptorOk() (*TaskDescriptorBankingCircleDescriptor, bool)`

GetDescriptorOk returns a tuple with the Descriptor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescriptor

`func (o *TaskDescriptorBankingCircle) SetDescriptor(v TaskDescriptorBankingCircleDescriptor)`

SetDescriptor sets Descriptor field to given value.

### HasDescriptor

`func (o *TaskDescriptorBankingCircle) HasDescriptor() bool`

HasDescriptor returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


