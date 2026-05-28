# Bitstamp ↔ Formance Payments — field mapping

Authoritative reference for how the Bitstamp EE connector translates Bitstamp REST v2 responses into Formance PSP types. Co-located with the code so the mapping and the implementation cannot drift.

| Symbol | Meaning |
|---|---|
| **B** | Bitstamp REST v2 field (JSON, `snake_case`) |
| **F** | Formance PSP field (Go struct, `CamelCase`) |
| `→` | sync direction (Bitstamp → Formance) |

---

## 1. Overview

The connector is **read-only spot-only**, one install per Bitstamp account scope (Main or one named sub-account — Bitstamp API keys are scoped to a single account; there is no portable fan-out). It surfaces five capabilities:

| F — Capability | Bitstamp endpoints | Scope |
|---|---|---|
| `CAPABILITY_FETCH_ACCOUNTS` | `POST /api/v2/account_balances/` | Main + sub |
| `CAPABILITY_FETCH_BALANCES` | derived from `PSPAccount.Raw` (no extra call) | derived |
| `CAPABILITY_FETCH_PAYMENTS` | `user_transactions/` + `crypto-transactions/` (Main only) + `withdrawal-requests/` | Main + sub (crypto-tx Main only) |
| `CAPABILITY_FETCH_ORDERS` | `open_orders/all/` + `order_status/` | Main + sub |
| `CAPABILITY_FETCH_CONVERSIONS` | `user_transactions/` filtered to `type=36` | Main + sub |

---

## 2. Configuration

Defined in [`config.go`](config.go). OpenAPI v3: `V3BitstampConfig`.

| Field | Required | Default | Purpose |
|---|---|---|---|
| `apiKey` | yes | — | Used in `X-Auth` header + HMAC signature |
| `apiSecret` | yes | — | HMAC-SHA256 signing secret. Never logged. |
| `endpoint` | no | `https://www.bitstamp.net` | Override only for non-production environments |
| `pollingPeriod` | no | `30m` (min `20m`) | Periodic sync cadence |

### Auth — HMAC-SHA256 v2

Five headers per request:

| Header | Value |
|---|---|
| `X-Auth` | `BITSTAMP <apiKey>` |
| `X-Auth-Signature` | `hex(HMAC-SHA256(apiSecret, message))` |
| `X-Auth-Nonce` | UUID v4 |
| `X-Auth-Timestamp` | `time.Now().UnixMilli()` decimal string |
| `X-Auth-Version` | `v2` |

Message-to-sign is the raw concatenation (no separators):
`BITSTAMP <apiKey><method><host><path><query><contentType><nonce><timestamp>v2<body>`.
`<contentType>` is included only when the request carries a body.

### Scope decisions (no flags)

The config is deliberately minimal — no `accountScope`, `enableDerivatives`, `subAccountId`, or per-source toggles. The PSP is the source of truth: the API key already scopes the connection; permissions are detected at the response layer via the try-and-skip cache (§5). Adding scope flags would let a misconfigured install claim to be Main while authenticating as a sub-account.

---

## 3. Workflow & pagination

Declared in [`workflow.go`](workflow.go). Four periodic root tasks:

```text
fetch_accounts (periodic)
  └── fetch_balances (FromPayload — no extra API call)

fetch_payments     (periodic root)
fetch_orders       (periodic root)
fetch_conversions  (periodic root)
```

`fetch_balances` is non-periodic and fanned out per parent account: the balance comes from `PSPAccount.Raw` already returned by accounts, eliminating a second `/account_balances/` call per cycle (Qonto pattern). `payments`, `orders`, `conversions` are independent roots — Bitstamp endpoints are account-global at the API-key level, so no parent context is needed.

### Pagination by source

| Stream | Cursor | Bitstamp filter |
|---|---|---|
| `user_transactions/` (payments + conversions) | `since_id` watermark on `tx.ID` (independent per task) | `sort=asc, limit=N, since_id?` |
| `crypto-transactions/` | per-bucket `datetime` Unix-seconds (deposits, withdrawals, ripple IOUs each track separately) | `limit, offset, include_ious=true` |
| `withdrawal-requests/` | `id`-based after cold-start offset walk | `limit + offset` (both required), or `timedelta?` |
| `open_orders/all/` (orders snapshot) | none — full snapshot every cycle (server-cached ~10 s) | — |
| `order_status/` (per tracked id) | none — called per id | `id` in body |

**Invariants.** `since_id` is inclusive (the last row of cycle N reappears as the first of N+1; the framework dedupes by `PSPPayment.Reference`). End-of-pagination keeps the watermark — never reset (see §5 advanceCursor). `since_timestamp` / `until_timestamp` on `user_transactions/` and `crypto-transactions/` are bounded to the last 30 days; watermark cursors avoid this entirely.

---

## 4. Resource mappings

### 4.1 PSPAccount — `account_balances/` row → `PSPAccount`

Implemented in [`mappers/account.go`](mappers/account.go). One PSPAccount per `(connector install, currency)`. Multi-scope orgs install one connector per API key.

| F — `models.PSPAccount` | B — `AccountBalance` | Notes |
|---|---|---|
| `Reference` | `currency` (uppercased) | Stable per-key, per-currency. Namespaced by `ConnectorID`. |
| `Name` | same as `Reference` (currency ticker) | Connector-level name (e.g. `bitstamp-main`) disambiguates the account scope; the per-currency `Name` only needs the ticker. |
| `CreatedAt` | `BitstampGenesis` = `2011-08-02 UTC` | Bitstamp does not expose per-currency creation dates. The launch-date sentinel is stable across reinstalls (§7). |
| `DefaultAsset` | `currency` → `SYMBOL/<precision>` | Set only when the symbol is in the currencies cache. |
| `Raw` | full `AccountBalance` JSON | Drives the FromPayload-driven balances task. |
| `Metadata` | enrichment keys (§5.5 + §6.1) | Populated only when the install-time enrichment cache has data for this currency. |

