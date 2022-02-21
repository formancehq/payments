# \PaymentsApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ListPayments**](PaymentsApi.md#ListPayments) | **Get** /organizations/{organizationId}/payments | Returns a list of payments.
[**SavePayment**](PaymentsApi.md#SavePayment) | **Put** /organizations/{organizationId}/payments | Persist payment



## ListPayments

> []Payment ListPayments(ctx, organizationId).Limit(limit).Skip(skip).Sort(sort).Execute()

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
    organizationId := "organizationId_example" // string | 
    limit := int32(56) // int32 |  (optional)
    skip := int32(56) // int32 |  (optional)
    sort := []string{"Inner_example"} // []string |  (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.PaymentsApi.ListPayments(context.Background(), organizationId).Limit(limit).Skip(skip).Sort(sort).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `PaymentsApi.ListPayments``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListPayments`: []Payment
    fmt.Fprintf(os.Stdout, "Response from `PaymentsApi.ListPayments`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**organizationId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiListPaymentsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **limit** | **int32** |  | 
 **skip** | **int32** |  | 
 **sort** | **[]string** |  | 

### Return type

[**[]Payment**](Payment.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SavePayment

> SavePayment(ctx, organizationId).PaymentData(paymentData).Execute()

Persist payment

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    openapiclient "./openapi"
)

func main() {
    organizationId := "organizationId_example" // string | 
    paymentData := *openapiclient.NewPaymentData("Provider_example", "Status_example", "Id_example", *openapiclient.NewPaymentDataValue(int32(123), "Asset_example"), time.Now()) // PaymentData | 

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.PaymentsApi.SavePayment(context.Background(), organizationId).PaymentData(paymentData).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `PaymentsApi.SavePayment``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**organizationId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSavePaymentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **paymentData** | [**PaymentData**](PaymentData.md) |  | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

