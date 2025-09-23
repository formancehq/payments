Continuous Payments Writer

This is a small Go program that continuously inserts rows into the public.payments table.
It is intended to be run ad-hoc (outside the main application) to generate write load during
migrations or operational testing.

Build

  go build ./tools/continuous-writer

Run

Environment variables (flags override env):
- PG_DSN: Postgres DSN (default: postgres://payments:payments@localhost:5432/payments?sslmode=disable)
- INTERVAL_MS: Delay per worker between row generations in milliseconds (default: 20)
- WORKERS: Number of concurrent workers (default: 5)
- BATCH_SIZE: Number of rows to accumulate per INSERT (default: 1 = single-row)
- FLUSH_MS: Max time to wait before flushing a partial batch (default: INTERVAL_MS)
- CONNECTOR_ID: Connector ID to stamp on rows (default: test-connector)
- CREATE_CONNECTOR: If 'false', skip creating the connector row (default: true)
- CONNECTOR_NAME: Name for public.connectors (default: CONNECTOR_ID)
- CONNECTOR_PROVIDER: Provider (text) for public.connectors (default: OTHER)
- CONNECTOR_CONFIG_BASE64: Base64-encoded config bytes for public.connectors (optional)
- ASSET: Asset string (default: USD/2)
- SCHEME: Scheme string (default: test)
- TYPE: payin|payout|transfer (default: payin)

Example:

  PG_DSN=postgres://payments:payments@localhost:5432/payments?sslmode=disable \
  INTERVAL_MS=50 WORKERS=4 BATCH_SIZE=200 CONNECTOR_ID=my-connector \
  go run ./tools/continuous-writer

Or with flags:

  go run ./tools/continuous-writer \
    -dsn "postgres://payments:payments@localhost:5432/payments?sslmode=disable" \
    -workers 4 -interval-ms 50 -connector my-connector -type payout \
    -create-connector=true -connector-name "My Connector" -connector-provider OTHER \
    -batch-size 200

Notes
- The program handles SIGINT/SIGTERM and will shut down gracefully.
- It uses the minimal set of required columns for public.payments based on the provided schema.
- Adjust DSN to match your docker-compose Postgres (see docker-compose.yml).
