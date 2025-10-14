# Formance Payments Project Guidelines

This document consolidates development guidelines for working on the Formance Payments project.

If you are contributing a new connector, first read [CONTRIBUTING.md](./CONTRIBUTING.md) for the full end-to-end connector tutorial. This file focuses on day-to-day development workflow and conventions.

## Project Overview
Formance Payments is a framework for ingesting payments (payins and payouts) from different payment service providers (PSPs). Connectors translate PSP-specific formats to a generalized format and run under a unified engine.

## Project Structure
- /cmd: CLI entry points
- /internal: Internal packages
  - /api: API definitions and handlers
  - /connectors: Connectors for different PSPs
    - /engine: Core engine for connector workflows
    - /plugins: PSP connector implementations
  - /models: Shared data models
  - /storage: Database access and migrations
  - /utils: Utility functions
- /pkg: Public packages
  - /client: Client for the Payments API
- /test: Tests and utilities
- /tools: Dev/build tools

## Running the Project
Use Docker Compose after compiling plugins:

```sh
just compile-plugins
docker compose up
```

- Earthly: https://earthly.dev
- Docker Compose: https://docs.docker.com/compose/
- Dev compose setup: see [docker-compose.dev.yml](./docker-compose.dev.yml) and [docker-compose.yml](./docker-compose.yml)

Services started:
- PostgreSQL
- Temporal
- Payments API server (port 8080)
- Payments worker

## Testing
Run all tests:
```sh
go test ./...
```

Run tests for a specific public connector:
```sh
go test ./internal/connectors/plugins/public/<connector_name>/...
```

## Building Connectors
Follow the Plugin interface in [internal/models/plugin.go](./internal/models/plugin.go). Implement at least:
- Install
- Uninstall

And implement other methods as needed by the connector. Ensure:
- Metadata uses namespaced keys
- Asset and amount handling follow conventions
- State management persists and resumes pagination

For a comprehensive tutorial, see [CONTRIBUTING.md](./CONTRIBUTING.md) at the repo root. You may also find the Payments API reference helpful: https://docs.formance.com/api-reference/paymentsv3/

## Code Style
- Idiomatic Go practices
- Proper error wrapping and context
- Document public functions/types
- Meaningful names
- Small, focused functions

## Pre-commit Checks
Run before submitting changes:
```sh
just pre-commit
```
This runs linting, code generation, and related steps.

- Just: https://github.com/casey/just
- OpenAPI merge tool mentioned in Justfile requires Node.js (npx). See [Justfile](./Justfile).

## Deployment
The project is deployed as containerized services:
1. Payments API server
2. Payments worker
3. PostgreSQL
4. Temporal

Configure via environment variables (see [docker-compose.yml](./docker-compose.yml)).
