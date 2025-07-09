# V1
(*Payments.V1*)

## Overview

### Available Operations

* [GetServerInfo](#getserverinfo) - Get server info
* [CreatePayment](#createpayment) - Create a payment
* [ListPayments](#listpayments) - List payments
* [GetPayment](#getpayment) - Get a payment
* [UpdateMetadata](#updatemetadata) - Update metadata
* [ListTransferInitiations](#listtransferinitiations) - List Transfer Initiations
* [CreateTransferInitiation](#createtransferinitiation) - Create a TransferInitiation
* [GetTransferInitiation](#gettransferinitiation) - Get a transfer initiation
* [DeleteTransferInitiation](#deletetransferinitiation) - Delete a transfer initiation
* [UpdateTransferInitiationStatus](#updatetransferinitiationstatus) - Update the status of a transfer initiation
* [ReverseTransferInitiation](#reversetransferinitiation) - Reverse a transfer initiation
* [RetryTransferInitiation](#retrytransferinitiation) - Retry a failed transfer initiation
* [ListPools](#listpools) - List Pools
* [CreatePool](#createpool) - Create a Pool
* [GetPool](#getpool) - Get a Pool
* [DeletePool](#deletepool) - Delete a Pool
* [AddAccountToPool](#addaccounttopool) - Add an account to a pool
* [RemoveAccountFromPool](#removeaccountfrompool) - Remove an account from a pool
* [GetPoolBalances](#getpoolbalances) - Get historical pool balances at a particular point in time
* [GetPoolBalancesLatest](#getpoolbalanceslatest) - Get latest pool balances
* [CreateAccount](#createaccount) - Create an account
* [ListAccounts](#listaccounts) - List accounts
* [GetAccount](#getaccount) - Get an account
* [GetAccountBalances](#getaccountbalances) - Get account balances
* [CreateBankAccount](#createbankaccount) - Create a BankAccount in Payments and on the PSP
* [ListBankAccounts](#listbankaccounts) - List bank accounts created by user on Formance
* [GetBankAccount](#getbankaccount) - Get a bank account created by user on Formance
* [ForwardBankAccount](#forwardbankaccount) - Forward a bank account to a connector
* [UpdateBankAccountMetadata](#updatebankaccountmetadata) - Update metadata of a bank account
* [ListAllConnectors](#listallconnectors) - List all installed connectors
* [ListConfigsAvailableConnectors](#listconfigsavailableconnectors) - List the configs of each available connector
* [InstallConnector](#installconnector) - Install a connector
* [~~UninstallConnector~~](#uninstallconnector) - Uninstall a connector :warning: **Deprecated**
* [UninstallConnectorV1](#uninstallconnectorv1) - Uninstall a connector
* [~~ReadConnectorConfig~~](#readconnectorconfig) - Read the config of a connector :warning: **Deprecated**
* [UpdateConnectorConfigV1](#updateconnectorconfigv1) - Update the config of a connector
* [ReadConnectorConfigV1](#readconnectorconfigv1) - Read the config of a connector
* [~~ResetConnector~~](#resetconnector) - Reset a connector :warning: **Deprecated**
* [ResetConnectorV1](#resetconnectorv1) - Reset a connector
* [~~ListConnectorTasks~~](#listconnectortasks) - List tasks from a connector :warning: **Deprecated**
* [ListConnectorTasksV1](#listconnectortasksv1) - List tasks from a connector
* [~~GetConnectorTask~~](#getconnectortask) - Read a specific task of the connector :warning: **Deprecated**
* [GetConnectorTaskV1](#getconnectortaskv1) - Read a specific task of the connector
* [ConnectorsTransfer](#connectorstransfer) - Transfer funds between Connector accounts

## GetServerInfo

Get server info

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

    res, err := s.Payments.V1.GetServerInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ServerInfo != nil {
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

**[*operations.GetServerInfoResponse](../../models/operations/getserverinforesponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## CreatePayment

Create a payment

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/types"
	"math/big"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.CreatePayment(ctx, components.PaymentRequest{
        Reference: "<value>",
        ConnectorID: "<id>",
        CreatedAt: types.MustTimeFromString("2025-11-09T01:03:21.011Z"),
        Amount: big.NewInt(100),
        Type: components.PaymentTypeOther,
        Status: components.PaymentStatusRefundedFailure,
        Scheme: components.PaymentSchemeSepaDebit,
        Asset: "USD",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.PaymentResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                              | Type                                                                   | Required                                                               | Description                                                            |
| ---------------------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `ctx`                                                                  | [context.Context](https://pkg.go.dev/context#Context)                  | :heavy_check_mark:                                                     | The context to use for the request.                                    |
| `request`                                                              | [components.PaymentRequest](../../models/components/paymentrequest.md) | :heavy_check_mark:                                                     | The request object to use for the request.                             |
| `opts`                                                                 | [][operations.Option](../../models/operations/option.md)               | :heavy_minus_sign:                                                     | The options for this request.                                          |

### Response

**[*operations.CreatePaymentResponse](../../models/operations/createpaymentresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListPayments

List payments

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

    res, err := s.Payments.V1.ListPayments(ctx, nil, client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), []string{
        "date:asc",
        "status:desc",
    }, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.PaymentsCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                                                | Type                                                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                                                 | Description                                                                                                                                                                                                                                              | Example                                                                                                                                                                                                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The maximum number of results to return per page.<br/>                                                                                                                                                                                                   | 100                                                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Parameter used in pagination requests. Maximum page size is set to 15.<br/>Set to the value of next for the next page of results.<br/>Set to the value of previous for the previous page of results.<br/>No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                                                             |
| `sort`                                                                                                                                                                                                                                                   | []*string*                                                                                                                                                                                                                                               | :heavy_minus_sign:                                                                                                                                                                                                                                       | Fields used to sort payments (default is date:desc).                                                                                                                                                                                                     | [<br/>"date:asc",<br/>"status:desc"<br/>]                                                                                                                                                                                                                |
| `query`                                                                                                                                                                                                                                                  | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Filters used to filter resources.<br/>                                                                                                                                                                                                                   |                                                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                                                            |                                                                                                                                                                                                                                                          |

### Response

**[*operations.ListPaymentsResponse](../../models/operations/listpaymentsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetPayment

Get a payment

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

    res, err := s.Payments.V1.GetPayment(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res.PaymentResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `paymentID`                                              | *string*                                                 | :heavy_check_mark:                                       | The payment ID.                                          | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.GetPaymentResponse](../../models/operations/getpaymentresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## UpdateMetadata

Update metadata

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

    res, err := s.Payments.V1.UpdateMetadata(ctx, "XXX", map[string]string{
        "key": "<value>",
        "key1": "<value>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `paymentID`                                              | *string*                                                 | :heavy_check_mark:                                       | The payment ID.                                          | XXX                                                      |
| `requestBody`                                            | map[string]*string*                                      | :heavy_check_mark:                                       | N/A                                                      |                                                          |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.UpdateMetadataResponse](../../models/operations/updatemetadataresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListTransferInitiations

List Transfer Initiations

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

    res, err := s.Payments.V1.ListTransferInitiations(ctx, nil, client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), []string{
        "date:asc",
        "status:desc",
    }, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.TransferInitiationsCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                                                | Type                                                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                                                 | Description                                                                                                                                                                                                                                              | Example                                                                                                                                                                                                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The maximum number of results to return per page.<br/>                                                                                                                                                                                                   | 100                                                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Parameter used in pagination requests. Maximum page size is set to 15.<br/>Set to the value of next for the next page of results.<br/>Set to the value of previous for the previous page of results.<br/>No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                                                             |
| `sort`                                                                                                                                                                                                                                                   | []*string*                                                                                                                                                                                                                                               | :heavy_minus_sign:                                                                                                                                                                                                                                       | Fields used to sort payments (default is date:desc).                                                                                                                                                                                                     | [<br/>"date:asc",<br/>"status:desc"<br/>]                                                                                                                                                                                                                |
| `query`                                                                                                                                                                                                                                                  | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Filters used to filter resources.<br/>                                                                                                                                                                                                                   |                                                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                                                            |                                                                                                                                                                                                                                                          |

### Response

**[*operations.ListTransferInitiationsResponse](../../models/operations/listtransferinitiationsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## CreateTransferInitiation

Create a transfer initiation

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/types"
	"github.com/formancehq/payments/pkg/client/models/components"
	"math/big"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.CreateTransferInitiation(ctx, components.TransferInitiationRequest{
        Reference: "XXX",
        ScheduledAt: types.MustTimeFromString("2023-10-09T08:11:40.585Z"),
        Description: "worthy pace vague ick liberalize between um",
        SourceAccountID: "<id>",
        DestinationAccountID: "<id>",
        Type: components.TransferInitiationRequestTypePayout,
        Amount: big.NewInt(847873),
        Asset: "USD",
        Validated: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.TransferInitiationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |
| `request`                                                                                    | [components.TransferInitiationRequest](../../models/components/transferinitiationrequest.md) | :heavy_check_mark:                                                                           | The request object to use for the request.                                                   |
| `opts`                                                                                       | [][operations.Option](../../models/operations/option.md)                                     | :heavy_minus_sign:                                                                           | The options for this request.                                                                |

### Response

**[*operations.CreateTransferInitiationResponse](../../models/operations/createtransferinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetTransferInitiation

Get a transfer initiation

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

    res, err := s.Payments.V1.GetTransferInitiation(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res.TransferInitiationResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `transferID`                                             | *string*                                                 | :heavy_check_mark:                                       | The transfer ID.                                         | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.GetTransferInitiationResponse](../../models/operations/gettransferinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## DeleteTransferInitiation

Delete a transfer initiation by its id.

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

    res, err := s.Payments.V1.DeleteTransferInitiation(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `transferID`                                             | *string*                                                 | :heavy_check_mark:                                       | The transfer ID.                                         | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.DeleteTransferInitiationResponse](../../models/operations/deletetransferinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## UpdateTransferInitiationStatus

Update a transfer initiation status

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.UpdateTransferInitiationStatus(ctx, "XXX", components.UpdateTransferInitiationStatusRequest{
        Status: components.StatusValidated,
    })
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                            | Type                                                                                                                 | Required                                                                                                             | Description                                                                                                          | Example                                                                                                              |
| -------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                                | :heavy_check_mark:                                                                                                   | The context to use for the request.                                                                                  |                                                                                                                      |
| `transferID`                                                                                                         | *string*                                                                                                             | :heavy_check_mark:                                                                                                   | The transfer ID.                                                                                                     | XXX                                                                                                                  |
| `updateTransferInitiationStatusRequest`                                                                              | [components.UpdateTransferInitiationStatusRequest](../../models/components/updatetransferinitiationstatusrequest.md) | :heavy_check_mark:                                                                                                   | N/A                                                                                                                  |                                                                                                                      |
| `opts`                                                                                                               | [][operations.Option](../../models/operations/option.md)                                                             | :heavy_minus_sign:                                                                                                   | The options for this request.                                                                                        |                                                                                                                      |

### Response

**[*operations.UpdateTransferInitiationStatusResponse](../../models/operations/updatetransferinitiationstatusresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ReverseTransferInitiation

Reverse transfer initiation

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"math/big"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ReverseTransferInitiation(ctx, "XXX", components.ReverseTransferInitiationRequest{
        Reference: "XXX",
        Description: "emerge whose mechanically outside kissingly",
        Amount: big.NewInt(978360),
        Asset: "USD",
        Metadata: map[string]string{
            "key": "<value>",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                  | Type                                                                                                       | Required                                                                                                   | Description                                                                                                | Example                                                                                                    |
| ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                                      | :heavy_check_mark:                                                                                         | The context to use for the request.                                                                        |                                                                                                            |
| `transferID`                                                                                               | *string*                                                                                                   | :heavy_check_mark:                                                                                         | The transfer ID.                                                                                           | XXX                                                                                                        |
| `reverseTransferInitiationRequest`                                                                         | [components.ReverseTransferInitiationRequest](../../models/components/reversetransferinitiationrequest.md) | :heavy_check_mark:                                                                                         | N/A                                                                                                        |                                                                                                            |
| `opts`                                                                                                     | [][operations.Option](../../models/operations/option.md)                                                   | :heavy_minus_sign:                                                                                         | The options for this request.                                                                              |                                                                                                            |

### Response

**[*operations.ReverseTransferInitiationResponse](../../models/operations/reversetransferinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## RetryTransferInitiation

Retry a failed transfer initiation

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

    res, err := s.Payments.V1.RetryTransferInitiation(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `transferID`                                             | *string*                                                 | :heavy_check_mark:                                       | The transfer ID.                                         | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.RetryTransferInitiationResponse](../../models/operations/retrytransferinitiationresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListPools

List Pools

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

    res, err := s.Payments.V1.ListPools(ctx, nil, client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), []string{
        "date:asc",
        "status:desc",
    }, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.PoolsCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                                                | Type                                                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                                                 | Description                                                                                                                                                                                                                                              | Example                                                                                                                                                                                                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The maximum number of results to return per page.<br/>                                                                                                                                                                                                   | 100                                                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Parameter used in pagination requests. Maximum page size is set to 15.<br/>Set to the value of next for the next page of results.<br/>Set to the value of previous for the previous page of results.<br/>No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                                                             |
| `sort`                                                                                                                                                                                                                                                   | []*string*                                                                                                                                                                                                                                               | :heavy_minus_sign:                                                                                                                                                                                                                                       | Fields used to sort payments (default is date:desc).                                                                                                                                                                                                     | [<br/>"date:asc",<br/>"status:desc"<br/>]                                                                                                                                                                                                                |
| `query`                                                                                                                                                                                                                                                  | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Filters used to filter resources.<br/>                                                                                                                                                                                                                   |                                                                                                                                                                                                                                                          |
| `opts`                                                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                                                            |                                                                                                                                                                                                                                                          |

### Response

**[*operations.ListPoolsResponse](../../models/operations/listpoolsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## CreatePool

Create a Pool

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.CreatePool(ctx, components.PoolRequest{
        Name: "<value>",
        AccountIDs: []string{
            "<value>",
            "<value>",
            "<value>",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.PoolResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                        | Type                                                             | Required                                                         | Description                                                      |
| ---------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------- |
| `ctx`                                                            | [context.Context](https://pkg.go.dev/context#Context)            | :heavy_check_mark:                                               | The context to use for the request.                              |
| `request`                                                        | [components.PoolRequest](../../models/components/poolrequest.md) | :heavy_check_mark:                                               | The request object to use for the request.                       |
| `opts`                                                           | [][operations.Option](../../models/operations/option.md)         | :heavy_minus_sign:                                               | The options for this request.                                    |

### Response

**[*operations.CreatePoolResponse](../../models/operations/createpoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetPool

Get a Pool

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

    res, err := s.Payments.V1.GetPool(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res.PoolResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID.                                             | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.GetPoolResponse](../../models/operations/getpoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## DeletePool

Delete a pool by its id.

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

    res, err := s.Payments.V1.DeletePool(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID.                                             | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.DeletePoolResponse](../../models/operations/deletepoolresponse.md), error**

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
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.AddAccountToPool(ctx, "XXX", components.AddAccountToPoolRequest{
        AccountID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              | Example                                                                                  |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |                                                                                          |
| `poolID`                                                                                 | *string*                                                                                 | :heavy_check_mark:                                                                       | The pool ID.                                                                             | XXX                                                                                      |
| `addAccountToPoolRequest`                                                                | [components.AddAccountToPoolRequest](../../models/components/addaccounttopoolrequest.md) | :heavy_check_mark:                                                                       | N/A                                                                                      |                                                                                          |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |                                                                                          |

### Response

**[*operations.AddAccountToPoolResponse](../../models/operations/addaccounttopoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## RemoveAccountFromPool

Remove an account from a pool by its id.

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

    res, err := s.Payments.V1.RemoveAccountFromPool(ctx, "XXX", "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID.                                             | XXX                                                      |
| `accountID`                                              | *string*                                                 | :heavy_check_mark:                                       | The account ID.                                          | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.RemoveAccountFromPoolResponse](../../models/operations/removeaccountfrompoolresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetPoolBalances

Get historical pool balances at a particular point in time

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/types"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.GetPoolBalances(ctx, "XXX", types.MustTimeFromString("2024-05-04T06:40:23.119Z"))
    if err != nil {
        log.Fatal(err)
    }
    if res.PoolBalancesResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID.                                             | XXX                                                      |
| `at`                                                     | [time.Time](https://pkg.go.dev/time#Time)                | :heavy_check_mark:                                       | Filter balances by date.<br/>                            |                                                          |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.GetPoolBalancesResponse](../../models/operations/getpoolbalancesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetPoolBalancesLatest

Get latest pool balances

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

    res, err := s.Payments.V1.GetPoolBalancesLatest(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res.PoolBalancesLatestResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `poolID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The pool ID.                                             | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.GetPoolBalancesLatestResponse](../../models/operations/getpoolbalanceslatestresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## CreateAccount

Create an account

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/types"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.CreateAccount(ctx, components.AccountRequest{
        Reference: "<value>",
        ConnectorID: "<id>",
        CreatedAt: types.MustTimeFromString("2025-08-19T02:15:08.152Z"),
        Type: components.AccountTypeInternal,
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.AccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                              | Type                                                                   | Required                                                               | Description                                                            |
| ---------------------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `ctx`                                                                  | [context.Context](https://pkg.go.dev/context#Context)                  | :heavy_check_mark:                                                     | The context to use for the request.                                    |
| `request`                                                              | [components.AccountRequest](../../models/components/accountrequest.md) | :heavy_check_mark:                                                     | The request object to use for the request.                             |
| `opts`                                                                 | [][operations.Option](../../models/operations/option.md)               | :heavy_minus_sign:                                                     | The options for this request.                                          |

### Response

**[*operations.CreateAccountResponse](../../models/operations/createaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListAccounts

List accounts

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

    res, err := s.Payments.V1.ListAccounts(ctx, operations.ListAccountsRequest{
        Cursor: client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="),
        Sort: []string{
            "date:asc",
            "status:desc",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.AccountsCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.ListAccountsRequest](../../models/operations/listaccountsrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |

### Response

**[*operations.ListAccountsResponse](../../models/operations/listaccountsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetAccount

Get an account

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

    res, err := s.Payments.V1.GetAccount(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res.AccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `accountID`                                              | *string*                                                 | :heavy_check_mark:                                       | The account ID.                                          | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.GetAccountResponse](../../models/operations/getaccountresponse.md), error**

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

    res, err := s.Payments.V1.GetAccountBalances(ctx, operations.GetAccountBalancesRequest{
        AccountID: "XXX",
        Cursor: client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="),
        Sort: []string{
            "date:asc",
            "status:desc",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.BalancesCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |
| `request`                                                                                    | [operations.GetAccountBalancesRequest](../../models/operations/getaccountbalancesrequest.md) | :heavy_check_mark:                                                                           | The request object to use for the request.                                                   |
| `opts`                                                                                       | [][operations.Option](../../models/operations/option.md)                                     | :heavy_minus_sign:                                                                           | The options for this request.                                                                |

### Response

**[*operations.GetAccountBalancesResponse](../../models/operations/getaccountbalancesresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## CreateBankAccount

Create a bank account in Payments and on the PSP.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.CreateBankAccount(ctx, components.BankAccountRequest{
        Country: "GB",
        ConnectorID: client.String("<id>"),
        Name: "My account",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.BankAccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [components.BankAccountRequest](../../models/components/bankaccountrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |

### Response

**[*operations.CreateBankAccountResponse](../../models/operations/createbankaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListBankAccounts

List all bank accounts created by user on Formance.

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

    res, err := s.Payments.V1.ListBankAccounts(ctx, nil, client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="), []string{
        "date:asc",
        "status:desc",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.BankAccountsCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                                                | Type                                                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                                                 | Description                                                                                                                                                                                                                                              | Example                                                                                                                                                                                                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The maximum number of results to return per page.<br/>                                                                                                                                                                                                   | 100                                                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Parameter used in pagination requests. Maximum page size is set to 15.<br/>Set to the value of next for the next page of results.<br/>Set to the value of previous for the previous page of results.<br/>No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                                                             |
| `sort`                                                                                                                                                                                                                                                   | []*string*                                                                                                                                                                                                                                               | :heavy_minus_sign:                                                                                                                                                                                                                                       | Fields used to sort payments (default is date:desc).                                                                                                                                                                                                     | [<br/>"date:asc",<br/>"status:desc"<br/>]                                                                                                                                                                                                                |
| `opts`                                                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                                                            |                                                                                                                                                                                                                                                          |

### Response

**[*operations.ListBankAccountsResponse](../../models/operations/listbankaccountsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetBankAccount

Get a bank account created by user on Formance

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

    res, err := s.Payments.V1.GetBankAccount(ctx, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res.BankAccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              | Example                                                  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |                                                          |
| `bankAccountID`                                          | *string*                                                 | :heavy_check_mark:                                       | The bank account ID.                                     | XXX                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |                                                          |

### Response

**[*operations.GetBankAccountResponse](../../models/operations/getbankaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ForwardBankAccount

Forward a bank account to a connector

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ForwardBankAccount(ctx, "XXX", components.ForwardBankAccountRequest{
        ConnectorID: "<id>",
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.BankAccountResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  | Example                                                                                      |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |                                                                                              |
| `bankAccountID`                                                                              | *string*                                                                                     | :heavy_check_mark:                                                                           | The bank account ID.                                                                         | XXX                                                                                          |
| `forwardBankAccountRequest`                                                                  | [components.ForwardBankAccountRequest](../../models/components/forwardbankaccountrequest.md) | :heavy_check_mark:                                                                           | N/A                                                                                          |                                                                                              |
| `opts`                                                                                       | [][operations.Option](../../models/operations/option.md)                                     | :heavy_minus_sign:                                                                           | The options for this request.                                                                |                                                                                              |

### Response

**[*operations.ForwardBankAccountResponse](../../models/operations/forwardbankaccountresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## UpdateBankAccountMetadata

Update metadata of a bank account

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.UpdateBankAccountMetadata(ctx, "XXX", components.UpdateBankAccountMetadataRequest{
        Metadata: map[string]string{
            "key": "<value>",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                  | Type                                                                                                       | Required                                                                                                   | Description                                                                                                | Example                                                                                                    |
| ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                                      | :heavy_check_mark:                                                                                         | The context to use for the request.                                                                        |                                                                                                            |
| `bankAccountID`                                                                                            | *string*                                                                                                   | :heavy_check_mark:                                                                                         | The bank account ID.                                                                                       | XXX                                                                                                        |
| `updateBankAccountMetadataRequest`                                                                         | [components.UpdateBankAccountMetadataRequest](../../models/components/updatebankaccountmetadatarequest.md) | :heavy_check_mark:                                                                                         | N/A                                                                                                        |                                                                                                            |
| `opts`                                                                                                     | [][operations.Option](../../models/operations/option.md)                                                   | :heavy_minus_sign:                                                                                         | The options for this request.                                                                              |                                                                                                            |

### Response

**[*operations.UpdateBankAccountMetadataResponse](../../models/operations/updatebankaccountmetadataresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListAllConnectors

List all installed connectors.

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

    res, err := s.Payments.V1.ListAllConnectors(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ConnectorsResponse != nil {
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

**[*operations.ListAllConnectorsResponse](../../models/operations/listallconnectorsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListConfigsAvailableConnectors

List the configs of each available connector.

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

    res, err := s.Payments.V1.ListConfigsAvailableConnectors(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ConnectorsConfigsResponse != nil {
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

**[*operations.ListConfigsAvailableConnectorsResponse](../../models/operations/listconfigsavailableconnectorsresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## InstallConnector

Install a connector by its name and config.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.InstallConnector(ctx, components.ConnectorAtlar, components.CreateConnectorConfigWise(
        components.WiseConfig{
            Name: "My Wise Account",
            APIKey: "XXX",
        },
    ))
    if err != nil {
        log.Fatal(err)
    }
    if res.ConnectorResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                | Type                                                                     | Required                                                                 | Description                                                              |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `ctx`                                                                    | [context.Context](https://pkg.go.dev/context#Context)                    | :heavy_check_mark:                                                       | The context to use for the request.                                      |
| `connector`                                                              | [components.Connector](../../models/components/connector.md)             | :heavy_check_mark:                                                       | The name of the connector.                                               |
| `connectorConfig`                                                        | [components.ConnectorConfig](../../models/components/connectorconfig.md) | :heavy_check_mark:                                                       | N/A                                                                      |
| `opts`                                                                   | [][operations.Option](../../models/operations/option.md)                 | :heavy_minus_sign:                                                       | The options for this request.                                            |

### Response

**[*operations.InstallConnectorResponse](../../models/operations/installconnectorresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ~~UninstallConnector~~

Uninstall a connector by its name.

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.UninstallConnector(ctx, components.ConnectorModulr)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                    | Type                                                         | Required                                                     | Description                                                  |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `ctx`                                                        | [context.Context](https://pkg.go.dev/context#Context)        | :heavy_check_mark:                                           | The context to use for the request.                          |
| `connector`                                                  | [components.Connector](../../models/components/connector.md) | :heavy_check_mark:                                           | The name of the connector.                                   |
| `opts`                                                       | [][operations.Option](../../models/operations/option.md)     | :heavy_minus_sign:                                           | The options for this request.                                |

### Response

**[*operations.UninstallConnectorResponse](../../models/operations/uninstallconnectorresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## UninstallConnectorV1

Uninstall a connector by its name.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.UninstallConnectorV1(ctx, components.ConnectorGeneric, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                    | Type                                                         | Required                                                     | Description                                                  | Example                                                      |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `ctx`                                                        | [context.Context](https://pkg.go.dev/context#Context)        | :heavy_check_mark:                                           | The context to use for the request.                          |                                                              |
| `connector`                                                  | [components.Connector](../../models/components/connector.md) | :heavy_check_mark:                                           | The name of the connector.                                   |                                                              |
| `connectorID`                                                | *string*                                                     | :heavy_check_mark:                                           | The connector ID.                                            | XXX                                                          |
| `opts`                                                       | [][operations.Option](../../models/operations/option.md)     | :heavy_minus_sign:                                           | The options for this request.                                |                                                              |

### Response

**[*operations.UninstallConnectorV1Response](../../models/operations/uninstallconnectorv1response.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ~~ReadConnectorConfig~~

Read connector config

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ReadConnectorConfig(ctx, components.ConnectorGeneric)
    if err != nil {
        log.Fatal(err)
    }
    if res.ConnectorConfigResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                    | Type                                                         | Required                                                     | Description                                                  |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `ctx`                                                        | [context.Context](https://pkg.go.dev/context#Context)        | :heavy_check_mark:                                           | The context to use for the request.                          |
| `connector`                                                  | [components.Connector](../../models/components/connector.md) | :heavy_check_mark:                                           | The name of the connector.                                   |
| `opts`                                                       | [][operations.Option](../../models/operations/option.md)     | :heavy_minus_sign:                                           | The options for this request.                                |

### Response

**[*operations.ReadConnectorConfigResponse](../../models/operations/readconnectorconfigresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## UpdateConnectorConfigV1

Update connector config

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.UpdateConnectorConfigV1(ctx, components.ConnectorAdyen, "XXX", components.CreateConnectorConfigStripe(
        components.StripeConfig{
            Name: "My Stripe Account",
            APIKey: "XXX",
        },
    ))
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                | Type                                                                     | Required                                                                 | Description                                                              | Example                                                                  |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `ctx`                                                                    | [context.Context](https://pkg.go.dev/context#Context)                    | :heavy_check_mark:                                                       | The context to use for the request.                                      |                                                                          |
| `connector`                                                              | [components.Connector](../../models/components/connector.md)             | :heavy_check_mark:                                                       | The name of the connector.                                               |                                                                          |
| `connectorID`                                                            | *string*                                                                 | :heavy_check_mark:                                                       | The connector ID.                                                        | XXX                                                                      |
| `connectorConfig`                                                        | [components.ConnectorConfig](../../models/components/connectorconfig.md) | :heavy_check_mark:                                                       | N/A                                                                      |                                                                          |
| `opts`                                                                   | [][operations.Option](../../models/operations/option.md)                 | :heavy_minus_sign:                                                       | The options for this request.                                            |                                                                          |

### Response

**[*operations.UpdateConnectorConfigV1Response](../../models/operations/updateconnectorconfigv1response.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ReadConnectorConfigV1

Read connector config

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ReadConnectorConfigV1(ctx, components.ConnectorCurrencyCloud, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res.ConnectorConfigResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                    | Type                                                         | Required                                                     | Description                                                  | Example                                                      |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `ctx`                                                        | [context.Context](https://pkg.go.dev/context#Context)        | :heavy_check_mark:                                           | The context to use for the request.                          |                                                              |
| `connector`                                                  | [components.Connector](../../models/components/connector.md) | :heavy_check_mark:                                           | The name of the connector.                                   |                                                              |
| `connectorID`                                                | *string*                                                     | :heavy_check_mark:                                           | The connector ID.                                            | XXX                                                          |
| `opts`                                                       | [][operations.Option](../../models/operations/option.md)     | :heavy_minus_sign:                                           | The options for this request.                                |                                                              |

### Response

**[*operations.ReadConnectorConfigV1Response](../../models/operations/readconnectorconfigv1response.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ~~ResetConnector~~

Reset a connector by its name.
It will remove the connector and ALL PAYMENTS generated with it.


> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ResetConnector(ctx, components.ConnectorAtlar)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                    | Type                                                         | Required                                                     | Description                                                  |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `ctx`                                                        | [context.Context](https://pkg.go.dev/context#Context)        | :heavy_check_mark:                                           | The context to use for the request.                          |
| `connector`                                                  | [components.Connector](../../models/components/connector.md) | :heavy_check_mark:                                           | The name of the connector.                                   |
| `opts`                                                       | [][operations.Option](../../models/operations/option.md)     | :heavy_minus_sign:                                           | The options for this request.                                |

### Response

**[*operations.ResetConnectorResponse](../../models/operations/resetconnectorresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ResetConnectorV1

Reset a connector by its name.
It will remove the connector and ALL PAYMENTS generated with it.


### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ResetConnectorV1(ctx, components.ConnectorGeneric, "XXX")
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                    | Type                                                         | Required                                                     | Description                                                  | Example                                                      |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `ctx`                                                        | [context.Context](https://pkg.go.dev/context#Context)        | :heavy_check_mark:                                           | The context to use for the request.                          |                                                              |
| `connector`                                                  | [components.Connector](../../models/components/connector.md) | :heavy_check_mark:                                           | The name of the connector.                                   |                                                              |
| `connectorID`                                                | *string*                                                     | :heavy_check_mark:                                           | The connector ID.                                            | XXX                                                          |
| `opts`                                                       | [][operations.Option](../../models/operations/option.md)     | :heavy_minus_sign:                                           | The options for this request.                                |                                                              |

### Response

**[*operations.ResetConnectorV1Response](../../models/operations/resetconnectorv1response.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ~~ListConnectorTasks~~

List all tasks associated with this connector.

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ListConnectorTasks(ctx, components.ConnectorModulr, nil, client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="))
    if err != nil {
        log.Fatal(err)
    }
    if res.TasksCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                                                | Type                                                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                                                 | Description                                                                                                                                                                                                                                              | Example                                                                                                                                                                                                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                          |
| `connector`                                                                                                                                                                                                                                              | [components.Connector](../../models/components/connector.md)                                                                                                                                                                                             | :heavy_check_mark:                                                                                                                                                                                                                                       | The name of the connector.                                                                                                                                                                                                                               |                                                                                                                                                                                                                                                          |
| `pageSize`                                                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The maximum number of results to return per page.<br/>                                                                                                                                                                                                   | 100                                                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Parameter used in pagination requests. Maximum page size is set to 15.<br/>Set to the value of next for the next page of results.<br/>Set to the value of previous for the previous page of results.<br/>No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                                                             |
| `opts`                                                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                                                            |                                                                                                                                                                                                                                                          |

### Response

**[*operations.ListConnectorTasksResponse](../../models/operations/listconnectortasksresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ListConnectorTasksV1

List all tasks associated with this connector.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ListConnectorTasksV1(ctx, components.ConnectorBankingCircle, "XXX", nil, client.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="))
    if err != nil {
        log.Fatal(err)
    }
    if res.TasksCursor != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                                                | Type                                                                                                                                                                                                                                                     | Required                                                                                                                                                                                                                                                 | Description                                                                                                                                                                                                                                              | Example                                                                                                                                                                                                                                                  |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                                                                                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                                                    | :heavy_check_mark:                                                                                                                                                                                                                                       | The context to use for the request.                                                                                                                                                                                                                      |                                                                                                                                                                                                                                                          |
| `connector`                                                                                                                                                                                                                                              | [components.Connector](../../models/components/connector.md)                                                                                                                                                                                             | :heavy_check_mark:                                                                                                                                                                                                                                       | The name of the connector.                                                                                                                                                                                                                               |                                                                                                                                                                                                                                                          |
| `connectorID`                                                                                                                                                                                                                                            | *string*                                                                                                                                                                                                                                                 | :heavy_check_mark:                                                                                                                                                                                                                                       | The connector ID.                                                                                                                                                                                                                                        | XXX                                                                                                                                                                                                                                                      |
| `pageSize`                                                                                                                                                                                                                                               | **int64*                                                                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The maximum number of results to return per page.<br/>                                                                                                                                                                                                   | 100                                                                                                                                                                                                                                                      |
| `cursor`                                                                                                                                                                                                                                                 | **string*                                                                                                                                                                                                                                                | :heavy_minus_sign:                                                                                                                                                                                                                                       | Parameter used in pagination requests. Maximum page size is set to 15.<br/>Set to the value of next for the next page of results.<br/>Set to the value of previous for the previous page of results.<br/>No other parameters can be set when this parameter is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                                                             |
| `opts`                                                                                                                                                                                                                                                   | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                                                 | :heavy_minus_sign:                                                                                                                                                                                                                                       | The options for this request.                                                                                                                                                                                                                            |                                                                                                                                                                                                                                                          |

### Response

**[*operations.ListConnectorTasksV1Response](../../models/operations/listconnectortasksv1response.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ~~GetConnectorTask~~

Get a specific task associated to the connector.

> :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.GetConnectorTask(ctx, components.ConnectorAdyen, "task1")
    if err != nil {
        log.Fatal(err)
    }
    if res.TaskResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                    | Type                                                         | Required                                                     | Description                                                  | Example                                                      |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `ctx`                                                        | [context.Context](https://pkg.go.dev/context#Context)        | :heavy_check_mark:                                           | The context to use for the request.                          |                                                              |
| `connector`                                                  | [components.Connector](../../models/components/connector.md) | :heavy_check_mark:                                           | The name of the connector.                                   |                                                              |
| `taskID`                                                     | *string*                                                     | :heavy_check_mark:                                           | The task ID.                                                 | task1                                                        |
| `opts`                                                       | [][operations.Option](../../models/operations/option.md)     | :heavy_minus_sign:                                           | The options for this request.                                |                                                              |

### Response

**[*operations.GetConnectorTaskResponse](../../models/operations/getconnectortaskresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## GetConnectorTaskV1

Get a specific task associated to the connector.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.GetConnectorTaskV1(ctx, components.ConnectorBankingCircle, "XXX", "task1")
    if err != nil {
        log.Fatal(err)
    }
    if res.TaskResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                    | Type                                                         | Required                                                     | Description                                                  | Example                                                      |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `ctx`                                                        | [context.Context](https://pkg.go.dev/context#Context)        | :heavy_check_mark:                                           | The context to use for the request.                          |                                                              |
| `connector`                                                  | [components.Connector](../../models/components/connector.md) | :heavy_check_mark:                                           | The name of the connector.                                   |                                                              |
| `connectorID`                                                | *string*                                                     | :heavy_check_mark:                                           | The connector ID.                                            | XXX                                                          |
| `taskID`                                                     | *string*                                                     | :heavy_check_mark:                                           | The task ID.                                                 | task1                                                        |
| `opts`                                                       | [][operations.Option](../../models/operations/option.md)     | :heavy_minus_sign:                                           | The options for this request.                                |                                                              |

### Response

**[*operations.GetConnectorTaskV1Response](../../models/operations/getconnectortaskv1response.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |

## ConnectorsTransfer

Execute a transfer between two accounts.

### Example Usage

```go
package main

import(
	"context"
	"github.com/formancehq/payments/pkg/client"
	"os"
	"github.com/formancehq/payments/pkg/client/models/components"
	"math/big"
	"log"
)

func main() {
    ctx := context.Background()

    s := client.New(
        "https://api.example.com",
        client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
    )

    res, err := s.Payments.V1.ConnectorsTransfer(ctx, components.ConnectorBankingCircle, components.TransferRequest{
        Amount: big.NewInt(100),
        Asset: "USD",
        Destination: "acct_1Gqj58KZcSIg2N2q",
        Source: client.String("acct_1Gqj58KZcSIg2N2q"),
    })
    if err != nil {
        log.Fatal(err)
    }
    if res.TransferResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                | Type                                                                     | Required                                                                 | Description                                                              |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ | ------------------------------------------------------------------------ |
| `ctx`                                                                    | [context.Context](https://pkg.go.dev/context#Context)                    | :heavy_check_mark:                                                       | The context to use for the request.                                      |
| `connector`                                                              | [components.Connector](../../models/components/connector.md)             | :heavy_check_mark:                                                       | The name of the connector.                                               |
| `transferRequest`                                                        | [components.TransferRequest](../../models/components/transferrequest.md) | :heavy_check_mark:                                                       | N/A                                                                      |
| `opts`                                                                   | [][operations.Option](../../models/operations/option.md)                 | :heavy_minus_sign:                                                       | The options for this request.                                            |

### Response

**[*operations.ConnectorsTransferResponse](../../models/operations/connectorstransferresponse.md), error**

### Errors

| Error Type         | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4XX, 5XX           | \*/\*              |