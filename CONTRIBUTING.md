# How to build a connector

You can build a connector for a new Payments Service Provider (PSP) or integrate new payment methods into an existing connector by using the Plugin interface.

This guide demonstrates the process of building a basic connector for a hypothetical PSP called `DummyPay` using the Plugin interface.

## Table of Contents

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
- [Testing a New Connector](#testing-a-new-connector)
    - [Additional Connector Configuration](#additional-connector-configuration)

## Understanding the Plugin interface

The [Plugin interface](https://github.com/formancehq/payments/blob/main/internal/models/plugin.go#L14-L36) defines the required methods for all connectors and serves as the blueprint for their implementation. Since it’s written in Go, Go’s type system requires all methods to be implemented to satisfy the interface, even if some are not used by the connector.

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

| Method                         | Description                                                                          |
| ------------------------------ | ------------------------------------------------------------------------------------ |
| Name()                         | Returns the name of the plugin, which is used to register the connector in the connector registry |
| Install(...)                   | Activates the connector, sets up the required configuration and start Data synchronization with the PSP |
| Uninstall(...)                 | Deactivates the connector, cleans up any resources created during installation, such as webhooks or cache data |
| FetchNextAccounts(...)         | Retrieves the next set of account data from the PSP for synchronization |
| FetchNextPayments(...)         | Retrieves the next set of payment data from the PSP for synchronization |
| FetchNextBalances(...)         | Retrieves the next set of balance data (e.g., account balances) from the PSP for synchronization |
| FetchNextExternalAccounts(...) | Retrieves external accounts (e.g., linked bank or card accounts) from the PSP for synchronization |
| FetchNextOthers(...)           | Fetches any additional or custom data from the PSP that doesn’t fall into the predefined categories |
| CreateBankAccount(...)         | Creates a new bank account or linked financial account in the PSP |
| CreateTransfer(...)            | Initiates a transfer of funds between accounts within the PSP or externally |
| ReverseTransfer(...)           | Reverses a previously initiated processed transfer |
| PollTransferStatus(...)        | Polls the status of a previously initiated transfer to determine whether it was successful, pending, or failed. Useful for PSPs whose APIs don’t provide synchronous feedback about whether or not a transfer was successful or not |
| CreatePayout(...)              | Initiates a payout from a PSP account to an external account (e.g., a bank or another PSP) |
| ReversePayout(...)             | Reverses a previously initiated payout |
| PollPayoutStatus(...)          | Polls the status of a previously initiated payout to determine whether it was successful, pending, or failed. Useful for PSPs whose APIs don’t provide synchronous feedback about whether or not a payout was successful or not |
| CreateWebhooks(...)            | Sets up webhooks in the PSP to notify the Payments Service of events (e.g., payment updates) |
| TranslateWebhook(...)          | Converts incoming webhook events from the PSP into a format that the Payments Service understands |

## Building a connector

In this tutorial, we'll build a connector for a hypothetical PSP, DummyPay, to read payment files from a directory containing fictional payments to be processed. We’ll define the connector capabilities and configuration, and use the Plugin interface to implement installation and data-fetching logic for the connector. A fully implemented version of the DummyPay connector is available in our integration testing environment. You can check out the code on GitHub as you follow along.

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

| Capability                                 | Description                      |
| ------------------------------------------ | -------------------------------- |
| CAPABILITY_FETCH_ACCOUNTS                  | Connector can fetch accounts from the PSP |
| CAPABILITY_FETCH_BALANCES                  | Connector can fetch account balances from the PSP |
| CAPABILITY_FETCH_EXTERNAL_ACCOUNTS         | Connector can fetch external accounts from the PSP |
| CAPABILITY_FETCH_PAYMENTS                  | Connector can fetch payments from the PSP |
| CAPABILITY_FETCH_OTHERS                    | Connector is going to fetch other object first from the PSP in order to be able to fetch accounts, balances, external accounts or payments from these other objects |
| CAPABILITY_CREATE_WEBHOOKS                 | Connector can create webhooks on the PSP |
| CAPABILITY_TRANSLATE_WEBHOOKS              | Connector can handle webhooks received from the PSP |
| CAPABILITY_CREATE_BANK_ACCOUNT             | Connector can create bank accounts on the PSP |
| CAPABILITY_CREATE_TRANSFER                 | Connector can create transfer between accounts on the PSP |
| CAPABILITY_CREATE_PAYOUT                   | Connector can create payout between accounts and external account on the PSP |
| CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION | Connector is allowed to have Formance account created directly from Formance API without being forwarded to the PSP. (This can be useful if the PSP does not provide a way to fetch the history of accounts, the user can directly create them via the Formance API) |
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
  // TODO: initialize a client using the config
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

In this step, we’ll define what our connector will do when it is installed. Connectors are typically designed to continuously synchronize data—such as accounts and payments—from the PSP to the local Formance instance in the background. To achieve this, we need to configure the types of data that will be synchronized.

Within the Payments Service, this is structured as a *Workflow Task Tree*. This tree can include independent parent nodes that execute in parallel, as well as child nodes that are triggered only by the completion of a parent node. To set this up, we’ll open the file called `workflow.go` in the `dummypay2` directory and define the task tree within it.

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

To connect to a PSP and fetch data, the connector needs to translate the PSP’s API responses into standardized data structures that the Plugin interface can use, such as Accounts, ExternalAccounts, Balances, and Payments.

First, let's define what an account will look like for our example. Suppose DummyPay provides an accounts list as follows:
```json
[
  {"id":"87afd68f-4441-4ec0-a5d4-342d241bbeca","name":"dummy-account-0","opening_date":"2024-12-02T17:34:07+01:00","currency":"EUR"},
  {"id":"182345d5-0546-4ae1-b470-2c6f015f7d6b","name":"dummy-account-1","opening_date":"2024-12-02T17:33:07+01:00","currency":"EUR"},
  {"id":"74f3b555-2fc0-4bd7-b171-1965bb44fa53","name":"dummy-account-2","opening_date":"2024-12-02T17:32:07+01:00","currency":"EUR"},
  {"id":"73d31ca4-78dd-4e86-aa3b-a971ff59c93e","name":"dummy-account-3","opening_date":"2024-12-02T17:31:07+01:00","currency":"EUR"},
  {"id":"367cb45d-440e-4d04-b5d8-d8603955358c","name":"dummy-account-4","opening_date":"2024-12-02T17:30:07+01:00","currency":"EUR"},
  {"id":"93068adb-aa41-4a6d-9f94-7f5ec315b57b","name":"dummy-account-5","opening_date":"2024-12-02T17:29:07+01:00","currency":"EUR"},
  {"id":"ad7baa78-dcf0-42b7-af2f-4db4a0f36208","name":"dummy-account-6","opening_date":"2024-12-02T17:28:07+01:00","currency":"EUR"},
  {"id":"5247256e-a1b8-497b-8d01-9f93a7557ef6","name":"dummy-account-7","opening_date":"2024-12-02T17:27:07+01:00","currency":"EUR"},
  {"id":"55bb43d1-bf5a-434d-95d9-222e75c189cc","name":"dummy-account-8","opening_date":"2024-12-02T17:26:07+01:00","currency":"EUR"},
  {"id":"440cc4b7-138f-4c6a-8120-1190e6e02714","name":"dummy-account-9","opening_date":"2024-12-02T17:25:07+01:00","currency":"EUR"}
]
```

Let’s create a DummyPay client which will read and unmarshal the json file, and convert it to a data structure that the payments service knows how to use.

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

Now that we’ve defined a way to ingest the data, let’s integrate this into `plugin.go` and connect it with the `FetchNextAccounts` function.
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
  {"account_id":"ab59834d-e94a-4547-b908-4be3eed9ed68","amount_in_minors":23,"currency":"EUR"},
  {"account_id":"f0946302-05c7-4775-84ed-81d19b39cb26","amount_in_minors":123,"currency":"EUR"},
  {"account_id":"3b1af736-9ca7-4a24-98d5-c626405f5fef","amount_in_minors":223,"currency":"EUR"},
  {"account_id":"ea651da2-0810-4422-8a1f-159b6e15bcee","amount_in_minors":323,"currency":"EUR"},
  {"account_id":"2f6d8a3c-f414-45a2-9c0a-2065c1ade017","amount_in_minors":423,"currency":"EUR"},
  {"account_id":"527d3a1f-ed48-4425-97be-3d5e97a9beb0","amount_in_minors":523,"currency":"EUR"},
  {"account_id":"152ed96b-8aba-4431-82ed-1478caad8eba","amount_in_minors":623,"currency":"EUR"},
  {"account_id":"cd680304-3769-4750-a4b7-017c4fc20803","amount_in_minors":723,"currency":"EUR"},
  {"account_id":"6cadea83-bac2-4ced-adcd-8758f4712aeb","amount_in_minors":823,"currency":"EUR"},
  {"account_id":"aac11dd3-360b-424b-a9b8-3f85337793e0","amount_in_minors":923,"currency":"EUR"}
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

In this tutorial, we’ve introduced the different moving parts that make up a basic Connector using DummyPay, which with its local storage gives us a way to test the connector without making requests to third-party servers.

In a real-world scenario, you’d want to build a connector that fetches data from a PSP rather than from files on the file-system.

To make this process easier, we’ve dockerized the Connectivity Service which allows you to run it directly from your local environment. You can bring up the project by calling docker from within the project’s home directory:

```sh
$ docker compose up
```

You’ll then have access to all [API endpoints](https://docs.formance.com/api#tag/Payments) via the default port of `:8080`

The [Connector installation endpoint](https://docs.formance.com/api#tag/Payments/operation/installConnector) is particularly helpful for testing the `FetchAccounts` and `FetchBalances` methods which are triggered periodically once a connector is installed.

Although the DummyPay connector is not useful outside of our integration test use-case, to demonstrate what installing a DummyPay connector would look like, let’s send a POST request with the configuration payload as defined in [config.go](https://github.com/formancehq/payments/blob/main/internal/connectors/plugins/public/dummypay/config.go).

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

**pollingPeriod**: a parameter which controls how frequently the Connector will trigger Fetch* operations in the background. The default is 2min and the smallest possible interval is 30s.

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