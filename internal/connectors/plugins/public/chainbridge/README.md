# Chain Bridge Connector

The Chain Bridge connector integrates Formance Payments with [Chain Bridge](https://github.com/formancehq/chain-bridge), a Formance SaaS service that monitors blockchain addresses and indexes on-chain token balances. The connector is **read-only** — it polls Chain Bridge for monitors (accounts) and balances.

## Capabilities

- Fetch accounts (blockchain address monitors)
- Fetch balances (ERC-20 and native token balances)

## Installation

```json
{
  "name": "chainbridge",
  "apiKey": "your-api-key",
  "endpoint": "https://chain-bridge.formance.cloud"
}
```

### Configuration Parameters

| Parameter | Description | Required |
|---|---|---|
| `name` | The name of the connector instance | Yes |
| `apiKey` | API key for Chain Bridge authentication | Yes |
| `endpoint` | Chain Bridge API endpoint URL | Yes |
| `pollingPeriod` | Polling frequency (default: `30m`) | No |

## Data Mapping

### Monitors to Accounts

Each Chain Bridge monitor is mapped to a `PSPAccount`:

| PSPAccount Field | Source |
|---|---|
| `Reference` | `monitor.id` (Chain Bridge internal ID) |
| `Name` | `monitor.address` |
| `CreatedAt` | `monitor.createdAt` |
| `Metadata.chain` | `monitor.chain` (e.g., `ethereum`) |
| `Metadata.address` | `monitor.address` |
| `Metadata.status` | `monitor.status` (e.g., `active`) |

### Token Balances to Balances

Each Chain Bridge token balance is mapped to a `PSPBalance`:

| PSPBalance Field | Source |
|---|---|
| `AccountReference` | `balance.monitorId` |
| `Amount` | `balance.amount` (minor units as `*big.Int`) |
| `Asset` | `balance.asset` (e.g., `ETH/18`, `USDC/6`) |
| `CreatedAt` | `balance.fetchedAt` |

### Asset Validation

Chain Bridge returns all ERC-20 token balances, including tokens with names that don't match the Payments asset format (`[A-Z][A-Z0-9_]{0,16}(/\d{1,6})?`). The connector silently skips balances with invalid asset names rather than failing. Examples of skipped assets:

- `$ETH6900/9` — starts with `$`
- `DEEPSEEK R1/8` — contains a space
- `GHIBLI2.0/9` — contains a dot
- `PÊPÊ/18` — contains unicode characters

## Workflow

The connector runs two independent periodic tasks:

- `fetch_monitors` — polls `GET /monitors` for new blockchain address monitors
- `fetch_balances` — polls `GET /balances` for current token balances (snapshot semantics)

Both tasks run as siblings (not parent-child), since `GET /balances` returns all balances across all monitors in a single response.

## Testing

```bash
go test ./internal/connectors/plugins/public/chainbridge/...
```
