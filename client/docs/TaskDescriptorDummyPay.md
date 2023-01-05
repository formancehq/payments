# TaskDescriptorDummyPay

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Provider** | Pointer to **interface{}** | The connector code | [optional] 
**CreatedAt** | Pointer to **interface{}** | The date when the task was created | [optional] 
**Status** | Pointer to **interface{}** | The task status | [optional] 
**Error** | Pointer to **interface{}** | The error message if the task failed | [optional] 
**State** | Pointer to **interface{}** | The task state | [optional] 
**Descriptor** | Pointer to [**TaskDescriptorDummyPayDescriptor**](TaskDescriptorDummyPayDescriptor.md) |  | [optional] 

## Methods

### NewTaskDescriptorDummyPay

`func NewTaskDescriptorDummyPay() *TaskDescriptorDummyPay`

NewTaskDescriptorDummyPay instantiates a new TaskDescriptorDummyPay object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTaskDescriptorDummyPayWithDefaults

`func NewTaskDescriptorDummyPayWithDefaults() *TaskDescriptorDummyPay`

NewTaskDescriptorDummyPayWithDefaults instantiates a new TaskDescriptorDummyPay object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetProvider

`func (o *TaskDescriptorDummyPay) GetProvider() interface{}`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *TaskDescriptorDummyPay) GetProviderOk() (*interface{}, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *TaskDescriptorDummyPay) SetProvider(v interface{})`

SetProvider sets Provider field to given value.

### HasProvider

`func (o *TaskDescriptorDummyPay) HasProvider() bool`

HasProvider returns a boolean if a field has been set.

### SetProviderNil

`func (o *TaskDescriptorDummyPay) SetProviderNil(b bool)`

 SetProviderNil sets the value for Provider to be an explicit nil

### UnsetProvider
`func (o *TaskDescriptorDummyPay) UnsetProvider()`

UnsetProvider ensures that no value is present for Provider, not even an explicit nil
### GetCreatedAt

`func (o *TaskDescriptorDummyPay) GetCreatedAt() interface{}`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *TaskDescriptorDummyPay) GetCreatedAtOk() (*interface{}, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *TaskDescriptorDummyPay) SetCreatedAt(v interface{})`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *TaskDescriptorDummyPay) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### SetCreatedAtNil

`func (o *TaskDescriptorDummyPay) SetCreatedAtNil(b bool)`

 SetCreatedAtNil sets the value for CreatedAt to be an explicit nil

### UnsetCreatedAt
`func (o *TaskDescriptorDummyPay) UnsetCreatedAt()`

UnsetCreatedAt ensures that no value is present for CreatedAt, not even an explicit nil
### GetStatus

`func (o *TaskDescriptorDummyPay) GetStatus() interface{}`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *TaskDescriptorDummyPay) GetStatusOk() (*interface{}, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *TaskDescriptorDummyPay) SetStatus(v interface{})`

SetStatus sets Status field to given value.

### HasStatus

`func (o *TaskDescriptorDummyPay) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### SetStatusNil

`func (o *TaskDescriptorDummyPay) SetStatusNil(b bool)`

 SetStatusNil sets the value for Status to be an explicit nil

### UnsetStatus
`func (o *TaskDescriptorDummyPay) UnsetStatus()`

UnsetStatus ensures that no value is present for Status, not even an explicit nil
### GetError

`func (o *TaskDescriptorDummyPay) GetError() interface{}`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *TaskDescriptorDummyPay) GetErrorOk() (*interface{}, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *TaskDescriptorDummyPay) SetError(v interface{})`

SetError sets Error field to given value.

### HasError

`func (o *TaskDescriptorDummyPay) HasError() bool`

HasError returns a boolean if a field has been set.

### SetErrorNil

`func (o *TaskDescriptorDummyPay) SetErrorNil(b bool)`

 SetErrorNil sets the value for Error to be an explicit nil

### UnsetError
`func (o *TaskDescriptorDummyPay) UnsetError()`

UnsetError ensures that no value is present for Error, not even an explicit nil
### GetState

`func (o *TaskDescriptorDummyPay) GetState() interface{}`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *TaskDescriptorDummyPay) GetStateOk() (*interface{}, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *TaskDescriptorDummyPay) SetState(v interface{})`

SetState sets State field to given value.

### HasState

`func (o *TaskDescriptorDummyPay) HasState() bool`

HasState returns a boolean if a field has been set.

### SetStateNil

`func (o *TaskDescriptorDummyPay) SetStateNil(b bool)`

 SetStateNil sets the value for State to be an explicit nil

### UnsetState
`func (o *TaskDescriptorDummyPay) UnsetState()`

UnsetState ensures that no value is present for State, not even an explicit nil
### GetDescriptor

`func (o *TaskDescriptorDummyPay) GetDescriptor() TaskDescriptorDummyPayDescriptor`

GetDescriptor returns the Descriptor field if non-nil, zero value otherwise.

### GetDescriptorOk

`func (o *TaskDescriptorDummyPay) GetDescriptorOk() (*TaskDescriptorDummyPayDescriptor, bool)`

GetDescriptorOk returns a tuple with the Descriptor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescriptor

`func (o *TaskDescriptorDummyPay) SetDescriptor(v TaskDescriptorDummyPayDescriptor)`

SetDescriptor sets Descriptor field to given value.

### HasDescriptor

`func (o *TaskDescriptorDummyPay) HasDescriptor() bool`

HasDescriptor returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