**Zero-balance filter.** Rows with all of `Available`, `Total`, `Reserved` zero are skipped at the orchestrator. Bitstamp returns every currency the account *could* hold; emitting hundreds of empty accounts pollutes the catalogue.

### 4.2 PSPBalance — `PSPAccount.Raw` → `PSPBalance`

Implemented in [`mappers/balance.go`](mappers/balance.go). Re-uses the snapshot in the parent's `Raw` field; no extra API call.

| F — `models.PSPBalance` | B — `AccountBalance` (via parent `Raw`) |
|---|---|
| `AccountReference` | `PSPAccount.Reference` (currency ticker) |
| `Asset` | parent `PSPAccount.DefaultAsset` |
| `Amount` | `available` at currency precision |
| `CreatedAt` | orchestrator's `time.Now().UTC()` |

`reserved` and `total` are kept in `PSPAccount.Raw` for downstream auditing; the engine's balance model is single-snapshot per `(account, asset)`.

### 4.3 PSPPayment — three-source union

A Formance PSPPayment originates from one of three Bitstamp sources, fanned in by the `fetch_payments` orchestrator. Per-row dedupe across sources is by `(source, id)` — each source uses its own ID space. The `com.bitstamp.spec/source` metadata is mandatory on every emission.

#### 4.3.1 `user_transactions/` row → `PSPPayment` (settled history)

Implemented in [`mappers/payment.go`](mappers/payment.go). Rows with `tx.type ∈ {2, 36}` (trades / instant buy-sell) are excluded — they feed orders + conversions respectively.

| F — `models.PSPPayment` | B — `UserTransaction` | Notes |
|---|---|---|
| `Reference` | `id` (int64 → string) | Bare numeric, no prefix. |
| `CreatedAt` | `datetime` via `BitstampDatetimeLayout` (microseconds) | UTC. |
| `Type` | derived from `type` per §4.3.4 | Default `PAYMENT_TYPE_OTHER` on unknown codes (Info-logged). |
| `Amount` | the single non-zero known currency amount, `abs`-stripped | `PSPPayment.Amount` is always positive. |
| `Asset` | chosen currency → `SYMBOL/<precision>` | Skips the row if no known-currency amount is non-zero. |
| `Scheme` | `PAYMENT_SCHEME_OTHER` | Bitstamp surfaces no scheme metadata on this stream. |
| `Status` | `PAYMENT_STATUS_SUCCEEDED` | `user_transactions/` returns settled-only. |
| `Metadata` | see §6.1 | Always carries `source="user_transactions"`. |
| `Raw` | full transaction JSON | Includes dynamic currency keys. |

**Multi-asset rule.** Exactly one non-zero known-currency amount → payment; exactly two → conversion (§4.5). Anything else (zero, or 3+) is Info-logged and skipped.

#### 4.3.2 `crypto-transactions/` row → `PSPPayment` (on-chain crypto)

**Main-account only.** Sub-account scopes hit the try-and-skip cache. Response carries three arrays: `deposits` (with PENDING / COMPLETED status + `pending_reason`), `withdrawals`, `ripple_iou_transactions`.

| F — `models.PSPPayment` | B — `crypto-transactions` entry | Notes |
|---|---|---|
| `Reference` | deposits: `ct-dep:<id>` / withdrawals: `ct-wd:<txid>` / IOUs: `ct-iou:<txid>` | Source-prefixed; withdrawals + IOUs have no top-level id on the wire. |
| `CreatedAt` | `datetime` (Unix seconds) → `time.Unix(_, 0).UTC()` | Endpoint serialises as int (NOT the `user_transactions/` string layout). |
| `Type` | deposits → `PAYIN`; withdrawals + IOUs → `PAYOUT` | |
| `Amount` | `amount` (json.Number) at `currencies[currency].decimals` | |
| `Asset` | `currency` → `SYMBOL/<precision>` | |
| `Scheme` | `PAYMENT_SCHEME_OTHER` | |
| `Status` | deposits: `PENDING` / `COMPLETED` per §4.3.6. Withdrawals + IOUs: `SUCCEEDED` (no status field — endpoint only lists processed rows). | |
| `Metadata` | see §6.1 — `source="crypto_transactions"` + `network` + `txid` + `destination_address` + `pending_reason?` | |
| `Raw` | full source-array entry | |

**Pagination.** Per-bucket Unix-seconds watermarks on `cryptoTransactionsState{DepositsSinceTs, WithdrawalsSinceTs, RipplesSinceTs}`. Each cycle: call with `limit=1000, include_ious=true`; advance each bucket to the max `datetime` seen.

#### 4.3.3 `withdrawal-requests/` row → `PSPPayment` (fiat withdrawal lifecycle)

Available on Main + sub.

| F — `models.PSPPayment` | B — `withdrawal-request` | Notes |
|---|---|---|
| `Reference` | `wr:<id>` | |
| `CreatedAt` | `datetime` via `2006-01-02 15:04:05` (no microseconds) | |
| `Type` | `PAYMENT_TYPE_PAYOUT` | |
| `Amount` | `amount` (string) at currency precision | |
| `Asset` | `currency` → `SYMBOL/<precision>` | |
| `Scheme` | `type` int enum → see §4.3.5 | |
| `Status` | `status` int enum → see §4.3.6 | |
| `Metadata` | see §6.1 — `source="withdrawal_requests"` + `bank_transaction_id?` + `network?` + `destination_address?` | |
| `Raw` | full row | |

