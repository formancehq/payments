# How to build a connector

You can build a connector for a new Payments Service Provider (PSP) or integrate new payment methods into an existing connector by using the Plugin interface.

This guide demonstrates the process of building a basic connector for a hypothetical PSP called `DummyPay` using the Plugin interface.

## Table of Contents

- [How to build a connector](#how-to-build-a-connector)
  - [Table of Contents](#table-of-contents)
  - [Understanding the Plugin interface](#understanding-the-plugin-interface)
  - [Building a connector](#building-a-connector)
    - [Set up the project](#set-up-the-project)
    - [Define connector capabilities](#define-connector-capabilities)
    - [Define connector configuration](#define-connector-configuration)
    - [Define the Connector Struct](#define-the-connector-struct)
    - [Implement Plugin interface methods](#implement-plugin-interface-methods)
    - [Implement installation logic](#implement-installation-logic)
    - [Implement uninstallation logic](#implement-uninstallation-logic)
    - [Connect to the PSP and fetch data](#connect-to-the-psp-and-fetch-data)
    - [Implement state management](#implement-state-management)
    - [Set up state persistence](#set-up-state-persistence)
    - [Handle child tasks](#handle-child-tasks)
  - [Launching a new connector](#launching-a-new-connector)
    - [Additional Connector Configuration](#additional-connector-configuration)
  - [Testing a connector](#testing-a-connector)
    - [Installing](#installing)
    - [Uninstalling](#uninstalling)
    - [Data Transformation](#data-transformation)
      - [Account/External Account Transformation](#accountexternal-account-transformation)
      - [Balances Transformation](#balances-transformation)
      - [Payments Transformation](#payments-transformation)
      - [Others Transformation](#others-transformation)
    - [Fetching Data via Polling](#fetching-data-via-polling)
      - [Errors](#errors)
      - [Polling](#polling)
      - [Transformation](#transformation)
    - [Fetching Data via Webhooks](#fetching-data-via-webhooks)
      - [Webhooks Creation](#webhooks-creation)
    - [Creating a Bank Account](#creating-a-bank-account)
      - [Errors](#errors-1)
      - [Bank Account Creation](#bank-account-creation)
    - [Creating a Transfer/Payout](#creating-a-transferpayout)
      - [Errors](#errors-2)
      - [Transfer/Payout Creation](#transferpayout-creation)
  - [Special Implementation details](#special-implementation-details)
    - [Metadata](#metadata)
      - [Use namespaces for metadata](#use-namespaces-for-metadata)
      - [Extracting Metadata values from Metadata](#extracting-metadata-values-from-metadata)
      - [Save extra data in Metadata](#save-extra-data-in-metadata)
    - [Asset and Amount Handling](#asset-and-amount-handling)
      - [Asset Format](#asset-format)
      - [Amount Handling](#amount-handling)
      - [Asset and Amount handling examples](#asset-and-amount-handling-examples)
    - [Important Considerations](#important-considerations)
    - [Setting up Pre-commit Checks](#setting-up-pre-commit-checks)
    - [Troubleshooting](#troubleshooting)
    - [Review Checklist](#review-checklist)

## Understanding the Plugin interface

The [Plugin interface](https://github.com/formancehq/payments/blob/main/internal/models/plugin.go#L14-L36) defines the required methods for all connectors and serves as the blueprint for their implementation. Since it's written in Go, Go's type system requires all methods to be implemented to satisfy the interface, even if some are not used by the connector.

The `Install()` and `Uninstall()` methods are essential for activating, deactivating, and managing data synchronization with a PSP, and must always be implemented. Other methods—such as those for data polling, transfer initiation, and webhook management—are optional in terms of functionality but must still be implemented to satisfy the interface. If these methods are not supported by the PSP, they can return an UNIMPLEMENTED error.

Here is the complete interface definition for your reference:

```go
type Plugin interface {
	Name() string

	Install(context.Context, InstallRequest) (InstallResponse, error)
	Uninstall(context.Context, UninstallRequest) (UninstallResponse, error)

	FetchNextAccounts(context.Context, FetchNextAccountsRequest) (FetchNextAccountsResponse, error)
	FetchNextPayments(context.Context, FetchNextPaymentsRequest) (FetchNextPaymentsResponse, error)
	FetchNextBalances(context.Context, FetchNextBalancesRequest) (FetchNextBalancesResponse, error)
	FetchNextExternalAccounts(context.Context, FetchNextExternalAccountsRequest) (FetchNextExternalAccountsResponse, error)
	FetchNextOthers(context.Context, FetchNextOthersRequest) (FetchNextOthersResponse, error)

	CreateBankAccount(context.Context, CreateBankAccountRequest) (CreateBankAccountResponse, error)
	CreateTransfer(context.Context, CreateTransferRequest) (CreateTransferResponse, error)
	ReverseTransfer(context.Context, ReverseTransferRequest) (ReverseTransferResponse, error)
	PollTransferStatus(context.Context, PollTransferStatusRequest) (PollTransferStatusResponse, error)
	CreatePayout(context.Context, CreatePayoutRequest) (CreatePayoutResponse, error)
	ReversePayout(context.Context, ReversePayoutRequest) (ReversePayoutResponse, error)
	PollPayoutStatus(context.Context, PollPayoutStatusRequest) (PollPayoutStatusResponse, error)

	CreateWebhooks(context.Context, CreateWebhooksRequest) (CreateWebhooksResponse, error)
	TranslateWebhook(context.Context, TranslateWebhookRequest) (TranslateWebhookResponse, error)
}
```

| Method                         | Description                                                                                                                                                                                                                         |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Name()                         | Returns the name of the plugin, which is used to register the connector in the connector registry                                                                                                                                   |
| Install(...)                   | Activates the connector, sets up the required configuration and start Data synchronization with the PSP                                                                                                                             |
| Uninstall(...)                 | Deactivates the connector, cleans up any resources created during installation, such as webhooks or cache data                                                                                                                      |
| FetchNextAccounts(...)         | Retrieves the next set of account data from the PSP for synchronization                                                                                                                                                             |
| FetchNextPayments(...)         | Retrieves the next set of payment data from the PSP for synchronization                                                                                                                                                             |
| FetchNextBalances(...)         | Retrieves the next set of balance data (e.g., account balances) from the PSP for synchronization                                                                                                                                    |
| FetchNextExternalAccounts(...) | Retrieves external accounts (e.g., linked bank or card accounts) from the PSP for synchronization                                                                                                                                   |
| FetchNextOthers(...)           | Fetches any additional or custom data from the PSP that doesn't fall into the predefined categories                                                                                                                                 |
| CreateBankAccount(...)         | Creates a new bank account or linked financial account in the PSP                                                                                                                                                                   |
| CreateTransfer(...)            | Initiates a transfer of funds between accounts within the PSP or externally                                                                                                                                                         |
| ReverseTransfer(...)           | Reverses a previously initiated processed transfer                                                                                                                                                                                  |
| PollTransferStatus(...)        | Polls the status of a previously initiated transfer to determine whether it was successful, pending, or failed. Useful for PSPs whose APIs don't provide synchronous feedback about whether or not a transfer was successful or not |
| CreatePayout(...)              | Initiates a payout from a PSP account to an external account (e.g., a bank or another PSP)                                                                                                                                          |
| ReversePayout(...)             | Reverses a previously initiated payout                                                                                                                                                                                              |
| PollPayoutStatus(...)          | Polls the status of a previously initiated payout to determine whether it was successful, pending, or failed. Useful for PSPs whose APIs don't provide synchronous feedback about whether or not a payout was successful or not     |
| CreateWebhooks(...)            | Sets up webhooks in the PSP to notify the Payments Service of events (e.g., payment updates)                                                                                                                                        |
| TranslateWebhook(...)          | Converts incoming webhook events from the PSP into a format that the Payments Service understands                                                                                                                                   |

## Building a connector

In this tutorial, we'll build a connector for a hypothetical PSP, DummyPay, to read payment files from a directory containing fictional payments to be processed. We'll define the connector capabilities and configuration, and use the Plugin interface to implement installation and data-fetching logic for the connector. A fully implemented version of the DummyPay connector is available in our integration testing environment. You can check out the code on GitHub as you follow along.

### Set up the project

To set up the project:

1. Clone the Payments repository:

```console
$ git clone git@github.com:formancehq/payments.git
$ cd payments
```

2. Use the connector-template tool to create the connector directory and generate all files needed for the connector to work:

```console
$ cd tools/connector-template
$ go run ./ --connector-dir-path ../../internal/connectors/plugins/public/ --connector-name dummypay2
```

### Define connector capabilities

We want the DummyPay connector to be capable of fetching various data types from the DummyPay directory.

Open the `capabilities.go` file in the `dummypay2` directory to outline the connector capabilities:

```go
package dummypay2

import "github.com/formancehq/payments/internal/models"

var capabilities = []models.Capability{
	models.CAPABILITY_FETCH_ACCOUNTS,
	models.CAPABILITY_FETCH_BALANCES,
	models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS,
	models.CAPABILITY_FETCH_PAYMENTS,

	models.CAPABILITY_CREATE_TRANSFER,
	models.CAPABILITY_CREATE_PAYOUT,
}
```

You can find below the list of capabilities supported:

| Capability                                 | Description                                                                                                                                                                                                                                                           |
| ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CAPABILITY_FETCH_ACCOUNTS                  | Connector can fetch accounts from the PSP                                                                                                                                                                                                                             |
| CAPABILITY_FETCH_BALANCES                  | Connector can fetch account balances from the PSP                                                                                                                                                                                                                     |
| CAPABILITY_FETCH_EXTERNAL_ACCOUNTS         | Connector can fetch external accounts from the PSP                                                                                                                                                                                                                    |
| CAPABILITY_FETCH_PAYMENTS                  | Connector can fetch payments from the PSP                                                                                                                                                                                                                             |
| CAPABILITY_FETCH_OTHERS                    | Connector is going to fetch other object first from the PSP in order to be able to fetch accounts, balances, external accounts or payments from these other objects                                                                                                   |
| CAPABILITY_CREATE_WEBHOOKS                 | Connector can create webhooks on the PSP                                                                                                                                                                                                                              |
| CAPABILITY_TRANSLATE_WEBHOOKS              | Connector can handle webhooks received from the PSP                                                                                                                                                                                                                   |
| CAPABILITY_CREATE_BANK_ACCOUNT             | Connector can create bank accounts on the PSP                                                                                                                                                                                                                         |
| CAPABILITY_CREATE_TRANSFER                 | Connector can create transfer between accounts on the PSP                                                                                                                                                                                                             |
| CAPABILITY_CREATE_PAYOUT                   | Connector can create payout between accounts and external account on the PSP                                                                                                                                                                                          |
| CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION | Connector is allowed to have Formance account created directly from Formance API without being forwarded to the PSP. (This can be useful if the PSP does not provide a way to fetch the history of accounts, the user can directly create them via the Formance API)  |
| CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION | Connector is allowed to have Formance payments created directly from Formance API without being forwarded to the PSP. (This can be useful if the PSP does not provide a way to fetch the history of payments, the user can directly create them via the Formance API) |

### Define connector configuration

Open the config.go file in the dummypay2 directory to define the connector configuration:

```go
package dummypay2

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	Directory string `json:"directory" validate:"required,dirpath"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
```

The `Config` struct contains any data needed to properly connect to and authenticate to the PSP. In a real-world scenario, this is likely going to be data such as APIKeys, Authorization Endpoint URLs, Client IDs, and anything needed by a PSP to identify the user communicating with their APIs.

Since our DummyPay PSP uses the local filesystem, the only information we require in the config is the directory where the files to poll will be stored.

### Define the Connector Struct

Open the file called plugin.go to define the connector struct:

```go
package dummypay2

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

func init() {
	registry.RegisterPlugin("dummypay", func(name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	name string
	logger logging.Logger
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	_, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		name: name,
		logger: logger,
	}, nil
}
```

**Explanation:**

- The `Plugin` struct represents the connector itself.
- The `init()` ensures the connector is registered with the Connectivity Service, allowing it to be recognized along with its capabilities. Without this, the plugin won't be loaded into the registry.
- The `New()` function initializes the plugin and validates the configuration before returning it.

### Implement Plugin interface methods

In the `plugin.go` file, add the methods required for the [Plugin interface](https://www.notion.so/Build-a-Connector-158066a5ef3280538d23c2fa239fa78a?pvs=21).

For now, return ErrNotImplemented for all methods except Name:

```go
func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	return models.FetchNextAccountsResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	return models.FetchNextBalancesResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	return models.FetchNextExternalAccountsResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	return models.FetchNextPaymentsResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	return models.CreateBankAccountResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	return models.CreateTransferResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	return models.ReverseTransferResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	return models.CreatePayoutResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	return models.ReversePayoutResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	return models.CreateWebhooksResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	return models.PollPayoutStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	return models.TranslateWebhookResponse{}, plugins.ErrNotImplemented
}

var _ models.Plugin = &Plugin{}
```

### Implement installation logic

In this step, we'll define what our connector will do when it is installed. Connectors are typically designed to continuously synchronize data—such as accounts and payments—from the PSP to the local Formance instance in the background. To achieve this, we need to configure the types of data that will be synchronized.

Within the Payments Service, this is structured as a _Workflow Task Tree_. This tree can include independent parent nodes that execute in parallel, as well as child nodes that are triggered only by the completion of a parent node. To set this up, we'll open the file called `workflow.go` in the `dummypay2` directory and define the task tree within it.

```go
func workflow() models.ConnectorTasksTree {
	return []models.ConnectorTaskTree{
		{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_BALANCES,
					Name:         "fetch_balances",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			},
		},
		{
			TaskType:     models.TASK_FETCH_PAYMENTS,
			Name:         "fetch_payments",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}
}
```

Here, we have defined a task tree containing two parent nodes. This configuration means that when this workflow is triggered, it will begin fetching accounts and fetching payments. Additionally, the `TASK_FETCH_ACCOUNTS` task includes a child node in its `NextTasks` list, which will perform the `TASK_FETCH_BALANCES` operation for each account retrieved in the parent node.

Now that the workflow has been established, we need to go back to our `Plugin` struct and implement the installation logic. Replace the `plugins.ErrNotImplemented` return value with a valid `models.InstallResponse`, which will include the workflow we've just defined.

```go
func (p *Plugin) Install(_ context.Context, _ models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}
```

### Implement uninstallation logic

To add the ability to uninstall the connector, we can return an empty model.UninstallResponse:

```go
func (p *Plugin) Uninstall(_ context.Context, _ models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}
```

### Connect to the PSP and fetch data

A typical PSP API allows authenticated users to connect to it via HTTPS to read or perform operations on payment data. Each PSP has unique authentication methods and endpoints, making this component the most variable among connectors.

To connect to a PSP and fetch data, the connector needs to translate the PSP's API responses into standardized data structures that the Plugin interface can use, such as Accounts, ExternalAccounts, Balances, and Payments.

First, let's define what an account will look like for our example. Suppose DummyPay provides an accounts list as follows:

```json
[
  {
    "id": "87afd68f-4441-4ec0-a5d4-342d241bbeca",
    "name": "dummy-account-0",
    "opening_date": "2024-12-02T17:34:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "182345d5-0546-4ae1-b470-2c6f015f7d6b",
    "name": "dummy-account-1",
    "opening_date": "2024-12-02T17:33:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "74f3b555-2fc0-4bd7-b171-1965bb44fa53",
    "name": "dummy-account-2",
    "opening_date": "2024-12-02T17:32:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "73d31ca4-78dd-4e86-aa3b-a971ff59c93e",
    "name": "dummy-account-3",
    "opening_date": "2024-12-02T17:31:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "367cb45d-440e-4d04-b5d8-d8603955358c",
    "name": "dummy-account-4",
    "opening_date": "2024-12-02T17:30:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "93068adb-aa41-4a6d-9f94-7f5ec315b57b",
    "name": "dummy-account-5",
    "opening_date": "2024-12-02T17:29:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "ad7baa78-dcf0-42b7-af2f-4db4a0f36208",
    "name": "dummy-account-6",
    "opening_date": "2024-12-02T17:28:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "5247256e-a1b8-497b-8d01-9f93a7557ef6",
    "name": "dummy-account-7",
    "opening_date": "2024-12-02T17:27:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "55bb43d1-bf5a-434d-95d9-222e75c189cc",
    "name": "dummy-account-8",
    "opening_date": "2024-12-02T17:26:07+01:00",
    "currency": "EUR"
  },
  {
    "id": "440cc4b7-138f-4c6a-8120-1190e6e02714",
    "name": "dummy-account-9",
    "opening_date": "2024-12-02T17:25:07+01:00",
    "currency": "EUR"
  }
]
```

Let's create a DummyPay client which will read and unmarshal the json file, and convert it to a data structure that the payments service knows how to use.

A `client` package should have been generated: `internal/connectors/plugins/public/dummypay2/client`

Within the `client` package, open the file called `account.go` and define the struct that matches the structure of the JSON data:

```go
package client

import "time"

type Account struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OpeningDate time.Time `json:"opening_date"`
	Currency    string    `json:"currency"`
}
```

Open the file called client.go and define the client interface and constructor:

```go
package client

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

type Client interface {
	FetchAccounts(ctx context.Context, startToken int, pageSize int) ([]models.PSPAccount, int, error)
}

type client struct {
	directory string
}

func New(dir string) Client {
	return &client{
		directory: dir,
	}
}
```

Implement the `FetchAccounts` function to read and convert the data:

```go
func (c *client) FetchAccounts(ctx context.Context, startToken int, pageSize int) ([]models.PSPAccount, int, error) {
	b, err := c.readFile("accounts.json")
	if err != nil {
		return []models.PSPAccount{}, 0, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	accounts := make([]Account, 0)
	err = json.Unmarshal(b, &accounts)
	if err != nil {
		return []models.PSPAccount{}, 0, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	next := -1
	pspAccounts := make([]models.PSPAccount, 0, pageSize)
	for i := startToken; i < len(accounts); i++ {
		if len(pspAccounts) >= pageSize {
			if len(accounts)-startToken > len(pspAccounts) {
				next = i
			}
			break
		}

		account := accounts[i]
		pspAccounts = append(pspAccounts, models.PSPAccount{
			Reference:    account.ID,
			CreatedAt:    account.OpeningDate,
			Name:         &account.Name,
			DefaultAsset: &account.Currency,
		})
	}
	return pspAccounts, next, nil
}

func (c *client) readFile(filename string) (b []byte, err error) {
	filePath := path.Join(c.directory, filename)
	file, err := os.Open(filePath)
	if err != nil {
		return b, fmt.Errorf("failed to create %q: %w", filePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return b, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	buf := make([]byte, fileInfo.Size())
	_, err = file.Read(buf)
	if err != nil {
		return b, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}
	return buf, nil
}
```

This ensures your implementation fetches data and paginates it as required.

Now that we've defined a way to ingest the data, let's integrate this into `plugin.go` and connect it with the `FetchNextAccounts` function.

```go
func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	accounts, next, err := p.client.FetchAccounts(ctx, 0, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch accounts from client: %w", err)
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		HasMore:  next > 0,
	}, nil
}
```

### Implement state management

To make our connector more robust, we need to handle cases where req.PageSize is smaller than the total number of accounts. This requires including a mechanism in FetchNextAccountsResponse to indicate if more data is available. By setting HasMore to true, we signal the underlying connector engine to schedule a follow-up fetch. We also need to save the current position so that fetching can resume seamlessly.

### Set up state persistence

On reviewing the `models.FetchNextAccountsRequest` and `models.FetchNextAccountsResponse` structures, you'll notice they both include `State` and `NewState` fields. These fields enable state persistence across independent fetch requests. The `json.RawMessage` format is utilized so that plugin authors can decide how to store the state data, which may differ based on the paging mechanism of each PSP. In most cases, this may be as simple as storing a page number.

In the `dummypay` client, we return an integer representing the position in the file to continue reading from. Let's define a struct for this purpose:

```go
type accountsState struct {
	NextToken int `json:"nextToken"`
}
```

Now, update the `FetchNextAccounts` function to handle state persistence. We will unmarshal the existing state, pass the token to the client's `FetchAccounts` method, and save the next token into `newState`. This way, the `NewState` field in `FetchNextAccountsResponse` will tell the Connectivity service where to resume on the next call.

```go
func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	accounts, next, err := p.client.FetchAccounts(ctx, oldState.NextToken, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch accounts from client: %w", err)
	}

	newState := accountsState{
		NextToken: next,
	}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  next > 0,
	}, nil
}
```

### Handle child tasks

The process for other `Fetch*` tasks is similar. Note that child tasks can access information about the parent task that triggered them.

**Example: Fetching Account Balances**

Consider adding a `FetchBalance` method to the client to fetch the balance of a specific account:

```go
type Client interface {
	FetchAccounts(ctx context.Context, startToken int, pageSize int) ([]models.PSPAccount, int, error)
	FetchBalance(ctx context.Context, accountID string) (*models.PSPBalance, error)
}
```

Suppose `dummypay` contains a `balances.json` file like this:

```json
[
  {
    "account_id": "ab59834d-e94a-4547-b908-4be3eed9ed68",
    "amount_in_minors": 23,
    "currency": "EUR"
  },
  {
    "account_id": "f0946302-05c7-4775-84ed-81d19b39cb26",
    "amount_in_minors": 123,
    "currency": "EUR"
  },
  {
    "account_id": "3b1af736-9ca7-4a24-98d5-c626405f5fef",
    "amount_in_minors": 223,
    "currency": "EUR"
  },
  {
    "account_id": "ea651da2-0810-4422-8a1f-159b6e15bcee",
    "amount_in_minors": 323,
    "currency": "EUR"
  },
  {
    "account_id": "2f6d8a3c-f414-45a2-9c0a-2065c1ade017",
    "amount_in_minors": 423,
    "currency": "EUR"
  },
  {
    "account_id": "527d3a1f-ed48-4425-97be-3d5e97a9beb0",
    "amount_in_minors": 523,
    "currency": "EUR"
  },
  {
    "account_id": "152ed96b-8aba-4431-82ed-1478caad8eba",
    "amount_in_minors": 623,
    "currency": "EUR"
  },
  {
    "account_id": "cd680304-3769-4750-a4b7-017c4fc20803",
    "amount_in_minors": 723,
    "currency": "EUR"
  },
  {
    "account_id": "6cadea83-bac2-4ced-adcd-8758f4712aeb",
    "amount_in_minors": 823,
    "currency": "EUR"
  },
  {
    "account_id": "aac11dd3-360b-424b-a9b8-3f85337793e0",
    "amount_in_minors": 923,
    "currency": "EUR"
  }
]
```

Implement the FetchBalance function as follows:

```go
func (c *client) FetchBalance(ctx context.Context, accountID string) (*models.PSPBalance, error) {
	b, err := c.readFile("balances.json")
	if err != nil {
		return &models.PSPBalance{}, fmt.Errorf("failed to fetch balances: %w", err)
	}

	balances := make([]Balance, 0)
	err = json.Unmarshal(b, &balances)
	if err != nil {
		return &models.PSPBalance{}, fmt.Errorf("failed to unmarshal balances: %w", err)
	}

	for _, balance := range balances {
		if balance.AccountID != accountID {
			continue
		}
		return &models.PSPBalance{
			AccountReference: balance.AccountID,
			CreatedAt:        time.Now().Truncate(time.Second),
			Asset:            balance.Currency,
			Amount:           big.NewInt(balance.AmountInMinors),
		}, nil
	}
	return &models.PSPBalance{}, nil
}
```

When implementing `FetchNextBalances`, use `FromPayload` to access the `models.PSPAccount` that triggered this task.

```go
func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balance, err := p.client.FetchBalance(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to fetch balance from client: %w", err)
	}

	balances := make([]models.PSPBalance, 0, 1)
	if balance != nil {
		balances = append(balances, *balance)
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
```

## Launching a new connector

In this tutorial, we've introduced the different moving parts that make up a basic Connector using DummyPay, which with its local storage gives us a way to test the connector without making requests to third-party servers.

In a real-world scenario, you'd want to build a connector that fetches data from a PSP rather than from files on the file-system.

To make this process easier, we've dockerized the Connectivity Service which allows you to run it directly from your local environment. You can bring up the project by calling docker from within the project's home directory:

```sh
$ docker compose up
```

You'll then have access to all [API endpoints](https://docs.formance.com/api#tag/Payments) via the default port of `:8080`

The [Connector installation endpoint](https://docs.formance.com/api#tag/Payments/operation/installConnector) is particularly helpful for testing the `FetchAccounts` and `FetchBalances` methods which are triggered periodically once a connector is installed.

Although the DummyPay connector is not useful outside of our integration test use-case, to demonstrate what installing a DummyPay connector would look like, let's send a POST request with the configuration payload as defined in [config.go](https://github.com/formancehq/payments/blob/main/internal/connectors/plugins/public/dummypay/config.go).

```sh
$ curl -D - \
--data '{"name":"my-dummypay-installation","directory":"./some/dir"}' \
-X POST \
http://localhost:8080/v3/connectors/install/dummypay
```

### Additional Connector Configuration

In addition to the configuration we defined explicitly in DummyPay, there are some additional configuration parameters [defined by the Connectivity service](https://github.com/formancehq/payments/blob/main/internal/models/config.go) itself, which control how polling works under the hood:

```sh
$ curl -D - \
--data '{"name":"my-dummypay-installation","directory": "./some/dir","pollingPeriod":"2m","pageSize":25}' \
-X POST \
http://localhost:8080/v3/connectors/install/dummypay
```

**name**: name of the connector installation

**pollingPeriod**: a parameter which controls how frequently the Connector will trigger Fetch\* operations in the background. The default is 2min and the smallest possible interval is 30s.

**pageSize:** useful for controlling how many records a Connector client will fetch at once from the PSP.

## Testing a connector

### Installing

Installing a connector:

- [ ] Does not return an error
- [ ] Launch the corresponding temporal schedules/workflows according to the workflow you defined
- [ ] Defines the webhooks configuration

### Uninstalling

Uninstalling a connector:

- [ ] Correctly deletes what you defined in the Uninstall method (correctly clean webhooks created on the PSP for example)

### Data Transformation

#### Account/External Account Transformation

PSP accounts should be transformed into Formance Accounts object with:

*Mandatory Fields*:

- [ ] **Reference**: the account's unique ID
- [ ] **CreatedAt**: the account's creation time
- [ ] **Raw**: PSP account json marshalled

*Optional Fields*:

- [ ] Name: Account's name if provided
- [ ] DefaultAsset: Account's default currency if provided
- [ ] Metadata: You can add whatever you want from the account inside Formance metadata

#### Balances Transformation

PSP balances should be transformed into Formance Balances object with:

*Mandatory Fields*:

- [ ] **AccountReference**: Reference of the related account
- [ ] **CreatedAt**: Creation time of the balance
- [ ] **Amount**: Balance amount
- [ ] **Asset** Currency

#### Payments Transformation

PSP payments should be transformed into Formance Payments object with:

*Mandatory Fields*:

- [ ] **Reference**: PSP payment/transcation unique id
- [ ] **CreatedAt**: Creation date of the payment/transaction
- [ ] **Type**: Payment/Transaction type (PAY-IN, PAYOUT, TRANSFER)
- [ ] **Amount**: Payment/Transaction amount
- [ ] **Asset**: Currency
- [ ] **Scheme**: Should be *models.PAYMENT_SCHEME_OTHER* if you don't know
- [ ] **Status**: Status of the payment/transaction
- [ ] **Raw**: PSP payment/transaction json marshalled

*Optional Fields*:

- [ ] ParentReference: If you're fetching transactions, in case of refunds, dispute etc...
      this reference should be the original payment reference.
- [ ] SourceAccountReference: Reference of the source account
- [ ] DestinationAccountReference: Reference of the destination account
- [ ] Metadata: You can add whatever you want from the payment inside Formance metadata

#### Others Transformation

Other data should be transformed into Formance Other object with:

*Mandatory Fields*:

- [ ] **ID**: Unique id of the PSP object
- [ ] **Other**: PSP Object json marshalled

### Fetching Data via Polling

#### Errors

- [ ] Should return *plugins.ErrNotYetInstalled* if client is nil
- [ ] Should return *plugins.ErrNotImplemented* if PSP does not have the related
      data type

#### Polling

- [ ] Must fetch all history of PSP accounts
- [ ] Installing with a different pageSize should still fetch all the history
- [ ] Should have a state in order to only fetch new accounts and not the history
      at every polling
- [ ] Once the connector has caught up to the backlog of historical data, creating a new object on the PSP (account,
      payment, etc...) should add it to Formance list of related objects
      after the next polling

#### Transformation

### Fetching Data via Webhooks

#### Webhooks Creation

- [ ] Webhooks should be created on PSP side with the right event types
- [ ] Creating a new object on PSP side should be added to Formance through
      webhooks
- [ ] Webhooks should have signatures if possible
- [ ] Polling should stop after fetching all the history if you use webhooks
      (you can do that by removing the *Periodically: true* when defining the
      connector workflow)

### Creating a Bank Account

#### Errors

- [ ] Should return *plugins.ErrNotYetInstalled* if client is nil
- [ ] Should return *plugins.ErrNotImplemented* if PSP does not handle bank account creation
- [ ] Should validate incoming bank account object and send *models.ErrInvalidRequest* error if needed

#### Bank Account Creation

- [ ] Should create the account on the PSP
- [ ] Should create the related EXTERNAL account object on Formance payments service

### Creating a Transfer/Payout

#### Errors

- [ ] Should return *plugins.ErrNotYetInstalled* if client is nil
- [ ] Should return *plugins.ErrNotImplemented* if PSP does not handle transfer/payout creation
- [ ] Should validate incoming transfer/payout object and send *models.ErrInvalidRequest* error
      if needed

#### Transfer/Payout Creation

- [ ] Must create the transfer/payout on the PSP
- [ ] If a payment is created, you must return it by using the *Payment* field of
      the response
- [ ] If it creates another entity than a payments, you must use the *PollingTransferID* field
      of the response to poll the entity until a payments is created
- [ ] Ensure that if the payment succeeds or fails later, the status of the
      related payment initiation changes also

## Special Implementation details

In order to keep the codebase neat and consistent across different implementations, there are some tips to be aware of when working on the codebase. Follow this pattern to avoid some pitfalls.

### Metadata

When working with a connector, there are cases where you need to save more data in the PSP model, but this property is not available in the PSP model. In this case, you will save this property and its value in the PSP Metadata property. There are also some scenarios where you need to collect more information when creating data, but these properties are not provided in the default creation model. Here, you will also use metadata. Below is how to set up the proper Metadata handling.

#### Use namespaces for metadata

When representing additional values, you need to use metadata to do this. First, all the metadata must be namespaced according to the connector you are working with. The metadata properties must be prefixed with the connector namespace. For instance, given a connector named `stripe`, you have to create a namespace that looks like this: `com.stripe.spec/`. This will be the prefix for all metadata created under the `stripe` connector.

Example: The Stripe connector payment endpoint is returning additional properties (`payment_reason`, `payment_arrival_time`). To represent these, we create `metadata.go` inside the client folder of the connector: `client/metadata.go`.

```go
package client

const (
	stripeMetadataSpecNamespace = "com.string.spec/"
	StripePaymentReasonMetadataKey = stripeMetadataSpecNamespace + "payment_reason"
	StripePaymentArrivalTimeMetadataKey = stripeMetadataSpecNamespace + "payment_arrival_time"
)

```

Now you are ready to use this metadata

#### Extracting Metadata values from Metadata

Given that a payload contains metadata and you need to extract its values, you can easily do that with `Metadata[StripePaymentArrivalTimeMetadataKey]`. However, to follow the codebase structure, you need to use a function from the models to extract the metadata values.

Here is an example

```go
func (p *Plugin) validateTransferRequest(pi models.PSPPaymentInitiation) error {
	hold := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnHoldMetadataKey)
}
```

As you can see in the example above, we are using the `ExtractNamespacedMetadata` function from the models package to extract the value. This is how it should be done across the codebase.

#### Save extra data in Metadata

Given that a provider returns more properties than the ones available in the PSPModel

```json
{
  "name": "Gistart",
  "amount": 500,
  "currency": "USD",
  "bank_name": "Citibank",
  "account_number": "3424325665",
  "routing_number": "783XHY29AK",
  "created_at": "2024-12-20 08:25:03",
  // properties not in psp model
  "payment_reason": "salary_payment",
  "payment_arrival_time": "2024-12-22 08:25:03"
}
```

```go
	providerPayments := p.fetchPayments()

	pspPayments []models.PSPPayment{}
	for _, payment := range payments {

		paymentReason := models.ExtractNamespacedMetadata(payment.Metadata, client.StripePaymentReasonMetadataKey)
		paymentArrivalTime := models.ExtractNamespacedMetadata(payment.Metadata, client.StripePaymentArrivalTimeMetadataKey)

		pspPayments = append(pspPayments, models.PSPAccount{
			//... all available properties her

			// Metadata properties filled with namespaced keys
			Metadata: {
				client.StripePaymentReasonMetadataKey: paymentReason,
				client.StripePaymentArrivalTimeMetadataKey: paymentArrivalTime
			}
		})
	}
```

### Asset and Amount Handling

When working with currency amounts in Formance connectors, it's crucial to understand how assets and amounts are handled. Formance uses a standardized format for both asset representation and amount storage.

#### Asset Format

Assets in Formance are represented with their precision using the format `CURRENCY/PRECISION`. For example:

- `USD/2` represents US Dollar with 2 decimal places
- `JPY/0` represents Japanese Yen with no decimal places
- `BHD/3` represents Bahraini Dinar with 3 decimal places

The currency package provides helper functions to work with assets:

```go
// Get currency and precision from an asset string
currency, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, "USD/2")
// Returns: "USD", 2, nil

// Format a currency into an asset string
asset := currency.FormatAsset(supportedCurrenciesWithDecimal, "USD")
// Returns: "USD/2"
```

#### Amount Handling

Formance stores amounts in their smallest unit (denominator). For example:

- 10.00 USD is stored as 1000 (cents)
- 1000 JPY is stored as 1000 (no decimals)
- 1.000 BHD is stored as 1000 (3 decimal places)

The currency package provides functions to handle amount conversions:

```go
// Convert string amount to smallest unit with precision
amount, err := currency.GetAmountWithPrecisionFromString("10.50", 2)
// Returns: big.NewInt(1050)

// Convert smallest unit back to string with precision
str, err := currency.GetStringAmountFromBigIntWithPrecision(big.NewInt(1050), 2)
// Returns: "10.50"
```

#### Asset and Amount handling examples

Here are examples of how to handle assets and amounts in connector implementations:

1. Creating a payout (converting from user input to PSP format):

```go
func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) error {
	// Extract currency and precision from asset (e.g., "USD/2")
	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(
		supportedCurrenciesWithDecimal,
		pi.Asset,
	)
	if err != nil {
		return fmt.Errorf("invalid asset format: %w", err)
	}

	// Amount is already in smallest unit in pi.Amount
	pspRequest := &client.PayoutRequest{
		Amount:       pi.Amount.Int64(),  // Use as is - already in cents
		CurrencyCode: curr,              // Use extracted currency code
	}
	// ...
}
```

2. Fetching balances (converting from PSP response to Formance format):

```go
func (p *Plugin) fetchBalances(balance *psp.Balance) (*models.PSPBalance, error) {
	// Get precision for the currency
	precision := supportedCurrenciesWithDecimal[balance.Currency]

	// Convert amount string to smallest unit
	amount, err := currency.GetAmountWithPrecisionFromString(
		balance.Amount,
		precision,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	return &models.PSPBalance{
		Amount: amount,                    // Amount in smallest unit
		Asset:  currency.FormatAsset(     // Format as CURRENCY/PRECISION
			supportedCurrenciesWithDecimal,
			balance.Currency,
		),
	}, nil
}
```

### Important Considerations

1. Always check if the PSP provides amounts in the smallest unit (e.g., cents) or in the main unit (e.g., dollars).
2. Define supported currencies and their precision in your connector's configuration.
3. Use the provided currency package functions to handle conversions consistently.
4. Validate asset formats and currency support early in your implementation.
5. Handle precision loss carefully when converting between formats.


### Setting up Pre-commit Checks
Pre-commit checks for the repository is done using Just.

To set up the Justfile dependencies, follow these steps:
1. Install just: https://github.com/casey/just
2. Install yq: https://github.com/mikefarah/yq
3. Install speakeasy: https://www.speakeasy.com/docs/speakeasy-reference/cli/getting-started

> Note: When installing Speakeasy, make sure to install version 1.525.0, as the latest version won't work with the project.

On Linux and macOS, you can use the guide below:

You can modify the `speakeasy.sh` install script to use the correct version of Speakeasy:

```bash
...
get_download_url() {
  local asset_name=$(get_asset_name $2 $3)
  echo "https://github.com/speakeasy-api/speakeasy/releases/download/v1.525.0/${asset_name}"
}

get_checksum_url() {
  echo "https://github.com/speakeasy-api/speakeasy/releases/download/v1.525.0/checksums.txt"
}
...
```

Then, run the following commands:
```bash
# run this command to make it executable.
chmod +x ./speakeasy.sh
./speakeasy.sh
```

4. Make sure you can run `go`, `npx` and `mockgen` with elevated privileges.
5. Run `just pre-commit` to handle linting, documentation generation and other pre-commit steps.

### Troubleshooting
1. If there are permission issues with the `just` command, try using the correct root access.
2. Some steps might fail, in this case, you can go ahead to run them individually.
   
   For example the following lines in [`Justfile`](./Justfile)
	```bash
	@npx openapi-merge-cli --config {{justfile_directory()}}/openapi/openapi-merge.json
	@yq -oy {{justfile_directory()}}/openapi.json > openapi.yaml
	@rm {{justfile_directory()}}/openapi.json
	```
	can be ran separately.

### Review Checklist
- [ ] Validate that the PSP API endpoint corresponds to the connector integration made.
- [ ] Validate that the PSP API authentication strategy matches with the connector integration.
- [ ] Validate that the PSP API request body corresponds to the connector integration.
- [ ] Validate support for all currencies listed on the PSP docs.
- [ ] Validate that currency formatting on all methods corresponds the PSP API documentation.
- [ ] Validate metadata are added to the PSP models if available.
- [ ] Ensure you don't have empty fields. Use “Unknown” as a placeholder instead.
- [ ] Validate all status covered on the API docs are formatted to `payments` payment status.
- [ ] Validate that `PAYIN` and PAYOUT status corresponds to the PSP API payment status. Incoming payouts should have `PAYIN` status and outgoing PAYOUT status.
