# Universal Connector — Mock Counterparty

A self-contained, single-tenant, in-memory reference implementation of the
Universal Connector contract (`../contract/universal-openapi.yaml`).
Run it locally to install the Universal CE Connector against a known-good
fixture without standing up a real PSP.

## Quick run (mock alone)

```bash
# Standalone Go binary
go run ./internal/connectors/plugins/public/universal/mock

# Or via Docker (single-binary scratch image)
docker build -t formance/universal-mock \
  -f internal/connectors/plugins/public/universal/mock/Dockerfile .
docker run --rm -p 8080:8080 -e MOCK_API_KEY=dev-key formance/universal-mock
```

## Full local stack — Payments + mock + Console + Temporal UI

The repo ships a `docker-compose.universal-mock.yml` overlay (kept at
the repo root alongside the other compose files for clean
build-context resolution) that layers on top of the existing dev
stack. From the repo root:

```bash
# 1. Build + start everything (postgres, temporal, payments,
#    payments-worker, console, gateway, AND the universal-mock).
#    First run takes ~2-3 min for the dev image build.
docker compose \
  -f docker-compose.yml \
  -f docker-compose.dev-override.yml \
  -f docker-compose.universal-mock.yml \
  up -d --build

# 2. Wait ~30s for migrations + worker to come up, then install the
#    Universal connector against the mock. The script is idempotent —
#    re-running uninstalls + reinstalls cleanly.
./internal/connectors/plugins/public/universal/mock/install.sh
```

The mock service has no `build:` section of its own — it reuses the
`payments-dev:latest` image that `payments` and `payments-worker`
already build, and just runs `go run ./internal/connectors/plugins/public/universal/mock`
as its command. This avoids any path-resolution headaches when the
compose files compose, and saves one image build per cycle.

The script:

1. Sanity-checks `payments` and `universal-mock` are reachable.
2. Removes any prior install named `universal-mock` (idempotent).
3. POSTs to `/api/payments/v3/connectors/install/Universal` with the
   right config — `endpoint=http://universal-mock:8080`,
   `apiKey=dev-key`, `webhookSharedSecret=dev-secret`,
   `pollingPeriod=20m`.
4. Waits for the install workflow to finish bootstrap.
5. Prints a status snapshot (accounts, payments, orders).
6. Prints a copy-paste menu of useful follow-up commands (fast-forward
   evolution, push synthetic webhooks, inspect adjustments,
   initiate payouts).

### Endpoints exposed

| Service              | Host port | Purpose                                                |
|----------------------|-----------|--------------------------------------------------------|
| `payments`           | `8080`    | Payments v3 API + webhook landing zone                 |
| `payments-worker`    | `9191`    | Temporal worker — health endpoint                      |
| `gateway` (Caddy)    | `8092`    | Reverse proxy                                          |
| `console`            | `3000`    | Formance UI                                            |
| `temporal-ui`        | `8081`    | Temporal workflow inspector                            |
| `universal-mock`     | `8090`    | The counterparty fixture (8080 inside docker)          |
| `postgres`           | `5432`    | Shared DB                                              |

### What to watch

```bash
# Real-time mock logs — watch poll-driven evolution + webhook deliveries.
docker compose \
  -f docker-compose.yml \
  -f docker-compose.universal-mock.yml \
  logs -f universal-mock

# Real-time worker logs — Temporal activities running.
docker compose \
  -f docker-compose.yml \
  -f docker-compose.dev-override.yml \
  logs -f payments-worker

# Open the Temporal UI to see workflows in flight.
open http://localhost:8081

# Open the Console to drive the engine through the UI.
open http://localhost:3000
```

### Logging & correlation

The mock logs every interesting event with a structured `slog` text
handler. Each HTTP request gets a 12-char request ID (`rid=…`) that
appears on every log line for that request **and** is echoed back to
the caller as the `X-Mock-Request-ID` response header — letting you
correlate engine-side activity with mock-side activity end-to-end.

