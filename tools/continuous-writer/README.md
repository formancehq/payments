## Continuous Payment Worker

This is a small Go program that continuously inserts rows into the public.payments table.
It is intended to be run ad-hoc (outside the main application) to generate write load during
migrations or operational testing.

Build

  go build ./tools/continuous-writer

Run

Environment variables (flags override env):
- PG_DSN: Postgres DSN (default: postgres://payments:payments@localhost:5432/payments?sslmode=disable)
- INTERVAL_MS: Delay between inserts per worker in milliseconds (default: 1000)
- WORKERS: Number of concurrent workers (default: 5)
- BATCH_SIZE: Number of rows per INSERT (default: 500)
- FLUSH_MS: Max time to wait before flushing a partial batch in milliseconds (default: 10000)
- ASSET: Asset string (default: USD/2)
- SCHEME: Scheme string (default: test)
- TYPE: payin|payout|transfer (default: payin)
- CONFIG_ENCRYPTION_KEY: Database encryption key to encrypt connector config (default: mysuperencryptionkey)

Notes on connector handling:
- The worker will ensure a connector row exists in public.connectors with a generated ID (provider: qonto) and set an example encrypted config using CONFIG_ENCRYPTION_KEY. It logs warnings if this upsert or encryption update fails.

Examples:

  PG_DSN=postgres://payments:payments@localhost:5432/payments?sslmode=disable \
  INTERVAL_MS=50 WORKERS=4 BATCH_SIZE=200 \
  ASSET=USD/2 SCHEME=test TYPE=payin \
  go run ./tools/continuous-writer

Or with flags:

  go run ./tools/continuous-writer \
    -dsn "postgres://payments:payments@localhost:5432/payments?sslmode=disable" \
    -workers 4 -interval-ms 50 -type payout \
    -batch-size 200 -scheme test -asset USD/2 -encryption-key mysuperencryptionkey

Notes
- The program handles SIGINT/SIGTERM and will shut down gracefully.
- It uses the minimal set of required columns for public.payments based on the current schema.
- Adjust DSN to match your docker-compose Postgres (see docker-compose.yml).
- Batching: rows are accumulated per worker and flushed either when BATCH_SIZE is reached or after FLUSH_MS, whichever comes first.
- Logging includes batch insert counts and connector ensure logs.
