# Formance Payments Project Guidelines

## Project Overview
Formance Payments is a framework for ingesting payments (both payin and payout) from different payment service providers (PSPs). The framework contains connectors that translate PSP-specific formats to a generalized format used by Formance. It's designed to be extensible, allowing for the addition of new connectors.

## Project Structure
- `/cmd`: Command-line interface entry points
- `/internal`: Internal packages not meant to be imported by other projects
  - `/api`: API definitions and handlers
  - `/connectors`: Connector implementations for different PSPs
    - `/engine`: Core engine for connector workflows
    - `/plugins`: Individual PSP connector implementations
  - `/models`: Data models used throughout the application
  - `/storage`: Database access and migrations
  - `/utils`: Utility functions
- `/pkg`: Public packages that can be imported by other projects
  - `/client`: Client library for interacting with the Payments API
- `/test`: Test files and utilities
- `/tools`: Development and build tools

## Development Guidelines

### Running the Project
The project can be run using Docker Compose:
```sh
earthly -P +compile-plugins --local_save=true
docker compose up
```

This will start all necessary services:
- PostgreSQL database
- Temporal workflow engine
- Payments API server (port 8080)
- Payments worker

### Testing
When implementing changes, Junie should run tests to verify the correctness of the solution:

```sh
go test ./...
```

For specific connector tests:
```sh
go test ./internal/connectors/plugins/public/<connector_name>/...
```

### Building Connectors
When working on connectors, follow these guidelines:
1. Use the Plugin interface defined in `internal/models/plugin.go`
2. Implement required methods (Install, Uninstall) and other methods as needed
3. Handle metadata properly with namespaced keys
4. Follow asset and amount handling conventions
5. Use state management for paginated data fetching

### Code Style
- Follow Go best practices and idiomatic Go
- Use proper error handling with wrapped errors
- Document public functions and types
- Use meaningful variable and function names
- Keep functions small and focused on a single responsibility

### Pre-commit Checks
Before submitting changes, run pre-commit checks:
```sh
just pre-commit
```

This will handle linting, documentation generation, and other pre-commit steps.

## Deployment
The project is designed to be deployed as a set of containerized services. The main components are:
1. The Payments API server
2. The Payments worker
3. PostgreSQL database
4. Temporal workflow engine

Configuration is primarily done through environment variables, as shown in the docker-compose.yml file.
