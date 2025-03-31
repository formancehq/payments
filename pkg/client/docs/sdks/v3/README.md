# V3
(*Payments.V3*)

## Overview

### Available Operations

* [CreateAccount](#createaccount) - Create a formance account object. This object will not be forwarded to the connector. It is only used for internal purposes.

* [ListAccounts](#listaccounts) - List all accounts
* [GetAccount](#getaccount) - Get an account by ID
* [GetAccountBalances](#getaccountbalances) - Get account balances
* [CreateBankAccount](#createbankaccount) - Create a formance bank account object. This object will not be forwarded to the connector until you called the forwardBankAccount method.

* [ListBankAccounts](#listbankaccounts) - List all bank accounts
* [GetBankAccount](#getbankaccount) - Get a Bank Account by ID
* [UpdateBankAccountMetadata](#updatebankaccountmetadata) - Update a bank account's metadata
* [ForwardBankAccount](#forwardbankaccount) - Forward a Bank Account to a PSP for creation
* [ListConnectors](#listconnectors) - List all connectors
* [InstallConnector](#installconnector) - Install a connector
* [ListConnectorConfigs](#listconnectorconfigs) - List all connector configurations
* [UninstallConnector](#uninstallconnector) - Uninstall a connector
* [GetConnectorConfig](#getconnectorconfig) - Get a connector configuration by ID
* [V3UpdateConnectorConfig](#v3updateconnectorconfig) - Update the config of a connector
* [ResetConnector](#resetconnector) - Reset a connector. Be aware that this will delete all data and stop all existing tasks like payment initiations and bank account creations.
* [ListConnectorSchedules](#listconnectorschedules) - List all connector schedules
* [GetConnectorSchedule](#getconnectorschedule) - Get a connector schedule by ID
* [ListConnectorScheduleInstances](#listconnectorscheduleinstances) - List all connector schedule instances
* [CreatePayment](#createpayment) - Create a formance payment object. This object will not be forwarded to the connector. It is only used for internal purposes.

* [ListPayments](#listpayments) - List all payments
* [GetPayment](#getpayment) - Get a payment by ID
* [UpdatePaymentMetadata](#updatepaymentmetadata) - Update a payment's metadata
* [InitiatePayment](#initiatepayment) - Initiate a payment
* [ListPaymentInitiations](#listpaymentinitiations) - List all payment initiations
* [DeletePaymentInitiation](#deletepaymentinitiation) - Delete a payment initiation by ID
* [GetPaymentInitiation](#getpaymentinitiation) - Get a payment initiation by ID
* [RetryPaymentInitiation](#retrypaymentinitiation) - Retry a payment initiation
* [ApprovePaymentInitiation](#approvepaymentinitiation) - Approve a payment initiation
* [RejectPaymentInitiation](#rejectpaymentinitiation) - Reject a payment initiation
* [ReversePaymentInitiation](#reversepaymentinitiation) - Reverse a payment initiation
* [ListPaymentInitiationAdjustments](#listpaymentinitiationadjustments) - List all payment initiation adjustments
* [ListPaymentInitiationRelatedPayments](#listpaymentinitiationrelatedpayments) - List all payments related to a payment initiation
* [CreatePool](#createpool) - Create a formance pool object
* [ListPools](#listpools) - List all pools
* [GetPool](#getpool) - Get a pool by ID
* [DeletePool](#deletepool) - Delete a pool by ID
* [GetPoolBalances](#getpoolbalances) - Get pool balances
* [AddAccountToPool](#addaccounttopool) - Add an account to a pool
* [RemoveAccountFromPool](#removeaccountfrompool) - Remove an account from a pool
* [GetTask](#gettask) - Get a task and its result by ID

## CreateAccount

Create a formance account object. This object will not be forwarded to the connector. It is only used for internal purposes.


### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.CreateAccount(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3CreateAccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                              | Type                                                                                   | Required                                                                               | Description                                                                            |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `ctx`                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                  | :heavy_check_mark:                                                                     | The context to use for the request.                                                    |
| `request`                                                                              | [components.V3CreateAccountRequest](../../models/components/v3createaccountrequest.md) | :heavy_check_mark:                                                                     | The request object to use for the request.                                             |
| `opts`                                                                                 | [][operations.Option](../../models/operations/option.md)                               | :heavy_minus_sign:                                                                     | The options for this request.                                                          |

### Response

**[*operations.V3CreateAccountResponse](../../models/operations/v3createaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListAccounts

List all accounts

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListAccounts(ctx, client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3AccountsCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListAccountsResponse](../../models/operations/v3listaccountsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetAccount

Get an account by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetAccount(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3GetAccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `accountID`                                              | *string*                                                 | :heavy_check_mark:                                       | The account ID                                           |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetAccountResponse](../../models/operations/v3getaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetAccountBalances

Get account balances

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/operations"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetAccountBalances(ctx, operations.V3GetAccountBalancesRequest{
        AccountID: "<id>",
        PageSize: client.Int64(100),
        Cursor: client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="),
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.V3BalancesCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                        | Type                                                                                             | Required                                                                                         | Description                                                                                      |
| ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                            | :heavy_check_mark:                                                                               | The context to use for the request.                                                              |
| `request`                                                                                        | [operations.V3GetAccountBalancesRequest](../../models/operations/v3getaccountbalancesrequest.md) | :heavy_check_mark:                                                                               | The request object to use for the request.                                                       |
| `opts`                                                                                           | [][operations.Option](../../models/operations/option.md)                                         | :heavy_minus_sign:                                                                               | The options for this request.                                                                    |

### Response

**[*operations.V3GetAccountBalancesResponse](../../models/operations/v3getaccountbalancesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## CreateBankAccount

Create a formance bank account object. This object will not be forwarded to the connector until you called the forwardBankAccount method.


### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.CreateBankAccount(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3CreateBankAccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                      | Type                                                                                           | Required                                                                                       | Description                                                                                    |
| ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `ctx`                                                                                          | [context.Context](https://pkg.go.dev/context#Context)                                          | :heavy_check_mark:                                                                             | The context to use for the request.                                                            |
| `request`                                                                                      | [components.V3CreateBankAccountRequest](../../models/components/v3createbankaccountrequest.md) | :heavy_check_mark:                                                                             | The request object to use for the request.                                                     |
| `opts`                                                                                         | [][operations.Option](../../models/operations/option.md)                                       | :heavy_minus_sign:                                                                             | The options for this request.                                                                  |

### Response

**[*operations.V3CreateBankAccountResponse](../../models/operations/v3createbankaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListBankAccounts

List all bank accounts

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListBankAccounts(ctx, client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3BankAccountsCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListBankAccountsResponse](../../models/operations/v3listbankaccountsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetBankAccount

Get a Bank Account by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetBankAccount(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3GetBankAccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `bankAccountID`                                          | *string*                                                 | :heavy_check_mark:                                       | The bank account ID                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetBankAccountResponse](../../models/operations/v3getbankaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## UpdateBankAccountMetadata

Update a bank account's metadata

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.UpdateBankAccountMetadata(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                       | Type                                                                                                            | Required                                                                                                        | Description                                                                                                     |
| --------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                           | [context.Context](https://pkg.go.dev/context#Context)                                                           | :heavy_check_mark:                                                                                              | The context to use for the request.                                                                             |
| `bankAccountID`                                                                                                 | *string*                                                                                                        | :heavy_check_mark:                                                                                              | The bank account ID                                                                                             |
| `v3UpdateBankAccountMetadataRequest`                                                                            | [*components.V3UpdateBankAccountMetadataRequest](../../models/components/v3updatebankaccountmetadatarequest.md) | :heavy_minus_sign:                                                                                              | N/A                                                                                                             |
| `opts`                                                                                                          | [][operations.Option](../../models/operations/option.md)                                                        | :heavy_minus_sign:                                                                                              | The options for this request.                                                                                   |

### Response

**[*operations.V3UpdateBankAccountMetadataResponse](../../models/operations/v3updatebankaccountmetadataresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ForwardBankAccount

Forward a Bank Account to a PSP for creation

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ForwardBankAccount(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ForwardBankAccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                         | Type                                                                                              | Required                                                                                          | Description                                                                                       |
| ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                             | [context.Context](https://pkg.go.dev/context#Context)                                             | :heavy_check_mark:                                                                                | The context to use for the request.                                                               |
| `bankAccountID`                                                                                   | *string*                                                                                          | :heavy_check_mark:                                                                                | The bank account ID                                                                               |
| `v3ForwardBankAccountRequest`                                                                     | [*components.V3ForwardBankAccountRequest](../../models/components/v3forwardbankaccountrequest.md) | :heavy_minus_sign:                                                                                | N/A                                                                                               |
| `opts`                                                                                            | [][operations.Option](../../models/operations/option.md)                                          | :heavy_minus_sign:                                                                                | The options for this request.                                                                     |

### Response

**[*operations.V3ForwardBankAccountResponse](../../models/operations/v3forwardbankaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListConnectors

List all connectors

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListConnectors(ctx, client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ConnectorsCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListConnectorsResponse](../../models/operations/v3listconnectorsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## InstallConnector

Install a connector

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.InstallConnector(ctx, "<value>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3InstallConnectorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                     | Type                                                                                          | Required                                                                                      | Description                                                                                   |
| --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `ctx`                                                                                         | [context.Context](https://pkg.go.dev/context#Context)                                         | :heavy_check_mark:                                                                            | The context to use for the request.                                                           |
| `connector`                                                                                   | *string*                                                                                      | :heavy_check_mark:                                                                            | The connector to filter by                                                                    |
| `v3InstallConnectorRequest`                                                                   | [*components.V3InstallConnectorRequest](../../models/components/v3installconnectorrequest.md) | :heavy_minus_sign:                                                                            | N/A                                                                                           |
| `opts`                                                                                        | [][operations.Option](../../models/operations/option.md)                                      | :heavy_minus_sign:                                                                            | The options for this request.                                                                 |

### Response

**[*operations.V3InstallConnectorResponse](../../models/operations/v3installconnectorresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListConnectorConfigs

List all connector configurations

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListConnectorConfigs(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ConnectorConfigsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3ListConnectorConfigsResponse](../../models/operations/v3listconnectorconfigsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## UninstallConnector

Uninstall a connector

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.UninstallConnector(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3UninstallConnectorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `connectorID`                                            | *string*                                                 | :heavy_check_mark:                                       | The connector ID                                         |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3UninstallConnectorResponse](../../models/operations/v3uninstallconnectorresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetConnectorConfig

Get a connector configuration by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetConnectorConfig(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3GetConnectorConfigResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `connectorID`                                            | *string*                                                 | :heavy_check_mark:                                       | The connector ID                                         |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetConnectorConfigResponse](../../models/operations/v3getconnectorconfigresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## V3UpdateConnectorConfig

Update connector config

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.V3UpdateConnectorConfig(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                   | Type                                                                                        | Required                                                                                    | Description                                                                                 |
| ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `ctx`                                                                                       | [context.Context](https://pkg.go.dev/context#Context)                                       | :heavy_check_mark:                                                                          | The context to use for the request.                                                         |
| `connectorID`                                                                               | *string*                                                                                    | :heavy_check_mark:                                                                          | The connector ID                                                                            |
| `v3UpdateConnectorRequest`                                                                  | [*components.V3UpdateConnectorRequest](../../models/components/v3updateconnectorrequest.md) | :heavy_minus_sign:                                                                          | N/A                                                                                         |
| `opts`                                                                                      | [][operations.Option](../../models/operations/option.md)                                    | :heavy_minus_sign:                                                                          | The options for this request.                                                               |

### Response

**[*operations.V3UpdateConnectorConfigResponse](../../models/operations/v3updateconnectorconfigresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ResetConnector

Reset a connector. Be aware that this will delete all data and stop all existing tasks like payment initiations and bank account creations.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ResetConnector(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ResetConnectorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `connectorID`                                            | *string*                                                 | :heavy_check_mark:                                       | The connector ID                                         |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3ResetConnectorResponse](../../models/operations/v3resetconnectorresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListConnectorSchedules

List all connector schedules

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListConnectorSchedules(ctx, "<id>", client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ConnectorSchedulesCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `connectorID`                                                                                                                                                                                                            | *string*                                                                                                                                                                                                                 | :heavy_check_mark:                                                                                                                                                                                                       | The connector ID                                                                                                                                                                                                         |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListConnectorSchedulesResponse](../../models/operations/v3listconnectorschedulesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetConnectorSchedule

Get a connector schedule by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetConnectorSchedule(ctx, "<id>", "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ConnectorScheduleResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `connectorID`                                            | *string*                                                 | :heavy_check_mark:                                       | The connector ID                                         |
| `scheduleID`                                             | *string*                                                 | :heavy_check_mark:                                       | The schedule ID                                          |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetConnectorScheduleResponse](../../models/operations/v3getconnectorscheduleresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListConnectorScheduleInstances

List all connector schedule instances

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListConnectorScheduleInstances(ctx, "<id>", "<id>", client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="))
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ConnectorScheduleInstancesCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `connectorID`                                                                                                                                                                                                            | *string*                                                                                                                                                                                                                 | :heavy_check_mark:                                                                                                                                                                                                       | The connector ID                                                                                                                                                                                                         |                                                                                                                                                                                                                          |
| `scheduleID`                                                                                                                                                                                                             | *string*                                                                                                                                                                                                                 | :heavy_check_mark:                                                                                                                                                                                                       | The schedule ID                                                                                                                                                                                                          |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListConnectorScheduleInstancesResponse](../../models/operations/v3listconnectorscheduleinstancesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## CreatePayment

Create a formance payment object. This object will not be forwarded to the connector. It is only used for internal purposes.


### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.CreatePayment(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3CreatePaymentResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                              | Type                                                                                   | Required                                                                               | Description                                                                            |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `ctx`                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                  | :heavy_check_mark:                                                                     | The context to use for the request.                                                    |
| `request`                                                                              | [components.V3CreatePaymentRequest](../../models/components/v3createpaymentrequest.md) | :heavy_check_mark:                                                                     | The request object to use for the request.                                             |
| `opts`                                                                                 | [][operations.Option](../../models/operations/option.md)                               | :heavy_minus_sign:                                                                     | The options for this request.                                                          |

### Response

**[*operations.V3CreatePaymentResponse](../../models/operations/v3createpaymentresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListPayments

List all payments

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListPayments(ctx, client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3PaymentsCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListPaymentsResponse](../../models/operations/v3listpaymentsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetPayment

Get a payment by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetPayment(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3GetPaymentResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `paymentID`                                              | *string*                                                 | :heavy_check_mark:                                       | The payment ID                                           |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetPaymentResponse](../../models/operations/v3getpaymentresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## UpdatePaymentMetadata

Update a payment's metadata

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.UpdatePaymentMetadata(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                               | Type                                                                                                    | Required                                                                                                | Description                                                                                             |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                   | [context.Context](https://pkg.go.dev/context#Context)                                                   | :heavy_check_mark:                                                                                      | The context to use for the request.                                                                     |
| `paymentID`                                                                                             | *string*                                                                                                | :heavy_check_mark:                                                                                      | The payment ID                                                                                          |
| `v3UpdatePaymentMetadataRequest`                                                                        | [*components.V3UpdatePaymentMetadataRequest](../../models/components/v3updatepaymentmetadatarequest.md) | :heavy_minus_sign:                                                                                      | N/A                                                                                                     |
| `opts`                                                                                                  | [][operations.Option](../../models/operations/option.md)                                                | :heavy_minus_sign:                                                                                      | The options for this request.                                                                           |

### Response

**[*operations.V3UpdatePaymentMetadataResponse](../../models/operations/v3updatepaymentmetadataresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## InitiatePayment

Initiate a payment

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.InitiatePayment(ctx, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3InitiatePaymentResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                         | Type                                                                                                                              | Required                                                                                                                          | Description                                                                                                                       |
| --------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                             | [context.Context](https://pkg.go.dev/context#Context)                                                                             | :heavy_check_mark:                                                                                                                | The context to use for the request.                                                                                               |
| `noValidation`                                                                                                                    | **bool*                                                                                                                           | :heavy_minus_sign:                                                                                                                | If set to true, the request will not have to be validated. This is useful if we want to directly forward the request to the PSP.<br/> |
| `v3InitiatePaymentRequest`                                                                                                        | [*components.V3InitiatePaymentRequest](../../models/components/v3initiatepaymentrequest.md)                                       | :heavy_minus_sign:                                                                                                                | N/A                                                                                                                               |
| `opts`                                                                                                                            | [][operations.Option](../../models/operations/option.md)                                                                          | :heavy_minus_sign:                                                                                                                | The options for this request.                                                                                                     |

### Response

**[*operations.V3InitiatePaymentResponse](../../models/operations/v3initiatepaymentresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListPaymentInitiations

List all payment initiations

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListPaymentInitiations(ctx, client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3PaymentInitiationsCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListPaymentInitiationsResponse](../../models/operations/v3listpaymentinitiationsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## DeletePaymentInitiation

Delete a payment initiation by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.DeletePaymentInitiation(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `paymentInitiationID`                                    | *string*                                                 | :heavy_check_mark:                                       | The payment initiation ID                                |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3DeletePaymentInitiationResponse](../../models/operations/v3deletepaymentinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetPaymentInitiation

Get a payment initiation by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetPaymentInitiation(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3GetPaymentInitiationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `paymentInitiationID`                                    | *string*                                                 | :heavy_check_mark:                                       | The payment initiation ID                                |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetPaymentInitiationResponse](../../models/operations/v3getpaymentinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## RetryPaymentInitiation

Retry a payment initiation

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.RetryPaymentInitiation(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3RetryPaymentInitiationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `paymentInitiationID`                                    | *string*                                                 | :heavy_check_mark:                                       | The payment initiation ID                                |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3RetryPaymentInitiationResponse](../../models/operations/v3retrypaymentinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ApprovePaymentInitiation

Approve a payment initiation

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ApprovePaymentInitiation(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ApprovePaymentInitiationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `paymentInitiationID`                                    | *string*                                                 | :heavy_check_mark:                                       | The payment initiation ID                                |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3ApprovePaymentInitiationResponse](../../models/operations/v3approvepaymentinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## RejectPaymentInitiation

Reject a payment initiation

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.RejectPaymentInitiation(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `paymentInitiationID`                                    | *string*                                                 | :heavy_check_mark:                                       | The payment initiation ID                                |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3RejectPaymentInitiationResponse](../../models/operations/v3rejectpaymentinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ReversePaymentInitiation

Reverse a payment initiation

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ReversePaymentInitiation(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3ReversePaymentInitiationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                     | Type                                                                                                          | Required                                                                                                      | Description                                                                                                   |
| ------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                         | [context.Context](https://pkg.go.dev/context#Context)                                                         | :heavy_check_mark:                                                                                            | The context to use for the request.                                                                           |
| `paymentInitiationID`                                                                                         | *string*                                                                                                      | :heavy_check_mark:                                                                                            | The payment initiation ID                                                                                     |
| `v3ReversePaymentInitiationRequest`                                                                           | [*components.V3ReversePaymentInitiationRequest](../../models/components/v3reversepaymentinitiationrequest.md) | :heavy_minus_sign:                                                                                            | N/A                                                                                                           |
| `opts`                                                                                                        | [][operations.Option](../../models/operations/option.md)                                                      | :heavy_minus_sign:                                                                                            | The options for this request.                                                                                 |

### Response

**[*operations.V3ReversePaymentInitiationResponse](../../models/operations/v3reversepaymentinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListPaymentInitiationAdjustments

List all payment initiation adjustments

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListPaymentInitiationAdjustments(ctx, "<id>", client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3PaymentInitiationAdjustmentsCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `paymentInitiationID`                                                                                                                                                                                                    | *string*                                                                                                                                                                                                                 | :heavy_check_mark:                                                                                                                                                                                                       | The payment initiation ID                                                                                                                                                                                                |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListPaymentInitiationAdjustmentsResponse](../../models/operations/v3listpaymentinitiationadjustmentsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListPaymentInitiationRelatedPayments

List all payments related to a payment initiation

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListPaymentInitiationRelatedPayments(ctx, "<id>", client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3PaymentInitiationRelatedPaymentsCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `paymentInitiationID`                                                                                                                                                                                                    | *string*                                                                                                                                                                                                                 | :heavy_check_mark:                                                                                                                                                                                                       | The payment initiation ID                                                                                                                                                                                                |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListPaymentInitiationRelatedPaymentsResponse](../../models/operations/v3listpaymentinitiationrelatedpaymentsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## CreatePool

Create a formance pool object

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.CreatePool(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3CreatePoolResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [components.V3CreatePoolRequest](../../models/components/v3createpoolrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.V3CreatePoolResponse](../../models/operations/v3createpoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListPools

List all pools

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.ListPools(ctx, client.Int64(100), client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3PoolsCursorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                | Type                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                 | Description                                                                                                                                                                                                              | Example                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The number of items to return                                                                                                                                                                                            | 100                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                       | Parameter used in pagination requests. Set to the value of next for the next page of results. Set to the value of previous for the previous page of results. No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                             |
| `requestBody`                                                                                                                                                                                                            | map[string]*any*                                                                                                                                                                                                         | :heavy_minus_sign:                                                                                                                                                                                                       | N/A                                                                                                                                                                                                                      |                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                            |                                                                                                                                                                                                                          |

### Response

**[*operations.V3ListPoolsResponse](../../models/operations/v3listpoolsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetPool

Get a pool by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetPool(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3GetPoolResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID                                              |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetPoolResponse](../../models/operations/v3getpoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## DeletePool

Delete a pool by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.DeletePool(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID                                              |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3DeletePoolResponse](../../models/operations/v3deletepoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetPoolBalances

Get pool balances

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetPoolBalances(ctx, "<id>", nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V3PoolBalancesResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID                                              |
| `at`                                                     | [*time.Time](https://pkg.go.dev/time#Time)               | :heavy_minus_sign:                                       | The time to filter by                                    |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetPoolBalancesResponse](../../models/operations/v3getpoolbalancesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## AddAccountToPool

Add an account to a pool

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.AddAccountToPool(ctx, "<id>", "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID                                              |
| `accountID`                                              | *string*                                                 | :heavy_check_mark:                                       | The account ID                                           |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3AddAccountToPoolResponse](../../models/operations/v3addaccounttopoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## RemoveAccountFromPool

Remove an account from a pool

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.RemoveAccountFromPool(ctx, "<id>", "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID                                              |
| `accountID`                                              | *string*                                                 | :heavy_check_mark:                                       | The account ID                                           |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3RemoveAccountFromPoolResponse](../../models/operations/v3removeaccountfrompoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetTask

Get a task and its result by ID

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V3.GetTask(ctx, "<id>")
    if err != nil {
        log.Fatal(err)
    }
    if res.V3GetTaskResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `taskID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The task ID                                              |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |

### Response

**[*operations.V3GetTaskResponse](../../models/operations/v3gettaskresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |