# openapi

Developer-friendly & type-safe Go SDK specifically catered to leverage *openapi* API.

<div align="left">
    <a href="https://www.speakeasy.com/?utm_source=openapi&utm_campaign=go"><img src="https://custom-icon-badges.demolab.com/badge/-Built%20By%20Speakeasy-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <a href="https://opensource.org/licenses/MIT">
        <img src="https://img.shields.io/badge/License-MIT-blue.svg" style="width: 100px; height: 28px;" />
    </a>
</div>


<br /><br />
> [!IMPORTANT]
> This SDK is not yet ready for production use. To complete setup please follow the steps outlined in your [workspace](https://app.speakeasy.com/org/formance/formance). Delete this section before > publishing to a package manager.

<!-- Start Summary [summary] -->
## Summary


<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [openapi](#openapi)
  * [SDK Installation](#sdk-installation)
  * [SDK Example Usage](#sdk-example-usage)
  * [Authentication](#authentication)
  * [Available Resources and Operations](#available-resources-and-operations)
  * [Retries](#retries)
  * [Error Handling](#error-handling)
  * [Custom HTTP Client](#custom-http-client)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start SDK Installation [installation] -->
## SDK Installation

To add the SDK as a dependency to your project:
```bash
go get github.com/formancehq/payments/pkg/client
```
<!-- End SDK Installation [installation] -->

<!-- Start SDK Example Usage [usage] -->
## SDK Example Usage

### Example

```go
package main

import (
	"context"
	"github.com/formancehq/payments/pkg/client"
	"log"
	"os"
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
<!-- End SDK Example Usage [usage] -->

<!-- Start Authentication [security] -->
## Authentication

### Per-Client Security Schemes

This SDK supports the following security scheme globally:

| Name            | Type   | Scheme  | Environment Variable     |
| --------------- | ------ | ------- | ------------------------ |
| `Authorization` | apiKey | API key | `FORMANCE_AUTHORIZATION` |

You can configure it using the `WithSecurity` option when initializing the SDK client instance. For example:
```go
package main

import (
	"context"
	"github.com/formancehq/payments/pkg/client"
	"log"
	"os"
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
<!-- End Authentication [security] -->

<!-- Start Available Resources and Operations [operations] -->
## Available Resources and Operations

<details open>
<summary>Available methods</summary>


### [Payments](docs/sdks/payments/README.md)


#### [Payments.V1](docs/sdks/v1/README.md)

* [GetServerInfo](docs/sdks/v1/README.md#getserverinfo) - Get server info
* [CreatePayment](docs/sdks/v1/README.md#createpayment) - Create a payment
* [ListPayments](docs/sdks/v1/README.md#listpayments) - List payments
* [GetPayment](docs/sdks/v1/README.md#getpayment) - Get a payment
* [UpdateMetadata](docs/sdks/v1/README.md#updatemetadata) - Update metadata
* [ListTransferInitiations](docs/sdks/v1/README.md#listtransferinitiations) - List Transfer Initiations
* [CreateTransferInitiation](docs/sdks/v1/README.md#createtransferinitiation) - Create a TransferInitiation
* [GetTransferInitiation](docs/sdks/v1/README.md#gettransferinitiation) - Get a transfer initiation
* [DeleteTransferInitiation](docs/sdks/v1/README.md#deletetransferinitiation) - Delete a transfer initiation
* [UpdateTransferInitiationStatus](docs/sdks/v1/README.md#updatetransferinitiationstatus) - Update the status of a transfer initiation
* [ReverseTransferInitiation](docs/sdks/v1/README.md#reversetransferinitiation) - Reverse a transfer initiation
* [RetryTransferInitiation](docs/sdks/v1/README.md#retrytransferinitiation) - Retry a failed transfer initiation
* [ListPools](docs/sdks/v1/README.md#listpools) - List Pools
* [CreatePool](docs/sdks/v1/README.md#createpool) - Create a Pool
* [GetPool](docs/sdks/v1/README.md#getpool) - Get a Pool
* [DeletePool](docs/sdks/v1/README.md#deletepool) - Delete a Pool
* [AddAccountToPool](docs/sdks/v1/README.md#addaccounttopool) - Add an account to a pool
* [RemoveAccountFromPool](docs/sdks/v1/README.md#removeaccountfrompool) - Remove an account from a pool
* [GetPoolBalances](docs/sdks/v1/README.md#getpoolbalances) - Get historical pool balances at a particular point in time
* [GetPoolBalancesLatest](docs/sdks/v1/README.md#getpoolbalanceslatest) - Get latest pool balances
* [CreateAccount](docs/sdks/v1/README.md#createaccount) - Create an account
* [ListAccounts](docs/sdks/v1/README.md#listaccounts) - List accounts
* [GetAccount](docs/sdks/v1/README.md#getaccount) - Get an account
* [GetAccountBalances](docs/sdks/v1/README.md#getaccountbalances) - Get account balances
* [CreateBankAccount](docs/sdks/v1/README.md#createbankaccount) - Create a BankAccount in Payments and on the PSP
* [ListBankAccounts](docs/sdks/v1/README.md#listbankaccounts) - List bank accounts created by user on Formance
* [GetBankAccount](docs/sdks/v1/README.md#getbankaccount) - Get a bank account created by user on Formance
* [ForwardBankAccount](docs/sdks/v1/README.md#forwardbankaccount) - Forward a bank account to a connector
* [UpdateBankAccountMetadata](docs/sdks/v1/README.md#updatebankaccountmetadata) - Update metadata of a bank account
* [ListAllConnectors](docs/sdks/v1/README.md#listallconnectors) - List all installed connectors
* [ListConfigsAvailableConnectors](docs/sdks/v1/README.md#listconfigsavailableconnectors) - List the configs of each available connector
* [InstallConnector](docs/sdks/v1/README.md#installconnector) - Install a connector
* [~~UninstallConnector~~](docs/sdks/v1/README.md#uninstallconnector) - Uninstall a connector :warning: **Deprecated**
* [UninstallConnectorV1](docs/sdks/v1/README.md#uninstallconnectorv1) - Uninstall a connector
* [~~ReadConnectorConfig~~](docs/sdks/v1/README.md#readconnectorconfig) - Read the config of a connector :warning: **Deprecated**
* [UpdateConnectorConfigV1](docs/sdks/v1/README.md#updateconnectorconfigv1) - Update the config of a connector
* [ReadConnectorConfigV1](docs/sdks/v1/README.md#readconnectorconfigv1) - Read the config of a connector
* [~~ResetConnector~~](docs/sdks/v1/README.md#resetconnector) - Reset a connector :warning: **Deprecated**
* [ResetConnectorV1](docs/sdks/v1/README.md#resetconnectorv1) - Reset a connector
* [~~ListConnectorTasks~~](docs/sdks/v1/README.md#listconnectortasks) - List tasks from a connector :warning: **Deprecated**
* [ListConnectorTasksV1](docs/sdks/v1/README.md#listconnectortasksv1) - List tasks from a connector
* [~~GetConnectorTask~~](docs/sdks/v1/README.md#getconnectortask) - Read a specific task of the connector :warning: **Deprecated**
* [GetConnectorTaskV1](docs/sdks/v1/README.md#getconnectortaskv1) - Read a specific task of the connector
* [ConnectorsTransfer](docs/sdks/v1/README.md#connectorstransfer) - Transfer funds between Connector accounts

#### [Payments.V3](docs/sdks/v3/README.md)

* [CreateAccount](docs/sdks/v3/README.md#createaccount) - Create a formance account object. This object will not be forwarded to the connector. It is only used for internal purposes.

* [ListAccounts](docs/sdks/v3/README.md#listaccounts) - List all accounts
* [GetAccount](docs/sdks/v3/README.md#getaccount) - Get an account by ID
* [GetAccountBalances](docs/sdks/v3/README.md#getaccountbalances) - Get account balances
* [CreateBankAccount](docs/sdks/v3/README.md#createbankaccount) - Create a formance bank account object. This object will not be forwarded to the connector until you called the forwardBankAccount method.

* [ListBankAccounts](docs/sdks/v3/README.md#listbankaccounts) - List all bank accounts
* [GetBankAccount](docs/sdks/v3/README.md#getbankaccount) - Get a Bank Account by ID
* [UpdateBankAccountMetadata](docs/sdks/v3/README.md#updatebankaccountmetadata) - Update a bank account's metadata
* [ForwardBankAccount](docs/sdks/v3/README.md#forwardbankaccount) - Forward a Bank Account to a PSP for creation
* [ListConnectors](docs/sdks/v3/README.md#listconnectors) - List all connectors
* [InstallConnector](docs/sdks/v3/README.md#installconnector) - Install a connector
* [ListConnectorConfigs](docs/sdks/v3/README.md#listconnectorconfigs) - List all connector configurations
* [UninstallConnector](docs/sdks/v3/README.md#uninstallconnector) - Uninstall a connector
* [GetConnectorConfig](docs/sdks/v3/README.md#getconnectorconfig) - Get a connector configuration by ID
* [V3UpdateConnectorConfig](docs/sdks/v3/README.md#v3updateconnectorconfig) - Update the config of a connector
* [ResetConnector](docs/sdks/v3/README.md#resetconnector) - Reset a connector. Be aware that this will delete all data and stop all existing tasks like payment initiations and bank account creations.
* [ListConnectorSchedules](docs/sdks/v3/README.md#listconnectorschedules) - List all connector schedules
* [GetConnectorSchedule](docs/sdks/v3/README.md#getconnectorschedule) - Get a connector schedule by ID
* [ListConnectorScheduleInstances](docs/sdks/v3/README.md#listconnectorscheduleinstances) - List all connector schedule instances
* [CreatePayment](docs/sdks/v3/README.md#createpayment) - Create a formance payment object. This object will not be forwarded to the connector. It is only used for internal purposes.

* [ListPayments](docs/sdks/v3/README.md#listpayments) - List all payments
* [GetPayment](docs/sdks/v3/README.md#getpayment) - Get a payment by ID
* [UpdatePaymentMetadata](docs/sdks/v3/README.md#updatepaymentmetadata) - Update a payment's metadata
* [InitiatePayment](docs/sdks/v3/README.md#initiatepayment) - Initiate a payment
* [ListPaymentInitiations](docs/sdks/v3/README.md#listpaymentinitiations) - List all payment initiations
* [DeletePaymentInitiation](docs/sdks/v3/README.md#deletepaymentinitiation) - Delete a payment initiation by ID
* [GetPaymentInitiation](docs/sdks/v3/README.md#getpaymentinitiation) - Get a payment initiation by ID
* [RetryPaymentInitiation](docs/sdks/v3/README.md#retrypaymentinitiation) - Retry a payment initiation
* [ApprovePaymentInitiation](docs/sdks/v3/README.md#approvepaymentinitiation) - Approve a payment initiation
* [RejectPaymentInitiation](docs/sdks/v3/README.md#rejectpaymentinitiation) - Reject a payment initiation
* [ReversePaymentInitiation](docs/sdks/v3/README.md#reversepaymentinitiation) - Reverse a payment initiation
* [ListPaymentInitiationAdjustments](docs/sdks/v3/README.md#listpaymentinitiationadjustments) - List all payment initiation adjustments
* [ListPaymentInitiationRelatedPayments](docs/sdks/v3/README.md#listpaymentinitiationrelatedpayments) - List all payments related to a payment initiation
* [CreatePaymentServiceUser](docs/sdks/v3/README.md#createpaymentserviceuser) - Create a formance payment service user object
* [ListPaymentServiceUsers](docs/sdks/v3/README.md#listpaymentserviceusers) - List all payment service users
* [GetPaymentServiceUser](docs/sdks/v3/README.md#getpaymentserviceuser) - Get a payment service user by ID
* [DeletePaymentServiceUser](docs/sdks/v3/README.md#deletepaymentserviceuser) - Delete a payment service user by ID
* [ListPaymentServiceUserConnections](docs/sdks/v3/README.md#listpaymentserviceuserconnections) - List all connections for a payment service user
* [DeletePaymentServiceUserConnector](docs/sdks/v3/README.md#deletepaymentserviceuserconnector) - Delete a payment service user on a connector
* [ForwardPaymentServiceUserToBankBridge](docs/sdks/v3/README.md#forwardpaymentserviceusertobankbridge) - Forward a payment service user to a connector
* [CreateLinkForPaymentServiceUser](docs/sdks/v3/README.md#createlinkforpaymentserviceuser) - Create a link for a payment service user on a connector
* [ListPaymentServiceUserConnectionsFromConnectorID](docs/sdks/v3/README.md#listpaymentserviceuserconnectionsfromconnectorid) - List all connections for a payment service user on a connector
* [ListPaymentServiceUserLinkAttemptsFromConnectorID](docs/sdks/v3/README.md#listpaymentserviceuserlinkattemptsfromconnectorid) - List all link attempts for a payment service user on a connector
* [GetPaymentServiceUserLinkAttemptFromConnectorID](docs/sdks/v3/README.md#getpaymentserviceuserlinkattemptfromconnectorid) - Get a link attempt for a payment service user on a connector
* [DeletePaymentServiceUserConnectionFromConnectorID](docs/sdks/v3/README.md#deletepaymentserviceuserconnectionfromconnectorid) - Delete a connection for a payment service user on a connector
* [UpdateLinkForPaymentServiceUserOnConnector](docs/sdks/v3/README.md#updatelinkforpaymentserviceuseronconnector) - Update a link for a payment service user on a connector
* [AddBankAccountToPaymentServiceUser](docs/sdks/v3/README.md#addbankaccounttopaymentserviceuser) - Add a bank account to a payment service user
* [ForwardPaymentServiceUserBankAccount](docs/sdks/v3/README.md#forwardpaymentserviceuserbankaccount) - Forward a payment service user's bank account to a connector
* [CreatePool](docs/sdks/v3/README.md#createpool) - Create a formance pool object
* [ListPools](docs/sdks/v3/README.md#listpools) - List all pools
* [GetPool](docs/sdks/v3/README.md#getpool) - Get a pool by ID
* [DeletePool](docs/sdks/v3/README.md#deletepool) - Delete a pool by ID
* [GetPoolBalances](docs/sdks/v3/README.md#getpoolbalances) - Get historical pool balances from a particular point in time
* [GetPoolBalancesLatest](docs/sdks/v3/README.md#getpoolbalanceslatest) - Get latest pool balances
* [AddAccountToPool](docs/sdks/v3/README.md#addaccounttopool) - Add an account to a pool
* [RemoveAccountFromPool](docs/sdks/v3/README.md#removeaccountfrompool) - Remove an account from a pool
* [GetTask](docs/sdks/v3/README.md#gettask) - Get a task and its result by ID

</details>
<!-- End Available Resources and Operations [operations] -->

<!-- Start Retries [retries] -->
## Retries

Some of the endpoints in this SDK support retries. If you use the SDK without any configuration, it will fall back to the default retry strategy provided by the API. However, the default retry strategy can be overridden on a per-operation basis, or across the entire SDK.

To change the default retry strategy for a single API call, simply provide a `retry.Config` object to the call by using the `WithRetries` option:
```go
package main

import (
	"context"
	"github.com/formancehq/payments/pkg/client"
	"github.com/formancehq/payments/pkg/client/retry"
	"log"
	"models/operations"
	"os"
)

func main() {
	ctx := context.Background()

	s := client.New(
		"https://api.example.com",
		client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
	)

	res, err := s.Payments.V1.GetServerInfo(ctx, operations.WithRetries(
		retry.Config{
			Strategy: "backoff",
			Backoff: &retry.BackoffStrategy{
				InitialInterval: 1,
				MaxInterval:     50,
				Exponent:        1.1,
				MaxElapsedTime:  100,
			},
			RetryConnectionErrors: false,
		}))
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```

If you'd like to override the default retry strategy for all operations that support retries, you can use the `WithRetryConfig` option at SDK initialization:
```go
package main

import (
	"context"
	"github.com/formancehq/payments/pkg/client"
	"github.com/formancehq/payments/pkg/client/retry"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := client.New(
		"https://api.example.com",
		client.WithRetryConfig(
			retry.Config{
				Strategy: "backoff",
				Backoff: &retry.BackoffStrategy{
					InitialInterval: 1,
					MaxInterval:     50,
					Exponent:        1.1,
					MaxElapsedTime:  100,
				},
				RetryConnectionErrors: false,
			}),
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
<!-- End Retries [retries] -->

<!-- Start Error Handling [errors] -->
## Error Handling

Handling errors in this SDK should largely match your expectations. All operations return a response object or an error, they will never return both.

By Default, an API error will return `sdkerrors.SDKError`. When custom error responses are specified for an operation, the SDK may also return their associated error. You can refer to respective *Errors* tables in SDK docs for more details on possible error types for each operation.

For example, the `GetServerInfo` function may return the following errors:

| Error Type         | Status Code | Content Type |
| ------------------ | ----------- | ------------ |
| sdkerrors.SDKError | 4XX, 5XX    | \*/\*        |

### Example

```go
package main

import (
	"context"
	"errors"
	"github.com/formancehq/payments/pkg/client"
	"github.com/formancehq/payments/pkg/client/models/sdkerrors"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := client.New(
		"https://api.example.com",
		client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
	)

	res, err := s.Payments.V1.GetServerInfo(ctx)
	if err != nil {

		var e *sdkerrors.SDKError
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}
	}
}

```
<!-- End Error Handling [errors] -->

<!-- Start Custom HTTP Client [http-client] -->
## Custom HTTP Client

The Go SDK makes API calls that wrap an internal HTTP client. The requirements for the HTTP client are very simple. It must match this interface:

```go
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
```

The built-in `net/http` client satisfies this interface and a default client based on the built-in is provided by default. To replace this default with a client of your own, you can implement this interface yourself or provide your own client configured as desired. Here's a simple example, which adds a client with a 30 second timeout.

```go
import (
	"net/http"
	"time"
	"github.com/myorg/your-go-sdk"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	sdkClient  = sdk.New(sdk.WithClient(httpClient))
)
```

This can be a convenient way to configure timeouts, cookies, proxies, custom headers, and other low-level configuration.
<!-- End Custom HTTP Client [http-client] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This SDK is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

While we value open-source contributions to this SDK, this library is generated programmatically. Any manual changes added to internal files will be overwritten on the next generation. 
We look forward to hearing your feedback. Feel free to open a PR or an issue with a proof of concept and we'll do our best to include it in a future release. 

### SDK Created by [Speakeasy](https://www.speakeasy.com/?utm_source=openapi&utm_campaign=go)