**Pagination.** Both `limit` AND `offset` are required (passing only `limit` returns `"Both limit and offset must be present."`). State carries `withdrawalRequestsState{LastID int64}`; cold-start walks offsets until empty, then filters `id > LastID` on subsequent pages.

#### 4.3.4 Cross-account transfers (types 14 / 33 / 35)

When the same `tx.id` appears on both sides of a sub-account / settlement / inter-account transfer, the connector emits each leg as a signed PAYOUT (source side, negative amount) or PAYIN (destination side, positive amount), correlated via metadata. Downstream consumers materialise a full transfer by joining `(transfer_pair_id, asset)`.

| F — `models.PSPPayment` | Source leg (negative amount) | Destination leg (positive amount) |
|---|---|---|
| `Type` | `PAYMENT_TYPE_PAYOUT` | `PAYMENT_TYPE_PAYIN` |
| `Amount` | `abs(amount)` | `amount` |
| `SourceAccountReference` | local ticker | `nil` (counterparty in another connector) |
| `DestinationAccountReference` | `nil` | local ticker |
| `Metadata["com.bitstamp.spec/transfer_pair_id"]` | `tx.id` as string — identical on both sides | identical |
| `Metadata["com.bitstamp.spec/transfer_direction"]` | `"outgoing"` | `"incoming"` |
| `Status` | `PAYMENT_STATUS_SUCCEEDED` | identical |

**Why not `PAYMENT_TYPE_TRANSFER`?** The Formance `PSPPayment` model expects a single connector to populate both account references. Each Bitstamp connector sees only one side, so the payout/payin pair is the only model that stays internally consistent. Same convention as interbank wires elsewhere in the platform.

**Live-probed reality.** The probe found that a Main-account API key does NOT surface type-14 / 33 / 35 rows in `user_transactions/` even when the web UI shows transfers — see §8. The mapping above stays as defensive code that activates the moment Bitstamp exposes the rows on any endpoint already polled. Customers needing transfer reconciliation today install one connector per sub-account; the existing pair-id correlation works once both legs' API keys are integrated.

#### 4.3.5 `withdrawal-request.type` → `models.PaymentScheme`

| Bitstamp int | F — PaymentScheme |
|---|---|
| `0` (SEPA) | `PAYMENT_SCHEME_SEPA_CREDIT` |
| `1` (international wire) / `2` (ARDI) / `3` (international BIC) / `4` (crypto) | `PAYMENT_SCHEME_OTHER` — wire integer preserved in `metadata.type` |
| anything else | `PAYMENT_SCHEME_UNKNOWN` + Info |

#### 4.3.6 Status enums

`withdrawal-request.status` (int):

| Code | F — PaymentStatus |
|---|---|
| `0` (open) / `1` (in progress) | `PAYMENT_STATUS_PENDING` |
| `2` (finished) | `PAYMENT_STATUS_SUCCEEDED` |
| `3` (canceled) | `PAYMENT_STATUS_CANCELLED` |
| `4` (failed) | `PAYMENT_STATUS_FAILED` |
| anything else | `PAYMENT_STATUS_UNKNOWN` + Info |

`crypto-transactions.deposits[].status` (string; withdrawals + IOUs have no status):

| Wire | F — PaymentStatus |
|---|---|
| `"PENDING"` | `PAYMENT_STATUS_PENDING` |
| `"COMPLETED"` | `PAYMENT_STATUS_SUCCEEDED` |
| anything else | `PAYMENT_STATUS_UNKNOWN` + Info |

### 4.4 PSPOrder — `open_orders` + `order_status` → `PSPOrder`

Implemented in [`mappers/order.go`](mappers/order.go), orchestrated in [`orders.go`](orders.go).

Bitstamp does not expose an "orders since X" endpoint. The connector snapshots `open_orders/all/`, then polls `order_status/` per tracked id. `order_status/` returns the rich shape — market, type, subtype, datetime, amount_remaining, fills — so the only field requiring first-sight capture from `open_orders/` is the original limit `Price`.

Bitstamp's "Trade" primitive is the per-fill row in `user_transactions` (`type=2`) carrying the parent `order_id`. The current connector emits one PSPOrder per order (not per fill); fills are aggregated under their parent via `order_status.transactions[]`. See §4.4.4 for historical orders that pre-date the install.

#### 4.4.1 Lifecycle

Each `FetchNextOrders` cycle:

1. `GetOpenOrders(ctx)` → snapshot of currently-open orders.
2. For each snapshot ID not in `trackedOrders` → seed `{LastStatus: Open, FirstSeenAt: now, LimitPrice}`.
3. For every id in `snapshot ∪ trackedOrders` → call `GetOrderStatus(ctx, id)`.
4. Map to PSPOrder (§4.4.3).
5. Drop tracked entries on terminal status (`FILLED` / `CANCELLED`).
6. Drop tracked entries whose `FirstSeenAt + 25d` is exceeded, with `com.bitstamp.spec/retention_expired = true` on the final emission (5-day safety margin under Bitstamp's 30-day `order_status/` retention).

`HasMore` is always `false`. The dedupe surface (`OrderAdjustmentID`) includes Status + BaseQuantityFilled + Fee + FeeAsset, so re-emitting per cycle with refreshed fills is idempotent.

#### 4.4.2 Status

| B — `order_status.status` | F — `models.OrderStatus` |
|---|---|
| `In Queue` | `ORDER_STATUS_PENDING` |
| `Open` + `len(transactions) == 0` | `ORDER_STATUS_OPEN` |
| `Open` + `len(transactions) > 0` | `ORDER_STATUS_PARTIALLY_FILLED` |
| `Finished` | `ORDER_STATUS_FILLED` |
| `Canceled` / `Cancel pending` | `ORDER_STATUS_CANCELLED` |
| anything else | `ORDER_STATUS_OPEN` + Info |

Bitstamp does not emit `Expired` — unknown values default to OPEN rather than coercing to terminal.

#### 4.4.3 Field mapping

| F — `models.PSPOrder` | Source | Notes |
|---|---|---|
| `Reference` | `order_status.id` | Matches the live-pipeline reference. |
| `ClientOrderID` | `order_status.client_order_id` | Optional. |
| `CreatedAt` | `order_status.datetime` (or `trackedOrder.FirstSeenAt` fallback) | UTC. |
| `Direction` | `order_status.type` (`"0"`→BUY / `"1"`→SELL) | |
| `Type` | `order_status.subtype` via §4.4.5 (LIMIT / MARKET / INSTANT / STOP_LIMIT) | MARKET orders surface here even though they never sit in `open_orders/`. |
| `Status` | per §4.4.2 | |
| `BaseQuantityOrdered` | `amount_remaining + sum(transactions[<base>])` at base precision | `nil` when `amount_remaining` is absent (surface unknown honestly). |
| `BaseQuantityFilled` | sum of `transactions[].<base>` at base precision | Self-trade rows (same `tid`) deduplicated. |
| `LimitPrice` | `trackedOrder.LimitPrice` at price precision | `nil` for MARKET / INSTANT (no first-sight capture). |
| `QuoteAmount` | sum of `transactions[].<quote>` at quote precision | |
| `QuoteAsset` / `PriceAsset` / `FeeAsset` | quote ticker → `SYMBOL/<precision>` | Bitstamp charges fees + quotes in quote currency. |
| `AverageFillPrice` | `QuoteAmount × 10^basePrec / BaseQuantityFilled` | Zero when no fills (downstream treats zero as "not yet filled"). |
| `Fee` | sum of `transactions[].fee` at quote precision | |
| `TimeInForce` | inferred from subtype per §4.4.5 | Bitstamp surfaces no explicit TIF. |
| `SourceAccountReference` / `DestinationAccountReference` | tickers from `market` ("BTC/USD" → ["BTC","USD"]) | BUY: source=quote, dest=base; SELL: source=base, dest=quote. |
| `Metadata` | see §6.2 — includes `order_subtype` + `order_status_datetime` | |
| `Raw` | full `order_status` JSON | Self-contained — first-sight snapshot no longer merged in. |

#### 4.4.4 Historical-trade gap

The live pipeline above captures orders the connector has seen at least once while open. It does NOT capture:

1. Orders placed and fully filled **before** the connector was installed.
2. Orders > 30 days old that fell out of `order_status/` retention before the connector seeded them.

For both cases, the per-fill rows exist in `user_transactions/` as `type=2` rows with `order_id`. Aggregating by `order_id` reconstructs a `PSPOrder` per historical order (Status = FILLED, Type = UNKNOWN, LimitPrice = nil, metadata `com.bitstamp.spec/historical = "true"`).

**Not wired today.** `account_order_data/` was evaluated and rejected: it surfaces only `orderbook`-source events (no instant buy/sell) and requires 32-hex MarketEventID cursors. The planned implementation aggregates `type=2` rows directly. See §9.

#### 4.4.5 Subtype → OrderType + TimeInForce

| B — `subtype` | F — `OrderType` | TIF inferred |
|---|---|---|
| `LIMIT` | `ORDER_TYPE_LIMIT` | `GOOD_UNTIL_CANCELLED` |
| `MARKET` | `ORDER_TYPE_MARKET` | `IMMEDIATE_OR_CANCEL` |
| `INSTANT` | `ORDER_TYPE_MARKET` (no separate constant in Formance) | `IMMEDIATE_OR_CANCEL` |
| `STOP_LIMIT` | `ORDER_TYPE_STOP_LIMIT` | `GOOD_UNTIL_CANCELLED` |
| anything else | `ORDER_TYPE_UNKNOWN` + Info | `GOOD_UNTIL_CANCELLED` |

The raw subtype is preserved in `metadata.order_subtype` so consumers can disambiguate MARKET vs INSTANT.

### 4.5 PSPConversion — `user_transactions` type-36 → `PSPConversion`

Implemented in [`mappers/conversion.go`](mappers/conversion.go).

#### 4.5.1 Why `type=36` is a Conversion (not an Order)

Bitstamp returns two distinct primitives in `user_transactions/` that both look like "buys" and "sells" in the web UI. The reliable read-side distinction:

| Wire | Has `order_id`? | Lifecycle | Formance model |
|---|---|---|---|
| `type=2` (Trade — order fill) | yes | order-book — In Queue → Open → Finished / Canceled | `PSPOrder` (§4.4) |
| `type=36` (Instant buy/sell) | no | atomic — settled in one round-trip | `PSPConversion` (this section) |

Asset class plays **no** role. Bitstamp tags every crypto (BTC, USDC, EURC, …) as `currency.type = "crypto"` — there is no native `stablecoin` tag. A `type=36` BTC↔EUR row and a `type=36` USDC↔EUR row are the same primitive (spot-priced atomic swap). Downstream consumers wanting "market exposure" vs "stable-value swap" semantics apply their own stablecoin allow-list against `SourceAsset` / `DestinationAsset`.

The `/buy/instant/` and `/sell/instant/` write endpoints can produce **both** `type=2` and `type=36` rows depending on request shape — the wire `type` (not the write endpoint) is the disambiguator.

#### 4.5.2 Mapping

The conversions task shares the `user_transactions/` stream with payments but holds its own watermark (`conversionsState`), so the two cursors advance independently.

| F — `models.PSPConversion` | B — `UserTransaction` | Notes |
|---|---|---|
| `Reference` | `id` (int64 → string) | |
| `CreatedAt` | `datetime` | |
| `SourceAsset` / `SourceAmount` | negative-amount leg → `SYMBOL/<precision>`, `abs` value | The leg the user paid with. |
| `DestinationAsset` / `DestinationAmount` | positive-amount leg | The leg the user received. |
| `Fee` / `FeeAsset` | `fee` at quote precision / quote ticker | Nil when `fee` is zero (no fabricated zero in the wrong asset). |
| `Status` | constant `CONVERSION_STATUS_COMPLETED` | `user_transactions/` returns settled-only. |
| `SourceAccountReference` / `DestinationAccountReference` | currency tickers | Mirror PSPAccount references. |
| `Metadata` | see §6.3 — includes `rate` (e.g. `0.86047` from `usdc_eur`) | |
| `Raw` | full row | |

**Detection.** `tx.Type == "36"` AND exactly one negative + one positive non-zero known-currency amount. Non-matching rows return `(nil, nil)` — orchestrator skips silently.

**Regression fixture** (locked in [`mappers/conversion_test.go`](mappers/conversion_test.go)):

```json
{"id": 458254264, "datetime": "2025-09-25 14:42:59.894846", "type": "36", "fee": "0.000000",
 "eur": "-5.00", "usdc": "5.810770", "usdc_eur": 0.86047, "usd": 0.0, "btc": 0.0}
```

→ `EUR → USDC`, `SourceAmount = 500` (precision 2), `DestinationAmount = 5810770` (precision 6), `metadata.rate = "0.86047"`.

#### 4.5.3 `user_transactions.type` → `PaymentType` (complete enum)

Implemented as a table-driven function in [`mappers/status.go`](mappers/status.go).

| B — code | Constant | F — PaymentType | Notes |
|---|---|---|---|
| `0` | `txTypeDeposit` | `PAYMENT_TYPE_PAYIN` | |
| `1` | `txTypeWithdrawal` | `PAYMENT_TYPE_PAYOUT` | |
| `2` | `txTypeMarketTrade` | **skipped** | Order fill — surfaced as `PSPOrder`; historical-trade gap §4.4.4. |
| `14` | `txTypeSubAccountTransfer` | sign-based PAYOUT / PAYIN | §4.3.4 cross-account. |
| `25` | `txTypeStakingCredit` | `PAYMENT_TYPE_PAYIN` | Funds arriving in a staking wallet. |
| `26` | `txTypeStakingSent` | `PAYMENT_TYPE_PAYOUT` | Funds leaving for staking. |
| `27` | `txTypeStakingReward` | `PAYMENT_TYPE_PAYIN` | Yield. |
| `32` | `txTypeReferralReward` | `PAYMENT_TYPE_PAYIN` | |
| `33` | `txTypeSettlementTransfer` | sign-based PAYOUT / PAYIN | Undocumented but observed; §4.3.4. |
| `35` | `txTypeInterAccountTransfer` | sign-based PAYOUT / PAYIN | Internal movement; §4.3.4. |
| `36` | `txTypeBuySell` | **skipped** | Conversion — §4.5. |
| anything else | — | `PAYMENT_TYPE_OTHER` | Info-logged with `tx.id`. |

---

## 5. Design principles

These patterns appear in every resource above. Documented once here, referenced rather than re-explained.

**Try-and-skip cache.** Endpoints whose availability depends on permissions the API key may not hold (`crypto-transactions/` on sub-accounts; `my_markets/` + `fees/*` on restricted scopes) are wrapped in a process-lifetime skip cache. The first `DerivativesUnsupportedError` (or equivalent typed error) on a given path flags it; subsequent calls short-circuit. Logged at Info on the first mark for operator visibility.

> **Operational note.** The skip cache lives for the lifetime of the connector process. There is no portable signal that Bitstamp permissions changed, so a key whose ACLs are widened after the fact will continue skipping the previously-blocked endpoint until the connector is restarted (uninstall + reinstall, or worker rollout).

**Source-prefixed References.** Payments from `user_transactions/` use the bare `id` (numeric); `crypto-transactions/` uses `ct-dep:<id>` / `ct-wd:<txid>` / `ct-iou:<txid>`; `withdrawal-requests/` uses `wr:<id>`. The metadata `source` key carries the same information for filtering; the prefix makes Reference itself unambiguous. The mandatory `com.bitstamp.spec/source` metadata makes the source filterable without parsing the Reference.

**advanceCursor — never reset on empty.** Every watermark in `state.go` is updated via `advanceInt64Cursor(current, candidate) → max(current, candidate)`. An empty response (candidate = 0) preserves the watermark; equal is a no-op; strictly larger advances. Three independent regressions on other connectors were variants of "wrote candidate over current without checking" — keeping the primitive in one place makes the bug unreachable here.

**Spot-only stance.** Bitstamp ships derivatives (perps + margin) on the same REST host as spot, gated by account-type permissions. The connector is spot-only:
- `client/error.go` recognises `API5506: "Trade account does not support derivatives."` and the client returns the typed `DerivativesUnsupportedError` for the orchestrator's `errors.As` checks.
- Any `user_transactions` / `order_status` row carrying `margin_mode` / `leverage_rate` triggers an Error log and skip at the mapper level. A derivatives-enabled customer sees the loud signal that derivatives support is missing rather than silent mis-classification.

**FromPayload pattern (balances).** `fetch_balances` is non-periodic and fanned out per parent account; the balance derives from `PSPAccount.Raw`. No second API call per cycle.

**One connector per account scope.** Bitstamp API keys are bound to a single account scope (Main or one named sub-account); endpoint permissions are granted at the key level. Multi-scope orgs install one connector per key. The sign-based cross-account transfer model (§4.3.4) lets downstream consumers reconcile both halves across separate installs.

**Error-policy matrix.** Two axes, applied consistently across capabilities:

| Trigger | Policy | Rationale |
|---|---|---|
| Top-level source error (PSP returns 5xx, network failure, unmarshal of state JSON, etc.) | **fail the whole cycle** — return wrapped error to the engine | The engine treats any activity error as a cycle-level failure; partial responses are dropped by the workflow layer. Failing loudly is the only honest signal. |
| Per-row map error (single malformed wire row that fails currency lookup / amount parse / datetime parse) | **log + continue** at Error level | A single bad row must not block the other 999 rows in the page. The cursor still advances past the row — re-fetching would fail again identically. |
| Unknown / new enum value (tx type, status, scheme not in our tables) | **map to `*_UNKNOWN` / `*_OTHER` and log at Info** | Loud enough that ops see new codes; conservative enough that the row still surfaces. Never coerces to a terminal status. |
| Permission-gated endpoint returns `DerivativesUnsupportedError` (or equivalent) | **mark in try-and-skip cache + log Info on first hit + treat as empty response** | Bitstamp endpoints are scope-specific; sub-account keys cannot poll Main-only endpoints. The skip cache prevents churn without surfacing the gap as an error. |

**Accounts is the exception**: it fails the cycle on a per-row map error (because account currencies are a finite, install-time-known set and a single failure indicates a genuine mismatch with the currencies cache). Payments / orders / conversions log+continue because their row volume can be large.

---

## 6. Metadata keys

All keys are namespaced under `MetadataPrefix = "com.bitstamp.spec/"` (declared in [`mappers/metadata.go`](mappers/metadata.go)).

### 6.1 Payments

| Key | Value | Present? |
|---|---|---|
| `source` | `"user_transactions"` / `"crypto_transactions"` / `"withdrawal_requests"` | always |
| `type` | `tx.type` (user_transactions) / `"deposit"` / `"withdrawal"` / `"ripple_iou"` (crypto-transactions) / integer subtype (withdrawal-requests) | always |
| `fee` | wire fee value | when non-zero |
| `order_id` | `tx.order_id` | when present on user_transactions row |
| `transfer_pair_id` | `tx.id` as string | only on rows that are one leg of a two-legged movement |
| `transfer_direction` | `"outgoing"` / `"incoming"` | same |
| `counterparty_sub_account_id` / `..._name` | counterparty id / display name | only when the row carries them |
| `network` | `"bitcoin"` / `"ethereum"` / `"solana"` / … | crypto-source rows + withdrawal-requests when present |
| `txid` | on-chain transaction id | crypto-source rows |
| `destination_address` | wallet address (crypto) or bank address (fiat) | when present |
| `pending_reason` | e.g. `"ADDRESS_VERIFICATION_NEEDED"` | only on PENDING crypto deposits |
| `bank_transaction_id` | `withdrawal-requests.transaction_id` | when present on processed fiat withdrawals |

### 6.2 Orders

| Key | Source |
|---|---|
| `currency_pair` | from `order_status.market`, lowercased |
| `order_type` | lowercased `subtype` |
| `order_subtype` | raw `subtype` (preserves MARKET vs INSTANT distinction) |
| `order_status_datetime` | raw wire timestamp |
| `client_order_id` | when non-empty |
| `retention_expired` | `"true"` only on the forced-final emit (§4.4.1 step 6) |

### 6.3 Conversions

| Key | Source |
|---|---|
| `type` | always `"36"` |
| `currency_pair` | `<src>_<dst>` lowercase |
| `rate` | dynamic `<src>_<dst>` value (e.g. `usdc_eur = 0.86047`) |
| `fee` | when non-zero |

### 6.4 Accounts (enrichment)

Populated only when the install-time caches have data for this currency. See §7 for the source endpoints.

| Key | Source |
|---|---|
| `networks` | JSON-encoded list of supported blockchain networks |
| `withdrawal_fees` | JSON-encoded map keyed by network |
| `tradable_markets` | JSON list of pair URL symbols this API key can trade |
| `fee_tier_maker` / `fee_tier_taker` | maker / taker rate for the matching pair |
| `min_order_value` | from `/markets/` |
| `market_type` | `"SPOT"` (derivatives markets skipped) |

---

## 7. State (cursors / watermarks)

Defined in [`state.go`](state.go). All persisted as opaque JSON via `req.State` / `resp.NewState`.

```go
type paymentsState struct {
    UserTransactions   userTransactionsState   `json:"userTransactions"`
    CryptoTransactions cryptoTransactionsState `json:"cryptoTransactions"`
    WithdrawalRequests withdrawalRequestsState `json:"withdrawalRequests"`
}

type userTransactionsState   struct { LastTransactionID int64 `json:"lastTransactionID"` }
type cryptoTransactionsState struct {
    DepositsSinceTs    int64 `json:"depositsSinceTs"`
    WithdrawalsSinceTs int64 `json:"withdrawalsSinceTs"`
    RipplesSinceTs     int64 `json:"ripplesSinceTs"`
}
type withdrawalRequestsState struct { LastID int64 `json:"lastID"` }

type ordersState  struct { TrackedOrders map[string]trackedOrder `json:"trackedOrders"` }
type trackedOrder struct {
    LastStatus  string    `json:"lastStatus"`
    FirstSeenAt time.Time `json:"firstSeenAt"`
    LimitPrice  string    `json:"limitPrice"`
}

type conversionsState struct { LastTransactionID int64 `json:"lastTransactionID"` }
```

`paymentsState.UnmarshalJSON` migrates the pre-multi-source flat shape (`{"lastTransactionID": N}`) into `UserTransactions.LastTransactionID` so existing installs keep their watermark on upgrade. `ordersState.UnmarshalJSON` tolerates legacy fuller `trackedOrder` shapes by silently ignoring obsolete fields.

The install-time enrichment caches (markets, my_markets, trading fees, withdrawal fees, full currency list) live on `Plugin` itself behind a 24h TTL — they are NOT part of the connector state. See [`enrichment.go`](enrichment.go).

---

## 8. Known limitations

| Limitation | Why | Workaround |
|---|---|---|
| One connector per account scope | API keys are bound to a single scope; permissions are key-level. No portable fan-out. | Install one connector per key. Sub-account transfers correlate via `transfer_pair_id` metadata once both legs are integrated. |
| `bitstampGenesis` sentinel as account `CreatedAt` | Bitstamp does not expose per-currency creation dates. | Treat as "creation date unknown, definitely before this". |
| `order_status/` retention ~30 days | Tracked orders inactive > 30d become unfetchable. | Force-final emit at 25d with `retention_expired = true`, then drop. |
| `order_status/` does not return original `price` | The endpoint only carries status, fills, and the live fields. | Captured on first sight from `open_orders/` (§4.4.1). |
| No historical-trade backfill | Orders filled before install (or > 30d old at first poll) are absent from `open_orders/` + `order_status/`. | Per-fill rows exist in `user_transactions/` as `type=2`. Planned mapper aggregates by `order_id` — see §9. |
| Sub-account transfers absent from API on Main scope | The web UI shows them but probe found zero type-14 / 33 / 35 rows in `user_transactions/` on a Main-account key. `/sub_accounts/` returns 404. The `X-Auth-Subaccount-Id` header is silently ignored. | Install one connector per sub-account. The PAYOUT/PAYIN mapping in §4.3.4 activates as soon as Bitstamp exposes the rows on any endpoint already polled. Hypothesis worth verifying with a sub-account-scoped key: type-14 rows may surface from the destination sub-account's scope. |
| Counterparty field name on type-14 / 33 / 35 rows | Unobservable from the Main scope (no rows). | Metadata keys `counterparty_sub_account_id` / `..._name` reserved; mapper omits them when absent. |
| `user_transactions/` settled-only | The endpoint returns settled history; there is no PENDING state. | Pending crypto deposits and fiat withdrawals surface via `crypto-transactions/` and `withdrawal-requests/` respectively. |
| Self-trade fill semantics | `user_transactions.type=2` may carry `self_trade` markers; the connector dedupes by `tid` in `mappers/order.go`. The live shape needs validation against a real self-trade fixture. | Defensive dedupe in place; revisit when a fixture is available. |
| Derivatives surface ignored | Spot-only stance (§5). | `API5506` skip-cache; rows with derivatives markers Error-log and skip. |
| Multi-asset payments rows skipped if neither 1 nor exactly 2 non-zero known currencies | Defensive — matches neither payment nor conversion. | Info-logged; row preserved in `Raw` if surfaced upstream. |

---

## 9. Future work

### 9.1 Historical-trade backfill from `user_transactions.type=2`

Closes the §4.4.4 gap. Single-PR scope, no new endpoints.

1. **Mapper** (`mappers/order_from_user_tx.go`):
   - `UserTransactionToHistoricalOrderFill(currencies, tx)` returns a fill for `tx.Type == "2"` rows; nil otherwise.
   - `AggregateHistoricalOrder(fills)` reduces N fills sharing the same `order_id` into one PSPOrder with `Status = FILLED`, `Type = UNKNOWN`, `LimitPrice = nil`, `BaseQuantityOrdered = BaseQuantityFilled`, metadata `historical = "true"`.

2. **Orchestrator** (`orders.go`): after the open-orders snapshot loop, walk the same `user_transactions/` stream payments uses, filter to `type=2`, group by `order_id`, skip ids already present in `state.TrackedOrders` (avoids double-emission).

3. **Dedupe**: `PSPOrder.Reference = order_id` in both pipelines; `OrderAdjustmentID` covers status + fill aggregates, so a historical order later visible via `open_orders/` (defensive case — cannot happen in practice) generates the same reference with a corrected richer payload.

4. **State migration**: zero-value `HistoricalLastTransactionID` triggers full backfill on first cycle. The existing `paymentsState` backward-compat decoder absorbs the addition unchanged.

5. **Tests**: 3 type-2 rows sharing one `order_id` → 1 PSPOrder; type-2 row whose `order_id` is in `TrackedOrders` (skipped); mixed cycle verifying both pipelines emit without overlap.

### 9.2 Surfaces explicitly out of scope

| Surface | Why excluded |
|---|---|
| WebSocket API v2 (`wss://ws.bitstamp.net/`) | Event-driven sync requires a persistent subscription, not periodic polling. |
| FIX v2 Gateway | Separate integration surface — would warrant a dedicated `bitstamp-fix` connector. |
| Derivatives (margin / perps) | Distinct asset model (positions, leverage). Spot-only stance (§5). |
| Staking / lending (`/earn/*`) | New product domain — would need new PSP model concepts. |
| Travel rule (TFR) compliance (`/travel_rule/*`) | Compliance configuration, not a sync primitive. |
| Instant convert addresses (`/instant_convert_address/*`) | Configuration UX — underlying deposits flow through `crypto-transactions/`. |
| `account_order_data/` | Surfaces only `orderbook`-source events; instant buy/sell is NOT exposed. Cursors are 32-hex MarketEventIDs requiring anchoring. Not a substitute for the §9.1 historical-trade backfill. |
| OTC RFQ | API endpoints non-public; gated behind institutional accounts. |
| Open Banking PSD2 (`/api-psd2/`) | Regulatory TPP API — not a payment-connector use case. |

---

## 10. References

- [Bitstamp REST API v2 docs](https://www.bitstamp.net/api/) — canonical endpoint reference.
- This document is the source of truth for the mapping; any change to API surface or mapping logic must update this file in the same commit as the code change.

---

## Appendix A: Endpoint inventory

Every Bitstamp REST v2 endpoint the connector touches or explicitly skips, with the verdict for each. Validated against a Main-account API key.

| Endpoint | Method | Status | Note |
|---|---|---|---|
| `/api/v2/currencies/` | GET | **USED** | Loaded at install + every 24h. `networks[]` enrichment. |
| `/api/v2/account_balances/` | POST | **USED** | One row per supported currency. Drives PSPAccount + PSPBalance. |
| `/api/v2/user_transactions/` | POST | **USED** | Cold-start `since_id=nil, sort=asc` walks full history. Inclusive `since_id`. Feeds payments + conversions. |
| `/api/v2/open_orders/all/` | POST | **USED** | Server-cached ~10s. First-sight `LimitPrice` capture. |
| `/api/v2/order_status/` | POST | **USED** | Rich shape — market / type / subtype / datetime / amount_remaining / fills. |
| `/api/v2/crypto-transactions/` | POST | **USED** | Main-account only. 3 buckets: deposits (PENDING/COMPLETED), withdrawals, ripple IOUs. |
| `/api/v2/withdrawal-requests/` | POST | **USED** | `limit + offset` both required. Full fiat lifecycle. |
| `/api/v2/markets/` | GET | **USED** | Public. `market_type` distinguishes SPOT vs derivatives variants. |
| `/api/v2/my_markets/` | POST | **USED** | Signed POST (GET returns 400). Per-key tradable allow-list. |
| `/api/v2/fees/trading/` | POST | **USED** | Per-pair maker/taker rates. |
| `/api/v2/fees/withdrawal/` | POST | **USED** | One row per (currency, network). |
| `/api/v2/sub_accounts/` | POST | OUT — probed 404 | No discovery endpoint exists. §4.3.4 / §8. |
| `/api/v2/account_order_data/`, `/api/v2/order_data/` | POST | OUT — wrong surface | Orderbook-source events only (no instant buy/sell). 32-hex MarketEventID cursors. |
| `/api/v2/crypto-transactions/deposits/`, `…/{id}/`, `…/{id}/reject/` | GET/POST | OUT — redundant | Subset of bulk `/crypto-transactions/`. Includes write paths (travel-rule). |
| `/api/v2/earn/*` | GET/POST | OUT — out of scope | Staking / lending position management. |
| `/api/v2/travel_rule/*` | GET/POST | OUT — compliance | TFR compliance configuration. |
| `/api/v2/instant_convert_address/*` | POST | OUT — configuration | Underlying deposits surface via `/crypto-transactions/`. |
| `/api/v2/transfer-to-main/`, `…/transfer-from-main/` | POST | OUT — write | Require `subAccount` int only obtainable from web UI. |
| `/api/v2/buy/*`, `/api/v2/sell/*`, `…/cancel_order/`, `…/cancel_all_orders/`, `…/replace_order/`, `…/get_max_order_amount/` | POST | OUT — write | Read-only connector. |
| `/api/v2/withdrawal/open/`, `…/cancel/`, `…/status/`, `…/{currency}_withdrawal/`, `…/ripple_withdrawal/` | POST | OUT — write | Read-only connector. Pending requests surface via `/withdrawal-requests/`. |
| `/api/v2/{currency}_address/`, `…/btc_unconfirmed/`, `…/ripple_address/` | POST | OUT — write | Address issuance. |
| `/api/v2/revoke_all_api_keys/` | POST | OUT — destructive | Never. |
| `/api/v2/websockets_token/` | POST | OUT — out of scope | WS integration would be a separate connector. |
| `/api/v2/open_positions/`, `…/position_*`, `…/trade_history/`, `…/margin_*`, `…/leverage_settings/`, `…/funding_rate*`, `…/close_position*`, `…/adjust_position_collateral/`, `…/collateral_*`, `…/estimated_order_impact/` | GET/POST | OUT — derivatives | Spot-only stance (§5). Try-and-skip catches HTTP 403 on first hit. |
| `/api/v2/account_balances/{currency}/`, `/api/v2/fees/trading/{market}/`, `/api/v2/fees/withdrawal/{currency}/`, `/api/v2/user_transactions/{market}/`, `/api/v2/open_orders/{market}/` | POST | OUT — redundant | Per-currency / per-pair convenience variants — bulk endpoints already cover. |
| `/api/v2/ticker/`, `/api/v2/order_book/`, `/api/v2/ohlc/`, `/api/v2/transactions/{market}/` | GET | OUT — market data | Public market data — not a payment-connector use case. |
