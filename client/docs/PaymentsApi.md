# \PaymentsApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetPayment**](PaymentsApi.md#GetPayment) | **Get** /payments/{paymentId} | Returns a payment.
[**ListPayments**](PaymentsApi.md#ListPayments) | **Get** /payments | Returns a list of payments.



## GetPayment

> Payment GetPayment(ctx, paymentId).Execute()

Returns a payment.

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    paymentId := "XXX" // string | The payment id

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.PaymentsApi.GetPayment(context.Background(), paymentId).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `PaymentsApi.GetPayment``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetPayment`: Payment
    fmt.Fprintf(os.Stdout, "Response from `PaymentsApi.GetPayment`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**paymentId** | **string** | The payment id | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetPaymentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**Payment**](Payment.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListPayments

> ListPaymentsResponse ListPayments(ctx).Limit(limit).Skip(skip).Sort(sort).Execute()

Returns a list of payments.

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    limit := int32(10) // int32 | Limit the number of payments to return, pagination can be achieved in conjunction with 'skip' parameter. (optional)
    skip := int32(100) // int32 | How many payments to skip, pagination can be achieved in conjunction with 'limit' parameter. (optional)
    sort := []string{"Inner_example"} // []string | Field used to sort payments (Default is by date). (optional)

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.PaymentsApi.ListPayments(context.Background()).Limit(limit).Skip(skip).Sort(sort).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `PaymentsApi.ListPayments``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListPayments`: ListPaymentsResponse
    fmt.Fprintf(os.Stdout, "Response from `PaymentsApi.ListPayments`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListPaymentsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **limit** | **int32** | Limit the number of payments to return, pagination can be achieved in conjunction with &#39;skip&#39; parameter. | 
 **skip** | **int32** | How many payments to skip, pagination can be achieved in conjunction with &#39;limit&#39; parameter. | 
 **sort** | **[]string** | Field used to sort payments (Default is by date). | 

### Return type

[**ListPaymentsResponse**](ListPaymentsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

