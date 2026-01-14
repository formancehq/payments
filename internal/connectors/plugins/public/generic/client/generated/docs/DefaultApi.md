# \DefaultApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetAccountBalances**](DefaultApi.md#GetAccountBalances) | **Get** /accounts/{accountId}/balances | Get account balance
[**GetAccounts**](DefaultApi.md#GetAccounts) | **Get** /accounts | Get all accounts
[**GetBeneficiaries**](DefaultApi.md#GetBeneficiaries) | **Get** /beneficiaries | Get all beneficiaries
[**GetTransactions**](DefaultApi.md#GetTransactions) | **Get** /transactions | Get all transactions
[**CreatePayout**](DefaultApi.md#CreatePayout) | **Post** /payouts | Create payout
[**GetPayoutStatus**](DefaultApi.md#GetPayoutStatus) | **Get** /payouts/{payoutId} | Get payout status
[**CreateTransfer**](DefaultApi.md#CreateTransfer) | **Post** /transfers | Create transfer
[**GetTransferStatus**](DefaultApi.md#GetTransferStatus) | **Get** /transfers/{transferId} | Get transfer status
[**CreateBankAccount**](DefaultApi.md#CreateBankAccount) | **Post** /bank-accounts | Create bank account



## GetAccountBalances

> Balances GetAccountBalances(ctx, accountId).Execute()

Get account balance

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "github.com/formancehq/payments/genericclient"
)

func main() {
    accountId := "accountId_example"

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.DefaultApi.GetAccountBalances(context.Background(), accountId).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DefaultApi.GetAccountBalances``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    fmt.Fprintf(os.Stdout, "Response from `DefaultApi.GetAccountBalances`: %v\n", resp)
}
```

### Path Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**accountId** | **string** |  | 

### Return type

[**Balances**](Balances.md)

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAccounts

> []Account GetAccounts(ctx).PageSize(pageSize).Page(page).Sort(sort).CreatedAtFrom(createdAtFrom).Execute()

Get all accounts

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**pageSize** | **int64** | Number of items per page | [default to 100]
**page** | **int64** | Page number | [default to 1]
**sort** | **string** | Sort order | 
**createdAtFrom** | **time.Time** | Filter by created at date | 

### Return type

[**[]Account**](Account.md)

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetBeneficiaries

> []Beneficiary GetBeneficiaries(ctx).PageSize(pageSize).Page(page).Sort(sort).CreatedAtFrom(createdAtFrom).Execute()

Get all beneficiaries (external accounts)

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**pageSize** | **int64** | Number of items per page | [default to 100]
**page** | **int64** | Page number | [default to 1]
**sort** | **string** | Sort order | 
**createdAtFrom** | **time.Time** | Filter by created at date | 

### Return type

[**[]Beneficiary**](Beneficiary.md)

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetTransactions

> []Transaction GetTransactions(ctx).PageSize(pageSize).Page(page).Sort(sort).UpdatedAtFrom(updatedAtFrom).Execute()

Get all transactions

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**pageSize** | **int64** | Number of items per page | [default to 100]
**page** | **int64** | Page number | [default to 1]
**sort** | **string** | Sort order | 
**updatedAtFrom** | **time.Time** | Filter by updated at date | 

### Return type

[**[]Transaction**](Transaction.md)

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreatePayout

> Payout CreatePayout(ctx).PayoutRequest(payoutRequest).Execute()

Create payout

Create an outgoing payment to an external beneficiary account.

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "github.com/formancehq/payments/genericclient"
)

func main() {
    payoutRequest := *openapiclient.NewPayoutRequest(
        "payout-123",
        "10000",
        "USD/2",
        "acc_source_001",
        "ben_dest_002",
    )

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.DefaultApi.CreatePayout(context.Background()).PayoutRequest(payoutRequest).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DefaultApi.CreatePayout``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    fmt.Fprintf(os.Stdout, "Response from `DefaultApi.CreatePayout`: %v\n", resp)
}
```

### Request Body

[**PayoutRequest**](PayoutRequest.md)

### Return type

[**Payout**](Payout.md)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetPayoutStatus

> Payout GetPayoutStatus(ctx, payoutId).Execute()

Get payout status

Retrieve the current status of a payout. Used for polling until the payout reaches a final status.

### Path Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**payoutId** | **string** | The payout ID returned from CreatePayout | 

### Return type

[**Payout**](Payout.md)

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateTransfer

> Transfer CreateTransfer(ctx).TransferRequest(transferRequest).Execute()

Create transfer

Create an internal transfer between two accounts within the PSP.

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "github.com/formancehq/payments/genericclient"
)

func main() {
    transferRequest := *openapiclient.NewTransferRequest(
        "transfer-456",
        "50000",
        "EUR/2",
        "acc_main_001",
        "acc_savings_002",
    )

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.DefaultApi.CreateTransfer(context.Background()).TransferRequest(transferRequest).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DefaultApi.CreateTransfer``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    fmt.Fprintf(os.Stdout, "Response from `DefaultApi.CreateTransfer`: %v\n", resp)
}
```

### Request Body

[**TransferRequest**](TransferRequest.md)

### Return type

[**Transfer**](Transfer.md)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetTransferStatus

> Transfer GetTransferStatus(ctx, transferId).Execute()

Get transfer status

Retrieve the current status of a transfer. Used for polling until the transfer reaches a final status.

### Path Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**transferId** | **string** | The transfer ID returned from CreateTransfer | 

### Return type

[**Transfer**](Transfer.md)

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateBankAccount

> BankAccount CreateBankAccount(ctx).BankAccountRequest(bankAccountRequest).Execute()

Create bank account

Create an external bank account (beneficiary/counterparty) that can be used as a destination for payouts.

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "github.com/formancehq/payments/genericclient"
)

func main() {
    bankAccountRequest := *openapiclient.NewBankAccountRequest("John Doe")
    bankAccountRequest.SetIban("DE89370400440532013000")
    bankAccountRequest.SetSwiftBicCode("COBADEFFXXX")
    bankAccountRequest.SetCountry("DE")

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.DefaultApi.CreateBankAccount(context.Background()).BankAccountRequest(bankAccountRequest).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DefaultApi.CreateBankAccount``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    fmt.Fprintf(os.Stdout, "Response from `DefaultApi.CreateBankAccount`: %v\n", resp)
}
```

### Request Body

[**BankAccountRequest**](BankAccountRequest.md)

### Return type

[**BankAccount**](BankAccount.md)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

