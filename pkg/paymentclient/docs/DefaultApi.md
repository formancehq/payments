# \DefaultApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**PaymentsGet**](DefaultApi.md#PaymentsGet) | **Get** /payments | Returns a list of payments.
[**PaymentsPost**](DefaultApi.md#PaymentsPost) | **Post** /payments | Returns a list of payments.
[**PaymentsPut**](DefaultApi.md#PaymentsPut) | **Put** /payments | Update a payment (can upsert)



## PaymentsGet

> []Payment PaymentsGet(ctx).Limit(limit).Skip(skip).Sort(sort).Execute()

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
    limit := int32(56) // int32 |  (optional)
    skip := int32(56) // int32 |  (optional)
    sort := []string{"Inner_example"} // []string |  (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.DefaultApi.PaymentsGet(context.Background()).Limit(limit).Skip(skip).Sort(sort).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DefaultApi.PaymentsGet``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `PaymentsGet`: []Payment
    fmt.Fprintf(os.Stdout, "Response from `DefaultApi.PaymentsGet`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPaymentsGetRequest struct via the builder pattern


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


## PaymentsPost

> Payment PaymentsPost(ctx).PaymentData(paymentData).Execute()

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
    paymentData := *openapiclient.NewPaymentData("Provider_example", "Status_example", *openapiclient.NewPaymentDataValue(int32(123), "Asset_example"), "Date_example") // PaymentData | 

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.DefaultApi.PaymentsPost(context.Background()).PaymentData(paymentData).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DefaultApi.PaymentsPost``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `PaymentsPost`: Payment
    fmt.Fprintf(os.Stdout, "Response from `DefaultApi.PaymentsPost`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPaymentsPostRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **paymentData** | [**PaymentData**](PaymentData.md) |  | 

### Return type

[**Payment**](Payment.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PaymentsPut

> PaymentsPut(ctx).PaymentData(paymentData).Upsert(upsert).Execute()

Update a payment (can upsert)

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
    paymentData := *openapiclient.NewPaymentData("Provider_example", "Status_example", *openapiclient.NewPaymentDataValue(int32(123), "Asset_example"), "Date_example") // PaymentData | 
    upsert := true // bool |  (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.DefaultApi.PaymentsPut(context.Background()).PaymentData(paymentData).Upsert(upsert).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `DefaultApi.PaymentsPut``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiPaymentsPutRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **paymentData** | [**PaymentData**](PaymentData.md) |  | 
 **upsert** | **bool** |  | 

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