Default level is `info`; `MOCK_LOG_LEVEL=debug` adds per-record lane
progressions and per-request traces.

```text
# At INFO (default) — high-signal events:
level=INFO msg="capabilities discovery"     rid=… supported_count=12 signature_scheme=hmac-sha256
level=INFO msg="webhook subscription created" rid=… id=sub_payment.updated event=payment.updated callback=…
level=INFO msg="poll-driven evolution"      rid=… records_advanced=20 first_ref=pay_00028
level=INFO msg="payout initiated"           rid=… reference=po-1 amount=50000 mode=polling polling_id=ppayout_po-1
level=INFO msg="admin evolve"               rid=… requested=15 advanced=15 webhooks_delivered=5
level=INFO msg="admin trigger delivered"    rid=… event=payment.updated callback=…
level=WARN msg="auth rejected"              rid=… have_header=false scheme_ok=false
level=INFO msg="← response"                 rid=… status=200 ms=12

# At DEBUG — every state transition + every webhook attempt:
level=DEBUG msg="→ request"                 rid=… query="n=5" idem=…
level=DEBUG msg="evolved payment"           ref=pay_00000 from=PENDING to=AUTHORISATION remaining=2
level=DEBUG msg="evolved order"             ref=ord_00000 from=PENDING to=OPEN fill_pct=0 remaining=4
level=DEBUG msg="auto-emit skipped (no subscription)" rid=… event=payment.updated ref=pay_00000
level=DEBUG msg="auto-emit delivered"       rid=… event=payment.updated ref=pay_00037 callback=…
level=DEBUG msg="listed payments"           rid=… count=100 has_more=true
```

### Tear down

```bash
docker compose \
  -f docker-compose.yml \
  -f docker-compose.dev-override.yml \
  -f docker-compose.universal-mock.yml \
  down -v   # -v wipes postgres state too — drop it for a clean re-run
```

## Env vars

| Var                          | Default          | Purpose                                                                                        |
|------------------------------|------------------|------------------------------------------------------------------------------------------------|
| `MOCK_PORT`                  | `8080`           | TCP port to listen on                                                                          |
| `MOCK_API_KEY`               | `dev-key`        | Bearer token the plugin must send                                                              |
| `MOCK_WEBHOOK_SECRET`        | `dev-secret`     | HMAC-SHA256 secret for outbound webhooks                                                       |
| `MOCK_WEBHOOK_SIGNATURE`     | `hmac-sha256`    | `hmac-sha256` to enforce signing, `none` to skip                                               |
| `MOCK_CAPABILITIES`          | full superset    | Comma-separated subset, e.g. `FETCH_ACCOUNTS,FETCH_PAYMENTS`                                   |
| `MOCK_EVOLVE_ON_POLL`        | `true`           | Each `GET /v1/{payments,orders,…}` advances `MOCK_AUTO_EVOLVE_BATCH` records IF no webhooks    |
| `MOCK_AUTO_EVOLVE_BATCH`     | `10`             | Records advanced per poll (or per ticker tick) — round-robin across payments/orders/conversions |
| `MOCK_AUTO_EVOLVE_INTERVAL`  | `0` (off)        | Optional wall-clock ticker for pure-demo runs without an installed engine                      |
| `MOCK_LOG_LEVEL`             | `info`           | `debug` / `info` / `warn` / `error`. Debug emits per-record lane progressions + per-request traces |

## Seeded data

| Endpoint                | Records | Notes                                                            |
|-------------------------|---------|------------------------------------------------------------------|
| `/v1/accounts`          |   5     | `acct_internal_000`..`004`, alternating EUR / USD / GBP / JPY    |
| `/v1/external-accounts` |   5     | `acct_ext_000`..`004`                                            |
| `/v1/payments`          | 250     | All start `PENDING`; assigned a **lane** for adjustment evolution (see below) |
| `/v1/orders`            | 150     | All start `PENDING`; assigned a fill-progression lane            |
| `/v1/conversions`       |  50     | All start `PENDING`; lane drives toward `COMPLETED` or `FAILED`  |
| `/v1/others/report`     |  30     | Opaque payloads                                                  |

