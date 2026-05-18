# Universal Connector — Data Model (v1)

This is the single source of truth for the wire schemas every counterparty
must serialise. Every field is reflected in
[`universal-openapi.yaml`](universal-openapi.yaml); this doc adds the
**semantic** rules that schemas can't enforce, and the mapping from each wire
field to the Formance PSP type the engine actually consumes.

> Companion docs: [`adjustments.md`](adjustments.md),
> [`state-machines.md`](state-machines.md),
> [`universal-events.md`](universal-events.md).

## Cross-cutting conventions

| Concern        | Rule                                                                                                                                                                                       |
|----------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Amounts        | Decimal string of integer **minor units**. `"1234"` at `asset:"USD/2"` means 12.34 USD. **No floats anywhere**, even for analytics fields.                                                 |
| Asset (UMN)    | `"<symbol>/<precision>"`. Examples: `"USD/2"`, `"EUR/2"`, `"BTC/8"`, `"USDC/6"`. Precision is the number of minor-unit digits.                                                             |
| References     | Raw counterparty IDs. The engine namespaces by `ConnectorID`. Do **not** prefix or otherwise transform.                                                                                    |
| Timestamps     | RFC3339 UTC. Example: `"2026-05-13T12:34:56.789Z"`.                                                                                                                                        |
| Pagination     | Choose one mode at install time, advertised in `features.pagination`: `"cursor"` (return `nextCursor`), `"page"` (echo a 1-based `page` query parameter), or `"none"` (return everything). |
| Idempotency    | Header on every mutating `POST`. Default name `Idempotency-Key`; override via `features.idempotencyHeader`. Counterparty MUST dedup on the key, never on the request body's `reference`.   |
| Metadata       | Free-form `string -> string` map on every entity. The engine forwards as-is and surfaces it under the entity's metadata in `/v3/...`.                                                      |
| `updatedAt`    | MUST be **strictly monotonic per record**. The engine uses `updatedAtFrom` to incrementally pull adjustments; non-monotonic timestamps cause silent missed updates.                        |
| Error envelope | Either legacy `{message, errors[].field/.message}` or RFC 7807 `application/problem+json`. The connector accepts both — see [`client/error.go`](../client/error.go).                        |

## Per-primitive schema reference

For every primitive: required vs optional, what each field maps to in
`internal/models/`, and "basic vs extensive" guidance.

### Account ([`PSPAccount`](../../../../models/accounts.go))

```json
{
  "reference":     "acct_001",
  "createdAt":     "2026-05-13T12:00:00Z",
  "name":          "Operating EUR",
  "defaultAsset":  "EUR/2",
  "metadata":      {"com.example.spec/branch": "FR-PAR-1"}
}
```

| Field          | Required | PSP field                       | Notes                                                                   |
|----------------|----------|---------------------------------|-------------------------------------------------------------------------|
| `reference`    | yes      | `Reference`                     | Raw counterparty ID.                                                    |
| `createdAt`    | yes      | `CreatedAt`                     | Account inception, not the time of fetch.                               |
| `name`         | no       | `Name`                          | Human-readable. Surfaced in `/v3/accounts/{id}` as `name`.              |
| `defaultAsset` | no       | `DefaultAsset`                  | UMN string. Validated by the engine against `assets.IsValid`.           |
| `metadata`     | no       | `Metadata`                      | Surfaces under `metadata` on the engine-side `Account`.                 |

**Basic vs extensive**: serving `reference` and `createdAt` is the minimum
to pass `Validate()`. Add `defaultAsset` to unlock `/v3/balances` aggregates;
add `metadata` to surface custom labels in API consumers.

### External account

Same wire shape as Account; served from `GET /v1/external-accounts`. Maps to
the engine's external-account record (used for transfers/payouts to
beneficiaries the counterparty knows but Formance doesn't own).

### Balance ([`PSPBalance`](../../../../models/balances.go))

```json
{
  "accountReference": "acct_001",
  "createdAt":        "2026-05-13T12:34:56Z",
  "amount":           "1234567",
  "asset":            "EUR/2"
}
```

| Field              | Required | PSP field          | Notes                                                          |
|--------------------|----------|--------------------|----------------------------------------------------------------|
| `accountReference` | yes      | `AccountReference` | Must be a valid `Account.reference` already known to Formance. |
| `createdAt`        | yes      | `CreatedAt`        | "As of" time of the balance snapshot.                          |
| `amount`           | yes      | `Amount`           | Decimal-string minor units.                                    |
| `asset`            | yes      | `Asset`            | UMN string.                                                    |

