# Bitstamp ↔ Formance Payments — field mapping

Authoritative reference for how the dedicated Bitstamp EE connector translates Bitstamp REST API v2 responses into Formance PSP types.

This document is the source of truth for connector reviewers, integrators, and operators tracing a Formance Payment / Order / Conversion back to its Bitstamp origin. It is co-located with the code so the mapping table and the implementation do not drift.

Ticket: **EN-1015**. See [PR #679](https://github.com/formancehq/payments/pull/679) for the original connector and its review thread; this document codifies the conventions that emerged from that review plus the orders + conversions additions.

> Symbols used in this doc
>
> - **B** — a Bitstamp REST v2 field (JSON, `snake_case`)
> - **F** — a Formance PSP field (Go struct, `CamelCase`)
> - `→` — sync direction (Bitstamp → Formance)

---

## 1. Connector configuration

Defined in [`config.go`](config.go), exposed through [`/openapi/v3/v3-connectors-config.yaml`](../../../openapi/v3/v3-connectors-config.yaml) as `V3BitstampConfig`.

| Field | Required | Default | Purpose |
|---|---|---|---|
| `apiKey` | yes | — | Bitstamp REST v2 API key. Used in the `X-Auth` header and HMAC string-to-sign. |
| `apiSecret` | yes | — | HMAC-SHA256 signing secret. Never logged. |
| `endpoint` | no | `https://www.bitstamp.net` | API root. Override only when running against a non-production environment. Bitstamp does not publish a public sandbox; rely on a low-funded production account for QA. |
| `pollingPeriod` | no | `30m` (min `20m`) | Periodic sync cadence for accounts / payments / orders / conversions. |

### 1.1 Authentication

Bitstamp REST v2 uses HMAC-SHA256 with five headers:

| Header | Value |
|---|---|
| `X-Auth` | `BITSTAMP <apiKey>` |
| `X-Auth-Signature` | `hex(HMAC-SHA256(apiSecret, message))` |
| `X-Auth-Nonce` | UUID v4 |
| `X-Auth-Timestamp` | `time.Now().UnixMilli()` as decimal string |
| `X-Auth-Version` | `v2` |

The message-to-sign is the raw concatenation (no separators):
`BITSTAMP <apiKey><method><host><path><query><contentType><nonce><timestamp>v2<body>`.

`<contentType>` is included only when the request has a body (it's set to `application/x-www-form-urlencoded` for the form-encoded POSTs Bitstamp uses).

### 1.2 Concurrency & immutability

`Plugin` (in [`plugin.go`](plugin.go)) holds the `client`, `config`, and a mutex-protected currencies cache with a 24h TTL (`ensureCurrencies`). The `client` and the underlying `httpwrapper.Client` are stateless beyond the HTTP connection pool. The engine may invoke any capability concurrently across worker goroutines without external synchronisation.

Pagination state (`paymentsState`, `ordersState`, `conversionsState` in [`state.go`](state.go)) lives entirely in the engine-managed checkpoint passed via `req.State` and returned via `resp.NewState`. A worker crash mid-cycle resumes deterministically from the last persisted checkpoint.

---

## 2. Capabilities and workflow

Declared in [`capabilities.go`](capabilities.go) and [`workflow.go`](workflow.go).

| Capability | Bitstamp endpoint(s) | Triggered by |
|---|---|---|
| `CAPABILITY_FETCH_ACCOUNTS` | `POST /api/v2/account_balances/` | Periodic `TASK_FETCH_ACCOUNTS` |
| `CAPABILITY_FETCH_BALANCES` | none (reads parent account's `Raw`) | `TASK_FETCH_BALANCES` nested under `TASK_FETCH_ACCOUNTS` (not periodic itself) |
| `CAPABILITY_FETCH_PAYMENTS` | `POST /api/v2/user_transactions/` | Periodic `TASK_FETCH_PAYMENTS` (root) |
| `CAPABILITY_FETCH_ORDERS` | `POST /api/v2/open_orders/all/` + `POST /api/v2/order_status/` | Periodic `TASK_FETCH_ORDERS` (root) |
| `CAPABILITY_FETCH_CONVERSIONS` | `POST /api/v2/user_transactions/` (filtered) | Periodic `TASK_FETCH_CONVERSIONS` (root) |

### 2.1 Workflow shape

```
fetch_accounts (periodic)
  └── fetch_balances (FromPayload — no extra API call)

fetch_payments     (periodic root)
fetch_orders       (periodic root)
fetch_conversions  (periodic root)
```

`TASK_FETCH_BALANCES` is **not periodic**; it is fanned out per parent account so the per-account balance is derived from the `PSPAccount.Raw` already returned by the accounts task. This avoids a second `/api/v2/account_balances/` call per cycle. Pattern matches [`internal/connectors/plugins/public/qonto/balances.go`](../../../internal/connectors/plugins/public/qonto/balances.go); aligns with the reviewer guidance on [#679](https://github.com/formancehq/payments/pull/679).

`payments`, `orders`, `conversions` are independent root tasks — none of them require a parent account ID (Bitstamp endpoints are account-global at the API-key level).

### 2.2 Pagination model

| Stream | Cursor | Bitstamp filter |
|---|---|---|
| `user_transactions` (payments) | `since_id` watermark on `tx.ID` | `sort=asc`, `limit=req.PageSize`, optional `since_id` |
| `user_transactions` (conversions) | same | same — filtered to `type=36` two-asset rows in mapper |
| `open_orders` (orders snapshot) | none — full snapshot every cycle | server-side cache: ~10 s |
| `order_status` (orders reconciliation) | none — called per tracked order ID | id sent in body |

`since_id` is the only viable backfill primitive (the `since_timestamp` filter is limited to a 30-day window). When set, Bitstamp implicitly clamps `limit` to 1000.

`since_id` is **inclusive**: the last row of cycle N reappears as the first row of cycle N+1. The framework dedupes by `PSPPayment.Reference` / `PSPConversion.Reference`, so this is wasted bandwidth, not a correctness problem.

End-of-pagination keeps the watermark — never reset (the canonical Coinbase Prime `advanceCursor` mistake from [#707](https://github.com/formancehq/payments/pull/707)).

---

## 3. Sync mappings (Bitstamp → Formance)

### 3.1 `account_balances/` row → `PSPAccount` (internal)

Implemented in [`mappers/account.go`](mappers/account.go) (`AccountBalanceToPSPAccount`).

Bitstamp does not have the notion of distinct "accounts" — every API key sees a single set of per-currency balance rows. The connector treats each currency-with-non-zero-balance as a synthetic `PSPAccount` whose `Reference` is the uppercase ticker. This is the same pattern Coinbase Prime would land at if it didn't have explicit wallet UUIDs.

| F — `models.PSPAccount` | B — `AccountBalance` | Notes |
|---|---|---|
| `Reference` | `currency` (uppercased + trimmed) | Stable per-key, per-currency. The engine namespaces by `ConnectorID`. |
| `CreatedAt` | constant `bitstampGenesis` (`2011-08-02 UTC`) | Bitstamp does not expose per-currency creation dates. The launch-date sentinel is stable across reinstalls — see [Known limitations](#7-known-limitations). |
| `DefaultAsset` | `currency` → `SYMBOL/<precision>` via `currency.FormatAsset` | Set only when the symbol is in the `getCurrencies` map; otherwise `nil`. |
| `Raw` | full `AccountBalance` JSON | Drives the FromPayload-driven balances task in §3.2. |
| `Metadata` | — | Empty for now (no per-currency metadata Bitstamp returns is worth surfacing). |

**Zero-balance filter.** A row is skipped when `Available`, `Total`, **and** `Reserved` are all zero. Rationale: Bitstamp returns every currency the account *could* hold even if never funded; emitting hundreds of empty accounts pollutes the catalogue. If reviewer guidance on #679 (row 3 of the checklist) ultimately prefers "emit everything", drop this filter — it lives in one place in `accounts.go`.

### 3.2 `PSPAccount.Raw` → `PSPBalance`

Implemented in [`mappers/balance.go`](mappers/balance.go) (`AccountBalanceToPSPBalance`).

Re-uses the `AccountBalance` JSON snapshotted in the parent `PSPAccount.Raw` — no extra Bitstamp API call (cf. the duplicate-call issue Quentin called out on [#679](https://github.com/formancehq/payments/pull/679)).

| F — `models.PSPBalance` | B — `AccountBalance` (via `PSPAccount.Raw`) | Notes |
|---|---|---|
| `AccountReference` | `PSPAccount.Reference` | Currency ticker. |
| `Asset` | parent `PSPAccount.DefaultAsset` | Required — unknown currencies are skipped at the accounts step, so balances always have a known asset. |
| `Amount` | `available` parsed at precision | Reserved/total are surfaced via account metadata, not as separate balances. The engine's balance model is single-snapshot per `(account, asset)`. |
| `CreatedAt` | `time.Now().UTC()` | Bitstamp returns no per-row timestamp; we stamp at read time. |

### 3.3 `user_transactions` row → `PSPPayment`

Implemented in [`mappers/payment.go`](mappers/payment.go) (`UserTransactionToPSPPayment`).

Bitstamp `user_transactions` returns settled history for deposits, withdrawals, transfers, staking, and trades. Rows with `tx.type ∈ {2, 36}` (trades / instant buy-sell) are excluded — they are surfaced by the orders + conversions capabilities respectively. Rows with two non-zero currency amounts are excluded by the payments mapper; they belong on the conversions path.

| F — `models.PSPPayment` | B — `UserTransaction` | Notes |
|---|---|---|
| `Reference` | `id` (int64 → string) | Globally unique within the connector. |
| `CreatedAt` | `datetime` parsed with `2006-01-02 15:04:05.000000` | UTC. |
| `Type` | derived from `type` — see §4.1 | Default `PAYMENT_TYPE_OTHER` on unknown codes (logged at Warn). |
| `Amount` | the single non-zero known currency amount, stripped of sign | Per CLAUDE.md, `PSPPayment.Amount` is always positive (`abs`). Withdrawals are signed negative on the wire and inverted here. |
| `Asset` | the symbol of the chosen currency → `SYMBOL/<precision>` | Skips the row if no known-currency amount is non-zero. |
| `Scheme` | constant `PAYMENT_SCHEME_OTHER` | Bitstamp does not surface scheme metadata. |
| `Status` | constant `PAYMENT_STATUS_SUCCEEDED` | `user_transactions` returns settled-only history; there is no pending state. |
| `Metadata` | `com.bitstamp.spec/*` keys — see §5.1 | |
| `Raw` | full transaction JSON (including dynamic currency keys) | |

**Multi-asset rule.** A row with exactly one non-zero known-currency amount is a payment; a row with exactly two is a conversion (handled in §3.5); anything else is logged at Warn and skipped. This is the post-#679 hardening of the original "smelly logic" call-out (row 25 of the checklist).

### 3.4 `open_orders` + `order_status` → `PSPOrder`

Implemented in [`mappers/order.go`](mappers/order.go) (`OrderStatusToPSPOrder`). Orchestrated in [`orders.go`](orders.go) — see §3.4.3 for the lifecycle.

Bitstamp does NOT expose a single "orders since X" endpoint. The connector reconciles `open_orders/all/` snapshots with per-order `order_status/` calls to derive a `PSPOrder` per cycle.

#### 3.4.1 First-sight capture

`order_status` returns `{ status, id, client_order_id, transactions[] }` — it does **not** return the original `price`, `amount`, `type` (BUY/SELL), or `currency_pair`. These come from `open_orders/all/`. The connector captures them on first sight and persists them in `ordersState.TrackedOrders[id]` (see [§6.2](#62-orders-state)).

#### 3.4.2 Status mapping

Bitstamp status enum (from ccxt's `parseOrderStatus`, confirmed against live docs):

| B — `order_status.status` | F — `models.OrderStatus` | Notes |
|---|---|---|
| `In Queue` | `ORDER_STATUS_PENDING` | Pre-matched state. |
| `Open` + `len(transactions) == 0` | `ORDER_STATUS_OPEN` | No fills yet. |
| `Open` + `len(transactions) > 0` | `ORDER_STATUS_PARTIALLY_FILLED` | At least one partial fill. |
| `Finished` | `ORDER_STATUS_FILLED` | Fully filled and closed. |
| `Canceled` | `ORDER_STATUS_CANCELLED` | Hard cancel, possibly after partial fills. |
| `Cancel pending` | `ORDER_STATUS_CANCELLED` | Treated as terminal — Formance has no transient cancelling state. |

**No `ORDER_STATUS_EXPIRED`** — Bitstamp does not emit `Expired`. Treat any unexpected status as `ORDER_STATUS_OPEN` and log Warn (a new status value should never silently coerce to terminal).

#### 3.4.3 Field mapping

| F — `models.PSPOrder` | Source | Notes |
|---|---|---|
| `Reference` | `order_status.id` | |
| `ClientOrderID` | `order_status.client_order_id` | Optional. |
| `CreatedAt` | `trackedOrder.FirstSeenAt` | First time the connector saw this order in `open_orders/`. |
| `Direction` | `trackedOrder.Type` (`0`→BUY / `1`→SELL) | Captured from `open_orders/`. |
| `Type` | constant `ORDER_TYPE_LIMIT` | Bitstamp's `open_orders/` only returns limit orders; market orders never sit in the book. |
| `Status` | per §3.4.2 | |
| `BaseQuantityOrdered` | `trackedOrder.Amount` at base precision | Captured from `open_orders/`. |
| `BaseQuantityFilled` | sum of `transactions[].<base_currency>` at base precision | Self-trade rows (same `tid`) deduplicated before summing. |
| `LimitPrice` | `trackedOrder.Price` at price precision | All Bitstamp `open_orders/` are limit orders. |
| `QuoteAmount` | sum of `transactions[].<quote_currency>` at quote precision | The cash leg of fills so far. |
| `QuoteAsset` | quote ticker → `SYMBOL/<precision>` | E.g. `USD/2`. |
| `AverageFillPrice` | `QuoteAmount * 10^pricePrecision / BaseQuantityFilled` | Set only when `BaseQuantityFilled > 0`. |
| `PriceAsset` | quote ticker → `SYMBOL/<precision>` | Same as `QuoteAsset` for Bitstamp (quote-denominated prices). |
| `Fee` | sum of `transactions[].fee` at quote precision | Bitstamp charges fees in the quote currency. |
| `FeeAsset` | quote ticker → `SYMBOL/<precision>` | |
| `TimeInForce` | constant `TIME_IN_FORCE_GOOD_UNTIL_CANCELLED` | Bitstamp limit orders have no TIF flag; GTC is the default behaviour. |
| `SourceAccountReference` / `DestinationAccountReference` | currency tickers split from `currency_pair` | BUY: source=quote, dest=base; SELL: source=base, dest=quote. |
| `Metadata` | `com.bitstamp.spec/*` keys — see §5.2 | |
| `Raw` | full `order_status` JSON merged with the first-sight `open_orders/` snapshot | Preserves original price/amount that `order_status` strips. |

#### 3.4.4 Lifecycle

Each `FetchNextOrders` cycle (in [`orders.go`](orders.go)):

1. `GetOpenOrders(ctx)` → snapshot of currently-open orders.
2. For each snapshot ID **not** in `trackedOrders` → seed `trackedOrders[id]` with the first-sight `Price/Amount/Type/CurrencyPair`.
3. For every `id` in the **union** of `snapshot ∪ trackedOrders` → call `GetOrderStatus(ctx, id)`.
4. Map to `PSPOrder` per §3.4.3.
5. Drop tracked entries whose `IsFinal()` status is true (FILLED / CANCELLED).
6. Drop tracked entries whose `FirstSeenAt` is older than `orderRetentionMax = 25 days` (5-day safety margin under Bitstamp's 30-day `order_status` retention), with metadata `com.bitstamp.spec/retention_expired = true` on the emitted PSPOrder.

`HasMore` is always `false` — the snapshot is what it is, and `order_status` calls are bounded by `len(tracked)`.

The dedupe surface (`OrderAdjustmentID`) already includes `Status`, `BaseQuantityFilled`, `Fee`, `FeeAsset` — so emitting a `PSPOrder` per cycle with refreshed fills is idempotent without spurious adjustments. This matches the laouji feedback on [#657](https://github.com/formancehq/payments/pull/657).

### 3.5 `user_transactions` row (type 36, two-asset) → `PSPConversion`

Implemented in [`mappers/conversion.go`](mappers/conversion.go) (`UserTransactionToPSPConversion`). Orchestrated in [`conversions.go`](conversions.go).

Bitstamp does not expose a `/api/v2/conversions/` endpoint. Instant buy/sell fills (`POST /api/v2/buy/instant/{pair}/`, `POST /api/v2/sell/instant/{pair}/`) are write-only; the resulting fills surface in `user_transactions` as `type=36` rows with **two** non-zero currency amounts (the base/quote swap) plus an extra dynamic key like `usdc_eur` carrying the rate.

The conversions task **shares the same `user_transactions` stream and `since_id` cursor** as payments, but uses a distinct `conversionsState` so the two cursors advance independently.

| F — `models.PSPConversion` | B — `UserTransaction` | Notes |
|---|---|---|
| `Reference` | `id` (int64 → string) | |
| `CreatedAt` | `datetime` | |
| `SourceAsset` | negative-amount currency → `SYMBOL/<precision>` | The leg the user paid with. |
| `DestinationAsset` | positive-amount currency → `SYMBOL/<precision>` | The leg the user received. |
| `SourceAmount` | abs(negative amount) in minor units | |
| `DestinationAmount` | positive amount in minor units | |
| `Fee` | `fee` in minor units at quote precision | Bitstamp fees are charged in the quote currency. When unknowable, `Fee = 0`, `FeeAsset = nil`. |
| `FeeAsset` | quote ticker → `SYMBOL/<precision>` | |
| `Status` | constant `CONVERSION_STATUS_COMPLETED` | `user_transactions` returns settled-only. |
| `SourceAccountReference` / `DestinationAccountReference` | currency tickers | Mirror PSPAccount references. |
| `Metadata` | `com.bitstamp.spec/*` — see §5.3 | Includes the rate (e.g. `0.86047` from `usdc_eur`). |
| `Raw` | full transaction JSON | |

**Detection rule** (in `UserTransactionToPSPConversion`):

```
tx.Type == "36"                            // instant buy/sell
&& len(twoAssetRow.knownCurrencies) == 2  // exactly two non-zero known currencies
```

Any non-matching row returns `(nil, nil)` — the orchestrator skips it without erroring.

**Regression fixture** (locked in [`mappers/conversion_test.go`](mappers/conversion_test.go), traceable to Quentin's [#679](https://github.com/formancehq/payments/pull/679) example):

```json
{
  "id": 458254264,
  "datetime": "2025-09-25 14:42:59.894846",
  "type": "36",
  "fee": "0.000000",
  "eur": "-5.00",
  "usdc": "5.810770",
  "usdc_eur": 0.86047,
  "usd": 0.0,
  "btc": 0.0
}
```

→ `EUR → USDC`, `SourceAmount = 500` (EUR minor units, precision 2), `DestinationAmount = 5810770` (USDC, precision 6), `Metadata["com.bitstamp.spec/rate"] = "0.86047"`.

---

## 4. Status / type / direction enum mapping

### 4.1 `user_transactions.type` → `models.PaymentType`

Implemented in [`mappers/status.go`](mappers/status.go) as a table-driven function so adding a new type code is a 2-line diff.

| Bitstamp code | Constant | F — `PaymentType` | Notes |
|---|---|---|---|
| `0` | `txTypeDeposit` | `PAYMENT_TYPE_PAYIN` | |
| `1` | `txTypeWithdrawal` | `PAYMENT_TYPE_PAYOUT` | |
| `2` | `txTypeMarketTrade` | **skipped** | Surfaced via `PSPOrder.transactions[]`. |
| `14` | `txTypeSubAccountTransfer` | `PAYMENT_TYPE_TRANSFER` | |
| `25` | `txTypeStakingCredit` | `PAYMENT_TYPE_TRANSFER` | Internal movement to staking wallet. |
| `26` | `txTypeStakingSent` | `PAYMENT_TYPE_TRANSFER` | Inverse of staking_credit. |
| `27` | `txTypeStakingReward` | `PAYMENT_TYPE_PAYIN` | Yield / reward. |
| `32` | `txTypeReferralReward` | `PAYMENT_TYPE_PAYIN` | |
| `33` | `txTypeSettlementTransfer` | `PAYMENT_TYPE_TRANSFER` | Undocumented but observed in production. |
| `35` | `txTypeInterAccountTransfer` | `PAYMENT_TYPE_TRANSFER` | |
| `36` | `txTypeBuySell` | **skipped** | Conversions; mapped to `PSPConversion`. |
| anything else | — | `PAYMENT_TYPE_OTHER` | Logged at Warn with `tx.id`. |

### 4.2 `open_orders.type` → `models.OrderDirection`

| Bitstamp | F |
|---|---|
| `"0"` | `ORDER_DIRECTION_BUY` |
| `"1"` | `ORDER_DIRECTION_SELL` |
| anything else | error (validation fails) |

### 4.3 `order_status.status` → `models.OrderStatus`

See §3.4.2.

### 4.4 Conversion status

Always `CONVERSION_STATUS_COMPLETED` (no other state surfaces in `user_transactions`).

---

## 5. Metadata keys

All keys are namespaced under `MetadataPrefix = "com.bitstamp.spec/"` (defined in [`mappers/metadata.go`](mappers/metadata.go)).

### 5.1 Payments

| Key | Source | Always present? |
|---|---|---|
| `com.bitstamp.spec/type` | `tx.type` (string) | yes |
| `com.bitstamp.spec/fee` | `tx.fee` (string) | only when non-zero |
| `com.bitstamp.spec/order_id` | `tx.order_id` (string) | only when present on the raw row |

### 5.2 Orders

| Key | Source |
|---|---|
| `com.bitstamp.spec/currency_pair` | from `trackedOrder.CurrencyPair`, lowercased |
| `com.bitstamp.spec/order_type` | always `"limit"` (Bitstamp `open_orders/` returns limit-only) |
| `com.bitstamp.spec/client_order_id` | only when non-empty |
| `com.bitstamp.spec/retention_expired` | `"true"` only on the final emit forced by the 25-day eviction policy |

### 5.3 Conversions

| Key | Source |
|---|---|
| `com.bitstamp.spec/type` | always `"36"` (per detection rule) |
| `com.bitstamp.spec/rate` | the dynamic `<src>_<dst>` key value (e.g. `usdc_eur` = `0.86047`) |
| `com.bitstamp.spec/currency_pair` | derived as `<src>_<dst>` (lowercase) |
| `com.bitstamp.spec/fee` | only when non-zero |

---

## 6. State (cursors / watermarks)

Defined in [`state.go`](state.go). All persisted as opaque JSON via `req.State` / `resp.NewState`.

### 6.1 Payments

```go
type paymentsState struct {
    LastTransactionID int64 `json:"lastTransactionID"`
}
```

Advances by `max(tx.ID)` seen in the page. End-of-pagination preserves the previous max — never resets to zero.

### 6.2 Orders

```go
type ordersState struct {
    LastTransactionID int64                   `json:"lastTransactionID"`
    TrackedOrders     map[string]trackedOrder `json:"trackedOrders"`
}

type trackedOrder struct {
    LastStatus   string    `json:"lastStatus"`
    FirstSeenAt  time.Time `json:"firstSeenAt"`
    Price        string    `json:"price"`        // captured from open_orders
    Amount       string    `json:"amount"`       // captured from open_orders
    CurrencyPair string    `json:"currencyPair"` // e.g. "btcusd"
    Type         int       `json:"type"`         // 0 = buy, 1 = sell
}
```

`LastTransactionID` is reserved for future `user_transactions`-driven fill aggregation; current implementation derives fills from `order_status.transactions[]`, but the field is present so the state migration is non-breaking when the design extends.

### 6.3 Conversions

```go
type conversionsState struct {
    LastTransactionID int64 `json:"lastTransactionID"`
}
```

Independent watermark from `paymentsState` — the two tasks scan the same `user_transactions` stream but at potentially different cadences.

---

## 7. Known limitations

| Limitation | Rationale | Workaround |
|---|---|---|
| `bitstampGenesis` sentinel `2011-08-02` as account `CreatedAt` | Bitstamp's `account_balances` does not expose per-currency creation dates. Using `time.Now()` would make accounts look "new" on every reinstall. | The sentinel is stable across reinstalls and idempotent. Downstream readers should treat it as "creation date unknown, definitely before this". |
| Sub-accounts (`X-Auth-Subaccount-Id`) out of scope | Single-account-per-connector keeps the scope tight. | Install one connector per Bitstamp sub-account; each carries its own credentials. |
| No historical backfill beyond `since_id` | Bitstamp's `since_timestamp` is limited to a 30-day window. | Cold start scans from the earliest available `since_id`. Historical data older than the API allows is unreachable by design. |
| `order_status` retention ~30 days | Tracked orders inactive for >30 days become unfetchable. | The connector force-emits a final `PSPOrder` at 25 days (5-day safety margin) with `com.bitstamp.spec/retention_expired = true`, then drops the entry. |
| `order_status` does not return original price/amount | The endpoint only carries `status`, `id`, `client_order_id`, `transactions[]`. | Captured on first sight from `open_orders/` and persisted in `trackedOrder` (§6.2). |
| Derivatives surface ignored | Spot-only stance (see §8). | `API5506` errors gracefully skip; rows with `margin_mode` / `leverage_rate` log Warn and are skipped. |
| Payments status always `SUCCEEDED` | `user_transactions` returns settled-only history. | Pending withdrawals are tracked via separate Bitstamp endpoints (`/withdrawal-requests/`) which the connector does not poll today. |
| Multi-asset payments rows silently skipped if neither one nor exactly two non-zero known currencies | Defensive: matches neither `payment` nor `conversion`. | Log line emitted at Warn; row added to `tx.id` is preserved in `Raw` if surfaced upstream. |

---

## 8. Spot-only stance — derivatives handling

Bitstamp ships derivatives (perpetual futures, margin trading) on the same REST host as spot, provided by a separate regulated legal entity (Bitstamp Financial Services Ltd., Slovenia / MiFID). The connector is **spot-only** for this release.

Concrete handling (matches §13 of [the formance-connector skill](https://github.com/formancehq/internal-skills/blob/c4853ff475f4ff027ed6ea44fe655f6b78feb317/skills/formance-connector/SKILL.md)):

- `client/error.go` recognises Bitstamp error code `API5506: "Trade account does not support derivatives."` and the client returns an empty result + an Info log rather than propagating it as a failure.
- Any `user_transactions` or `order_status` response carrying `margin_mode`, `leverage_rate`, or other derivatives-specific markers triggers a Warn log with the row ID and is skipped at the mapper level.
- A derivatives-enabled customer installing this connector will see those Warn logs as the signal that derivatives support is needed; the connector does not silently mis-classify their activity as spot.

---

## 9. Future extensions

Other Bitstamp API surfaces that exist today but are explicitly out of scope for this connector iteration. Listed so the next contributor knows what is plausible vs new work:

| Surface | URL | Rationale for exclusion |
|---|---|---|
| WebSocket API v2 | `wss://ws.bitstamp.net/` | Event-driven sync requires a different connector pattern (persistent subscription, not periodic polling). |
| FIX v2 Gateway | https://www.bitstamp.net/fix/v2/ | FIX session is a separate integration surface; would warrant a dedicated `bitstamp-fix` connector. |
| Derivatives (margin / perps) | same REST host, gated by account type | Distinct asset model (positions, leverage, margin) — needs a derivatives-aware mapper layer. |
| OTC RFQ | https://www.bitstamp.net/over-the-counter-otc-services/ | API endpoints non-public; gated behind institutional accounts. |
| Open Banking PSD2 | https://www.bitstamp.net/api-psd2/ | Regulatory TPP API; not a payment-connector use case. |
| Sub-accounts | `x_auth_subaccount_id` header | One sub-account per connector instance is sufficient; multi-sub-account fan-out is a separate scope. |

---

## 10. Open questions deferred

These are tracked here so reviewers and the next contributor know what was deliberately deferred rather than overlooked:

- **Self-trade fill semantics.** `user_transactions` may carry `self_trade: true` / `self_trade_order_id` markers on type-2 fills; these are dedupe-pairs and must not be double-counted when aggregating `order_status.transactions[]`. The connector dedupes by `tid` in `mappers/order.go`, but the live data shape needs validation against a real self-trade fixture.
- **Fee currency on type-36 rows.** ccxt's `parseTrade()` assumes quote-side; needs validation against a live row before locking the mapping. Until then, the conversion mapper derives the quote currency from the dynamic `<src>_<dst>` rate key.
- **Complete type enum.** Codes `33` (`settlement_transfer`) and `36` (`buy/sell` instant) are inferred from ccxt comments and observed production data. The connector logs Warn on any unknown type with the row ID so a future contributor can surface and document new codes.

---

## 11. References

- [Bitstamp REST API v2 docs](https://www.bitstamp.net/api/) — canonical endpoint reference.
- [ccxt Bitstamp source](https://github.com/ccxt/ccxt/blob/master/ts/src/bitstamp.ts) — third-party reference implementation used for cross-checking response shapes and status enums.
- [Formance connector skill](https://github.com/formancehq/internal-skills/blob/c4853ff475f4ff027ed6ea44fe655f6b78feb317/skills/formance-connector/SKILL.md) — phase-by-phase scaffolding workflow.
- [PR #679](https://github.com/formancehq/payments/pull/679) — original Bitstamp connector PR and review thread.
- [PR #657](https://github.com/formancehq/payments/pull/657) — orders + conversions platform on Coinbase Prime; idempotency-with-status convention.
- [PR #602](https://github.com/formancehq/payments/pull/602) — drop user-configurable `pageSize`.
- [PR #707](https://github.com/formancehq/payments/pull/707) — Coinbase Prime cursor / workflow fix; canonical "never reset cursor at end of pagination" rule.
- [PR #711](https://github.com/formancehq/payments/pull/711) — Routable EE connector; reference for the `mappers/` subpackage layout.
- [`internal/connectors/plugins/public/qonto/balances.go`](../../../internal/connectors/plugins/public/qonto/balances.go) — canonical FromPayload balances pattern.