Sizes are deliberately above the engine's default `PAGE_SIZE` of 100 so
the **cursor pagination state machine on the plugin side gets exercised
end-to-end** on first install — payments and orders both span multiple
pages.

Initial timestamps are seeded as `2026-01-01T00:00:00Z + index
minutes`. Every subsequent state transition (driven by auto-evolve or
`/_admin/evolve`) bumps `updatedAt = now`, so the engine's
`updatedAtFrom` cursor walks forward correctly across polls.

Account balances are served from `/v1/accounts/{id}/balances` for every
seeded internal account.

## Pagination

Every paginated `GET` honours `cursor`, `page` (1-based), `pageSize`, and
`updatedAtFrom`. Cursor is an opaque base64-encoded integer offset (easy
to debug; `echo MTAw | base64 -d` → "100") and is canonical when present.
`page=N` is honoured when `cursor` is absent. `pageSize` defaults to 100
to match the engine.

Test it manually:

```bash
curl -sS -H 'Authorization: Bearer dev-key' \
  'http://localhost:8080/v1/payments?pageSize=50' | jq '.hasMore, .nextCursor'
# → true
# → "NTA"   (base64-encoded "50")
```

## Polling state machine demo

`POST /v1/payouts` returns:

- `{mode: "terminal", payment: {...SUCCEEDED...}}` if `amount <= 10000`
  (i.e. €100.00 in minor units)
- `{mode: "polling", pollingID: "ppayout_<reference>"}` otherwise; three
  successive `GET /v1/payouts/{id}` calls return `PENDING`, then `SUCCEEDED`.

`POST /v1/transfers` follows the **same machine but separate state** — its
polling IDs are namespaced `ptransfer_*` and its idempotency keys live in
their own map. This guarantees same-key requests across primitives don't
collide (mirrors how a real PSP separates payouts and transfers).

This exercises both branches of the contract's terminal-or-polling envelope
end-to-end so you can validate Temporal's polling workflow with a single
local install.

### Payouts and transfers are visible in `/v1/payments`

Every payout/transfer initiated via `POST /v1/payouts` or
`POST /v1/transfers` is also written to the seeded payments list with
`Payment.Reference == InitiationRequest.Reference`. This means:

- `GET /v1/payments` will surface the new transaction the next time the
  engine polls.
- The Reference correlation lets the engine link the
  `PaymentInitiationAdjustment` trail (engine-derived from the poll loop)
  to the `PaymentAdjustment` trail (engine-derived from FetchNextPayments).
- A polling payout shows up as `PENDING` immediately, then transitions
  to `SUCCEEDED` on the third poll — exactly mirroring what a real PSP
  surfaces in its general transactions list.

## Driving adjustments — paced by the engine, not the wall clock

Every seeded record starts in `PENDING` and is assigned a "lane" — a
pre-planned sequence of next statuses. The mock advances records
through their lanes and bumps `updatedAt = now`, so the engine's next
poll observes the change and derives a fresh
`PaymentAdjustment` / `OrderAdjustment`.

The connector's minimum polling period is **20 minutes** (per
[`sharedconfig`](../../../sharedconfig/polling_period.go)). A wall-clock
ticker on the mock side would advance the dataset hundreds of times
between polls — the engine would see records jump from `PENDING`
straight to `SUCCEEDED` (or even REFUNDED) without observing the
intermediate states. So evolution is wired to the **engine's actual
poll cadence**, gated on whether webhooks are active.

### 1. Poll-driven evolution (default, when no webhooks registered)

Every `GET /v1/payments`, `/v1/orders`, `/v1/conversions`, `/v1/accounts`,
`/v1/external-accounts`, `/v1/others/{name}` advances
`MOCK_AUTO_EVOLVE_BATCH` records (default 10) **before** serving — so
the engine observes the new state on the same response. With the
20-minute polling period, each poll cycle yields 10 fresh adjustments
to derive, exactly mirroring how a real PSP would surface a steady
drip of changes between polls.

