# ConnectorTask

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**OldestId** | Pointer to **string** | The id of the oldest BalanceTransaction fetched from stripe for this account | [optional] 
**OldestDate** | Pointer to **time.Time** | The creation date of the oldest BalanceTransaction fetched from stripe for this account | [optional] 
**MoreRecentId** | Pointer to **string** | The id of the more recent BalanceTransaction fetched from stripe for this account | [optional] 
**MoreRecentDate** | Pointer to **time.Time** | The creation date of the more recent BalanceTransaction fetched from stripe for this account | [optional] 
**NoMoreHistory** | Pointer to **bool** | Indicate that there no more history to fetch on this account | [optional] 

## Methods

### NewConnectorTask

`func NewConnectorTask() *ConnectorTask`

NewConnectorTask instantiates a new ConnectorTask object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewConnectorTaskWithDefaults

`func NewConnectorTaskWithDefaults() *ConnectorTask`

NewConnectorTaskWithDefaults instantiates a new ConnectorTask object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetOldestId

`func (o *ConnectorTask) GetOldestId() string`

GetOldestId returns the OldestId field if non-nil, zero value otherwise.

### GetOldestIdOk

`func (o *ConnectorTask) GetOldestIdOk() (*string, bool)`

GetOldestIdOk returns a tuple with the OldestId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOldestId

`func (o *ConnectorTask) SetOldestId(v string)`

SetOldestId sets OldestId field to given value.

### HasOldestId

`func (o *ConnectorTask) HasOldestId() bool`

HasOldestId returns a boolean if a field has been set.

### GetOldestDate

`func (o *ConnectorTask) GetOldestDate() time.Time`

GetOldestDate returns the OldestDate field if non-nil, zero value otherwise.

### GetOldestDateOk

`func (o *ConnectorTask) GetOldestDateOk() (*time.Time, bool)`

GetOldestDateOk returns a tuple with the OldestDate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOldestDate

`func (o *ConnectorTask) SetOldestDate(v time.Time)`

SetOldestDate sets OldestDate field to given value.

### HasOldestDate

`func (o *ConnectorTask) HasOldestDate() bool`

HasOldestDate returns a boolean if a field has been set.

### GetMoreRecentId

`func (o *ConnectorTask) GetMoreRecentId() string`

GetMoreRecentId returns the MoreRecentId field if non-nil, zero value otherwise.

### GetMoreRecentIdOk

`func (o *ConnectorTask) GetMoreRecentIdOk() (*string, bool)`

GetMoreRecentIdOk returns a tuple with the MoreRecentId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMoreRecentId

`func (o *ConnectorTask) SetMoreRecentId(v string)`

SetMoreRecentId sets MoreRecentId field to given value.

### HasMoreRecentId

`func (o *ConnectorTask) HasMoreRecentId() bool`

HasMoreRecentId returns a boolean if a field has been set.

### GetMoreRecentDate

`func (o *ConnectorTask) GetMoreRecentDate() time.Time`

GetMoreRecentDate returns the MoreRecentDate field if non-nil, zero value otherwise.

### GetMoreRecentDateOk

`func (o *ConnectorTask) GetMoreRecentDateOk() (*time.Time, bool)`

GetMoreRecentDateOk returns a tuple with the MoreRecentDate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMoreRecentDate

`func (o *ConnectorTask) SetMoreRecentDate(v time.Time)`

SetMoreRecentDate sets MoreRecentDate field to given value.

### HasMoreRecentDate

`func (o *ConnectorTask) HasMoreRecentDate() bool`

HasMoreRecentDate returns a boolean if a field has been set.

### GetNoMoreHistory

`func (o *ConnectorTask) GetNoMoreHistory() bool`

GetNoMoreHistory returns the NoMoreHistory field if non-nil, zero value otherwise.

### GetNoMoreHistoryOk

`func (o *ConnectorTask) GetNoMoreHistoryOk() (*bool, bool)`

GetNoMoreHistoryOk returns a tuple with the NoMoreHistory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNoMoreHistory

`func (o *ConnectorTask) SetNoMoreHistory(v bool)`

SetNoMoreHistory sets NoMoreHistory field to given value.

### HasNoMoreHistory

`func (o *ConnectorTask) HasNoMoreHistory() bool`

HasNoMoreHistory returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