Balances are **latest-wins**; no adjustment history.

### Payment ([`PSPPayment`](../../../../models/payments.go))

```json
{
  "reference":                   "pay_42",
  "parentReference":             "pay_41",
  "createdAt":                   "2026-05-13T12:00:00Z",
  "updatedAt":                   "2026-05-13T12:00:05Z",
  "type":                        "PAYIN",
  "status":                      "SUCCEEDED",
  "scheme":                      "SEPA",
  "amount":                      "12345",
  "asset":                       "EUR/2",
  "sourceAccountReference":      "acct_ext_99",
  "destinationAccountReference": "acct_001",
  "metadata":                    {"com.example.spec/clientRef": "INV-2026-0001"}
}
```

Type / Status / Scheme enums must be the canonical strings from
[`internal/models/payment_status.go`](../../../../models/payment_status.go),
[`payment_type.go`](../../../../models/payment_type.go), and
[`payment_scheme.go`](../../../../models/payment_scheme.go). Unknown values
map to `OTHER` — never an error — so adding a new vendor-specific status
won't break ingest, but won't surface meaningfully either.

`parentReference` links refunds, captures, and adjustments back to the
original payment. The engine uses it to attach related records.

**Basic vs extensive**: minimum is `reference`, `createdAt`, `type`, `status`,
`amount`, `asset`. Adding `scheme`, source/destination references, and
`metadata` unlocks the full Payments V3 API surface (links to accounts,
proper grouping in dashboards, scheme-aware filtering).

See [`adjustments.md`](adjustments.md) for how `updatedAt` + `status` mutations
turn into a `PaymentAdjustment` history.

### Order ([`PSPOrder`](../../../../models/orders.go))

```json
{
  "reference":           "ord_007",
  "clientOrderID":       "fmnce-1d4a-ord_007",
  "createdAt":           "2026-05-13T12:00:00Z",
  "updatedAt":           "2026-05-13T12:00:05Z",
  "direction":           "BUY",
  "type":                "LIMIT",
  "status":              "PARTIALLY_FILLED",
  "sourceAsset":         "EUR/2",
  "destinationAsset":    "BTC/8",
  "baseQuantityOrdered": "100000000",
  "baseQuantityFilled":  "75000000",
  "limitPrice":          "8500000",
  "timeInForce":         "GTC",
  "quoteAmount":         "63750000",
  "quoteAsset":          "EUR/2",
  "fee":                 "12500",
  "feeAsset":            "EUR/2",
  "averageFillPrice":    "8500000",
  "priceAsset":          "EUR/2",
  "sourceAccountReference":      "acct_eur",
  "destinationAccountReference": "acct_btc"
}
```

The order direction defines what `sourceAsset` and `destinationAsset` mean:

| Direction | sourceAsset      | destinationAsset |
|-----------|------------------|------------------|
| `BUY`     | quote (e.g. EUR) | base  (e.g. BTC) |
| `SELL`    | base  (e.g. BTC) | quote (e.g. EUR) |

`baseQuantityOrdered` / `baseQuantityFilled` are always in **base-asset minor
units**. `quoteAmount` is in **quote-asset minor units**. The engine treats
each (status, baseQuantityFilled, fee, feeAsset) tuple as a distinct
adjustment — see [`adjustments.md`](adjustments.md).

**Basic vs extensive**: minimum to satisfy `Validate()` is `reference`,
`createdAt`, `direction`, `type`, `status`, `sourceAsset`, `destinationAsset`,
`baseQuantityOrdered`. Add `quoteAmount`/`quoteAsset` for the engine's
analytics; add `priceAsset` for fill-price aggregates; add the `…AccountReference`
fields to link orders into the rest of the accounts graph.

> **`timeInForce` is effectively required.** The engine's
> `orders.time_in_force` storage column is `NOT NULL` (see
> [`internal/storage/orders.go`](../../../../storage/orders.go)) — an
> empty / missing value maps to `TIME_IN_FORCE_UNKNOWN`, whose
> `Value()` rejects with a SQL error and the row is dropped. For
> `MARKET` orders use `IOC` (immediate-or-cancel); for `LIMIT` use
> `GTC`. The plugin does not default this on your behalf — counterparty
> implementers must always emit one of `GTC | IOC | FOK | GTD`.