The number of adjustments produced per polling cycle = `evolveBatch`,
so a fresh install needs ~75 poll cycles (~25 hours) to drain the full
seed dataset. Bump `MOCK_AUTO_EVOLVE_BATCH` to compress the demo —
e.g. `MOCK_AUTO_EVOLVE_BATCH=100` covers everything in ~8 cycles.

### 2. Webhook-driven auto-emission (when CREATE_WEBHOOKS has run)

As soon as the engine's CREATE_WEBHOOKS one-shot task registers any
subscription, **poll-driven evolution stops** — polls become quiet
heartbeats, and state changes propagate via webhook pushes instead.

Every **payment** advanced through `/_admin/evolve` (or by the
optional wall-clock ticker) **automatically emits a matching event**
for any registered subscription:

| Evolved record | Event emitted        | Resource attached |
|----------------|----------------------|-------------------|
| payment        | `payment.updated`    | full Payment      |
| order          | — (no webhook surface; pull via `FetchNextOrders`) |
| conversion     | — (no webhook surface; pull via `FetchNextConversions`) |

The engine's `WebhookResponse` struct exposes no Order or Conversion
field, so the contract intentionally has no `order.*` /
`conversion.*` events to subscribe to. Order / conversion evolutions
still happen on every `EvolveSteps` call (the dataset progresses), but
they're observed by the engine through the periodic
`FetchNextOrders` / `FetchNextConversions` polls — see
[`contract/webhooks.md`](../contract/webhooks.md) "Subscribed events"
for the rationale.

So in webhook-mode the loop is:

```text
POST /_admin/evolve?n=20 → 20 records advance → mock POSTs N signed payment events
                          ↓                          (orders / conversions evolve too,
            (only those whose event type is subscribed)   but stay quiet on the wire)
                          ↓
            engine VerifyWebhook → TranslateWebhook → PaymentAdjustment
```

The response body reports both counts:

```bash
curl -sS -X POST -H 'Authorization: Bearer dev-key' \
  'http://localhost:8080/_admin/evolve?n=20'
# → {"advanced":20,"webhooksDelivered":7}
# advanced = total records mutated (payments + orders + conversions)
# webhooksDelivered = subset that had a matching subscription (payments only today)
```

You can also push a single synthetic event manually with
`POST /_admin/trigger-webhook?name=<event>`:

```bash
curl -sS -X POST -H 'Authorization: Bearer dev-key' \
  'http://localhost:8080/_admin/trigger-webhook?name=payment.updated'
```

Both paths sign deliveries with HMAC-SHA256 and post the same envelope
shape, so the engine's `VerifyWebhook → TranslateWebhook` code path is
identical for manual triggers and auto-emitted events.

### 3. `POST /_admin/evolve` (manual override, always available)

```bash
curl -sS -X POST -H 'Authorization: Bearer dev-key' \
  'http://localhost:8080/_admin/evolve?n=500'
# → {"advanced": 500}
```

Bypasses both gates above. Used by tests, demos that want to
"fast-forward" the dataset, or to drive evolution in webhook-mode
without writing a webhook trigger.

### 4. `MOCK_AUTO_EVOLVE_INTERVAL` (opt-in wall-clock ticker)

For pure demos where no engine is polling (just `curl` from the
terminal), set `MOCK_AUTO_EVOLVE_INTERVAL=10s` to spin up a background
goroutine that calls `EvolveSteps(MOCK_AUTO_EVOLVE_BATCH)` every tick.
Off by default — the engine's polling is the canonical driver.

### Lane catalogue

Every payment is assigned one of these trajectories at seed time:

