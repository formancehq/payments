# Internal Development Support Tools

This directory contains a series of tools helping the develoment and build process.

## Core Build Tools

### `compile-plugins/`
**Shell script that auto-generates connector import list**
- Scans `internal/connectors/plugins/public/` directory
- Creates `list.go` with blank imports for all connectors
- Run: `just compile-plugins`

### `compile-configs/`
**Go tool that generates OpenAPI schemas from connector configs**
- Reads `config.go` files from each connector
- Parses struct tags to generate OpenAPI YAML
- Outputs: `openapi/v3/v3-connectors-config.yaml`
- Run: `just compile-connector-configs`

### `compile-capabilities/`
**Go tool that extracts connector capabilities**
- Reads connector capabilities from registry
- Generates JSON mapping of provider → capabilities
- Outputs: `docs/other/connector-capabilities.json`
- Run: `just compile-connector-capabilities`

## Development Tools

### `connector-dev-server/`
**Simple dev server for testing individual connectors**
- Imports single connector for isolated testing
- Provides basic HTTP API for connector operations
- Useful for development and debugging

### `connector-template/`
**Code generator for new connectors**
- Generates boilerplate connector structure
- Uses Go templates to create all required files
- Run: `./tools/connector-template/connector-template.sh <name>`

## Maintenance Tools

### `list-and-delete-temporal-schedules/`
**Temporal schedule management**
- Lists all Temporal schedules
- Can delete specific schedules
- Useful for cleanup and maintenance

### `list-and-delete-temporal-workflows/`
**Temporal workflow management**
- Lists all Temporal workflows
- Can delete specific workflows
- Useful for cleanup and maintenance

## Usage

All tools are orchestrated via the main `Justfile`:

```bash
just pre-commit  # Runs all build tools
just openapi     # Generates API documentation
just pc          # Alias for pre-commit
```
