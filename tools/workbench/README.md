# Connector Workbench

A lightweight development environment for building and testing payment connectors without requiring Temporal, PostgreSQL, or other heavy infrastructure.

## Why?

Running the full payments stack locally requires Docker, Temporal, PostgreSQL, and more - which can be slow and resource-intensive. The workbench provides a stripped-down alternative that lets you:

- **Develop connectors faster** - No Docker, no database, just run and test
- **Debug easily** - See all HTTP traffic, plugin calls, and state changes
- **Iterate quickly** - Hot reload-friendly, instant feedback

## How It Works

The workbench is an alternative entry point to the same binary:

```
payments binary
    │
    ├── payments server      ← Full production mode (Temporal + PostgreSQL)
    │
    └── payments workbench   ← Lightweight dev mode (in-memory)
```

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Workbench                                                       │
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   Engine    │    │   Storage   │    │  DebugStore │         │
│  │ (in-memory) │    │ (in-memory) │    │  (captures) │         │
│  └──────┬──────┘    └─────────────┘    └─────────────┘         │
│         │                                                       │
│         ▼                                                       │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Plugin (e.g., wise, stripe)                            │   │
│  │  - Loaded via registry                                   │   │
│  │  - Same code as production                               │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### The In-Memory Engine

In production, connector operations are orchestrated by Temporal workflows:

```
Temporal Workflow
    └─→ Schedule activity → Retry on failure → Persist state
```

The workbench replaces this with direct function calls:

```go
// engine.go - simplified orchestration
func (e *Engine) RunOneCycle(ctx context.Context) error {
    for _, task := range *tree {
        e.executeTask(ctx, task, nil)  // Direct call, no Temporal
    }
}
```

This means:
- **No Temporal needed** - Tasks execute synchronously
- **No PostgreSQL needed** - State stored in memory
- **Same connector code** - Plugin interface is identical

### Connector Loading

Connectors self-register via `init()`:

```go
// In wise/plugin.go
func init() {
    registry.RegisterPlugin("wise", connector.PluginTypePSP, func(...) {
        return New(name, logger, config)
    }, ...)
}
```

The workbench asks the registry for a connector by name:

```go
// In workbench.go
plugin, err := registry.GetPlugin(connectorID, logger, "wise", name, config)
```

## Usage

### Basic Usage

```bash
# Run with inline config
payments workbench --provider=wise --config='{"apiKey":"your-api-key"}'

# Run with config file
payments workbench --provider=stripe --config-file=./stripe-config.json

# List available providers
payments workbench --list-providers
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--provider, -p` | Connector provider name (required) | - |
| `--config, -c` | Connector config as JSON string | - |
| `--config-file, -f` | Path to config JSON file | - |
| `--listen` | HTTP server address | `127.0.0.1:8080` |
| `--auto-poll` | Enable automatic polling | `false` |
| `--poll-interval` | Polling interval | `30s` |
| `--page-size` | Page size for fetch operations | `25` |

### Web UI

Once running, open http://localhost:8080 in your browser to access the debug UI:

- **Debug Log** - All plugin calls, HTTP requests, state changes
- **Tasks** - Workflow task tree and execution status
- **Data** - Fetched accounts, payments, balances
- **HTTP Traffic** - Captured requests/responses to PSP APIs

### HTTP API

```bash
# Install connector (initializes workflow)
curl -X POST http://localhost:8080/api/install

# Run one fetch cycle
curl -X POST http://localhost:8080/api/cycle

# Get fetched data
curl http://localhost:8080/api/accounts
curl http://localhost:8080/api/payments
curl http://localhost:8080/api/balances

# Get debug info
curl http://localhost:8080/api/debug
```

## Key Components

| File | Purpose |
|------|---------|
| `workbench.go` | Main orchestrator, lifecycle management |
| `engine.go` | In-memory Temporal replacement |
| `storage.go` | In-memory data storage |
| `debug.go` | Debug info capture |
| `transport.go` | HTTP traffic interception |
| `server.go` | HTTP API and UI serving |
| `tasks.go` | Task tree tracking |

## Relation to `pkg/connector`

The workbench imports from `internal/models` (canonical types) because it's a **platform tool**, not a connector. Connectors themselves should import from `pkg/connector` (the public API):

```
┌─────────────────────┐
│  pkg/connector      │  ← Public API for connectors
│  (type aliases)     │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  internal/models    │  ← Canonical types
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  tools/workbench    │  ← Uses internal/models (platform tool)
└─────────────────────┘
```

## Example Session

```bash
# 1. Start workbench with wise connector
$ payments workbench -p wise -f ./wise-config.json
INFO HTTP debug transport installed
INFO Connector workbench starting
INFO Web UI available at http://127.0.0.1:8080

# 2. In another terminal, install and run
$ curl -X POST localhost:8080/api/install
{"status":"installed","tasks":3}

$ curl -X POST localhost:8080/api/cycle
{"status":"completed","accounts":5,"payments":23}

# 3. Check the web UI for detailed debug info
# Open http://localhost:8080
```

## Limitations

- **No persistence** - Data is lost on restart (by design)
- **No webhooks** - Webhook endpoints are mocked
- **Single connector** - One connector per workbench instance
- **No retries** - Errors fail immediately (easier to debug)

These limitations are intentional - the workbench is for development, not production.