```
{"AUTHORISATION", "CAPTURE", "SUCCEEDED"}                  // card-flow happy path
{"AUTHORISATION", "CAPTURE_FAILED"}                        // card-flow capture fails
{"SUCCEEDED"}                                              // direct success
{"FAILED"} / {"CANCELLED"} / {"EXPIRED"}                   // direct terminal failure
{"SUCCEEDED", "REFUNDED"}                                  // simple refund
{"SUCCEEDED", "REFUNDED", "REFUND_REVERSED"}               // refund reversed
{"SUCCEEDED", "REFUNDED_FAILURE"}                          // refund failed
{"SUCCEEDED", "DISPUTE", "DISPUTE_WON"}                    // dispute won
{"SUCCEEDED", "DISPUTE", "DISPUTE_LOST"}                   // dispute lost
```

Every order is assigned one of:

```
OPEN → PARTIALLY_FILLED@25 → @50 → @75 → FILLED            // gradual fill
OPEN → FILLED@100                                          // immediate fill
OPEN → PARTIALLY_FILLED@33 → CANCELLED@33                  // cancelled mid-fill
OPEN → EXPIRED                                             // unfilled expiry
FAILED                                                     // direct failure
```

Distributed round-robin across the 250 / 150 seeded records so the
union covers every adjustment path in
[`internal/models/payment_status.go`](../../../../models/payment_status.go) +
[`internal/models/order_status.go`](../../../../models/order_status.go).

### Typical local-install adjustment-test loop

```bash
# 1. Install Universal connector against the mock with
#    capabilities matching the test you want to run.
#    For polling-mode (no webhooks):
#      capabilityOverrides = "FETCH_PAYMENTS,FETCH_ORDERS,FETCH_ACCOUNTS,FETCH_BALANCES"
#    For webhook-mode (full superset):
#      no overrides
#
# 2a. POLLING MODE — no extra action needed. Each FetchNextPayments
#     poll (~20min cadence) advances 10 records. Bump
#     MOCK_AUTO_EVOLVE_BATCH=100 to compress the demo, OR drop the
#     polling period via the connector config (pollingPeriod = "20m"
#     is the floor).
#
# 2b. WEBHOOK MODE — push events on demand:
curl -sS -X POST 'http://localhost:8080/_admin/trigger-webhook?name=payment.updated' \
  -H 'Authorization: Bearer dev-key'
#     Each push delivers one signed event; the engine derives a
#     fresh PaymentAdjustment immediately.
#
# 2c. ANY MODE — manual fast-forward:
curl -sS -X POST 'http://localhost:8080/_admin/evolve?n=500' \
  -H 'Authorization: Bearer dev-key'
#
# 3. Inspect history:
#   GET /v3/payments/{id} → adjustments[]
#   GET /v3/orders/{id}   → adjustments[]
#   GET /v3/payment-initiations/{id} → adjustments[]
```

## Webhook trigger (debug)

Once the install registered subscriptions, push a synthetic event:

```bash
curl -sS -X POST 'http://localhost:8080/_admin/trigger-webhook?name=payment.updated' \
  -H 'Authorization: Bearer dev-key'
```

The mock signs the body with HMAC-SHA256 and posts it to whichever
`callbackUrl` Formance registered — exercising VerifyWebhook + TranslateWebhook
without waiting for the mock to autonomously fire events.

## Future: multi-tenancy

Today's mock is single-tenant. The planned evolution keeps backward
compatibility (single-tenant becomes "default tenant") while introducing:

- **Per-tenant store**: `map[tenantID]*store`, key derived from a
  `X-Tenant-ID` header or a path prefix.
- **Per-tenant capability matrix**: every tenant declares its own
  `/v1/capabilities`, so a single mock can simulate multiple counterparty
  configurations from one binary.
- **Latency / fault injection**: per-tenant or per-endpoint knobs to
  return 429s, time out, or delay responses — useful for testing
  Temporal's retry behaviour.
- **Persistence**: optional file-backed store (BoltDB / sqlite) for
  long-running fixtures across restarts.
- **Webhook scheduling**: per-tenant cron-style scheduler so the mock can
  autonomously emit events without `_admin/trigger-webhook` calls.

Until then, this binary is **deliberately simple** — its only job is to
unblock end-to-end tests and demos.