### Conversion ([`PSPConversion`](../../../../models/conversions.go))

```json
{
  "reference":         "conv_99",
  "createdAt":         "2026-05-13T12:00:00Z",
  "status":            "COMPLETED",
  "sourceAsset":       "USDC/6",
  "destinationAsset":  "USD/2",
  "sourceAmount":      "10000000",
  "destinationAmount": "1000000",
  "fee":               "2500",
  "feeAsset":          "USD/2",
  "sourceAccountReference":      "acct_usdc",
  "destinationAccountReference": "acct_usd"
}
```

Status enum from [`conversion_status.go`](../../../../models/conversion_status.go):
`PENDING`, `COMPLETED`, `FAILED`. Conversions are latest-wins (no adjustment
history); the engine refetches the same record on every poll until terminal.

### Other ([`PSPOther`](../../../../models/others.go))

```json
{
  "id":   "any-stable-key",
  "data": { "anything": "the counterparty wants" }
}
```

The engine forwards `data` untouched into `PSPOther.Other` (a
`json.RawMessage`). Used as an escape hatch for counterparty-specific
records the engine doesn't have a typed shape for.

### Payout / Transfer initiation ([`PSPPaymentInitiation`](../../../../models/payment_initiations.go))

Request body for `POST /v1/payouts` and `POST /v1/transfers`:

```json
{
  "reference":                   "init_2026_0042",
  "description":                 "Operational rebalance",
  "amount":                      "100000",
  "asset":                       "EUR/2",
  "sourceAccountReference":      "acct_001",
  "destinationAccountReference": "acct_ext_99",
  "metadata":                    {"com.example.spec/initiator": "treasury"}
}
```

Counterparty MUST dedup on the request's `Idempotency-Key` header — `reference`
is the engine's initiation reference (used in adjustment dedup) but not
guaranteed unique on retry.

Response body:

```json
{
  "mode":      "polling",
  "pollingID": "ext_payout_xyz"
}
```

or

```json
{
  "mode":    "terminal",
  "payment": { "...full Payment object..." }
}
```

`mode == "polling"` triggers Temporal to call `GET /v1/payouts/{pollingID}`
(or `GET /v1/transfers/{pollingID}`) until a terminal payment is returned or
the response sets `error`. See [`state-machines.md`](state-machines.md).

### Bank account creation

Request body for `POST /v1/bank-accounts`:

```json
{
  "id":            "f0123abc-…uuid…",
  "createdAt":     "2026-05-13T12:00:00Z",
  "name":          "Treasury EUR",
  "iban":          "FR7630006000011234567890189",
  "swiftBicCode":  "BNPAFRPP",
  "country":       "FR",
  "metadata":      {"com.universal.spec/owner.email": "ops@example.com"}
}
```

Response body returns the counterparty-side `Account` representation that
the engine should adopt as the related account (for subsequent transfers /
payouts). The engine then stores both the `BankAccount` aggregate and the
`Account` link.

### Webhook subscription

Request body for `POST /v1/webhooks`:

```json
{
  "name":        "payment.updated",
  "callbackUrl": "https://payments.formance.example.com/webhooks/connector_xyz/payment/updated"
}
```

Response body:

```json
{ "id": "sub_abc", "name": "payment.updated" }
```

See [`universal-events.md`](universal-events.md) for the full event catalogue
and signing rules.

### Real-time stream (optional)

A counterparty advertising `features.eventStream == "wss"` MUST also
expose `GET /v1/stream` (WebSocket upgrade, subprotocol
`formance-universal-v1`). Each frame is the same `WebhookEvent`
envelope as the HTTP webhook body — so the engine pipeline is byte-for-
byte identical regardless of transport. The hello handshake is signed
with the connector's `webhookSharedSecret` (same secret as HTTP webhook
HMAC) over `<timestamp>.<nonce>.<canonicalEventsJSON>`; the
counterparty rejects timestamps outside ±5min skew, nonces seen in the
last 10min, or bad signatures with WS close 1008. See
[`webhooks.md`](webhooks.md) "WebSocket transport" for the full
contract and counterparty obligations (nonce cache, connect rate limit,
stable event ids for cross-pod dedup).
