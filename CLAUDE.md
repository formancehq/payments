# Payments

## Development

Enter Nix dev shell before running commands: `nix develop`

## Key commands

- `just tests` — Runs the whole test suite
- `just pc` — **pre-commit**: tidy, generate, lint, openapi, compile plugins & capabilities. Run before committing.
- `just openapi` — regenerate `openapi.yaml` from `openapi/` sources (compile configs, merge, docs, validate)
- `just compile-api-yaml` — merge OpenAPI inputs only (skip docs/validation)
- `just generate` — regenerate SDK (speakeasy) + `go generate`
