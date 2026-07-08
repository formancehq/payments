# Kraken Pro ↔ Formance Payments — field mapping

Authoritative reference for how the Kraken Pro EE connector translates Kraken Spot REST v0 responses (Pro VIP variant) into Formance PSP types. Co-located with the code so the mapping and the implementation cannot drift.

| Symbol | Meaning |
|---|---|
| **K** | Kraken REST v0 field (JSON) |
| **F** | Formance PSP field (Go struct, `CamelCase`) |
| `→` | sync direction (Kraken → Formance) |

---

## 1. Overview

The connector is **read-only spot-only**, one install per Kraken Pro account. It surfaces five capabilities:

| F — Capability | Kraken endpoint(s) | Notes |
|---|---|---|
| `CAPABILITY_FETCH_ACCOUNTS` | `POST /0/private/BalanceEx` | one PSPAccount per asset present in BalanceEx |
| `CAPABILITY_FETCH_BALANCES` | `POST /0/private/BalanceEx` | derived from the same BalanceEx call, no extra hop |
| `CAPABILITY_FETCH_PAYMENTS` | `POST /0/private/Ledgers` (filtered) | deposit / withdrawal / transfer / staking / reward / adjustment / dividend / credit |
| `CAPABILITY_FETCH_ORDERS` | `POST /0/private/ClosedOrders` | each row is the order with cumulative `vol_exec`/`cost`/`fee`; per-fill txids ride along when `trades:true` (OpenOrders is intentionally not polled — see §8) |
| `CAPABILITY_FETCH_CONVERSIONS` | `POST /0/private/Ledgers` (filtered) | rows with `type` ∈ {`conversion`, `sale`, `marginconversion`, `margin_conversion`} grouped by `refid` |

The OpenAPI spec served at `https://api.vip.uat.lobster.kraken.com/spec` is the source of truth for request shapes, enums, and parameter names. Response fields are documented at [docs.kraken.com/api/docs/rest-api](https://docs.kraken.com/api/docs/rest-api).

---

## 2. Configuration

Defined in [`config.go`](config.go). OpenAPI v3: `V3KrakenproConfig`.

| Field | Required | Default | Purpose |
|---|---|---|---|
| `apiKey` | yes | — | Sent in `api-key` header |
| `apiSecret` | yes | — | Base64-encoded HMAC-SHA512 signing secret. Never logged. |
| `endpoint` | **yes** | — | **Required.** This client speaks the Kraken Pro VIP dialect (JSON body, lowercase `api-*` headers), which is incompatible with the public `api.kraken.com` Spot API (form-encoded, `API-Key` header). A blank endpoint must not silently fall back to the public host, so the install fails fast. UAT: `https://api.uat.kraken.com`. Do **not** use `https://api.vip.uat.lobster.kraken.com`: that host 301-redirects every private endpoint, and the Go HTTP client converts POST→GET on 301 (RFC 7231) which strips the signed body and produces a synthetic `EGeneral:Unknown method`. The lobster URL is fine for the browser-facing Pro UI; the API client must point at the un-redirected host directly. |
| `pollingPeriod` | no | `30m` (min `20m`) | Periodic sync cadence |

### Auth — HMAC-SHA512

Pro VIP uses **lowercase** headers and a JSON request body. The standard Kraken signing scheme still applies, with the nonce supplied both in the `api-nonce` header and the JSON body's `nonce` field.

| Header | Value |
|---|---|
| `api-key` | `<apiKey>` |
| `api-sign` | `base64( HMAC-SHA512( base64decode(apiSecret), uriPath \|\| SHA256(nonceASCII \|\| bodyBytes) ) )` |
| `api-nonce` | `time.Now().UnixNano()` decimal string (stateless, computed per call). Nanoseconds (19 digits) sit above any ms/us-precision client that may have already incremented the per-key nonce floor on Kraken's side. |
| `api-otp` | optional 2FA — not configured in v1 |
| `Content-Type` | `application/json` (always carries `{"nonce":"…"}` at minimum) |

`uriPath` is everything after the host (e.g. `/0/private/Balance`). The nonce is a strictly-increasing integer in **nanoseconds** (`time.Now().UnixNano()`). Re-using a stale (<=) nonce returns `EAPI:Invalid nonce`.

Nonce generation and signing live in the client (`client/client.go` `signRequest`), not in the HTTP transport: the signature covers the request body that carries the nonce, so it is computed where the body is built. `/0/private/*` requests are signed; `/0/public/*` are sent unsigned.

**Nonce & multiple workers.** The nonce is intentionally stateless (no stored counter): two `payments-worker` pods share one API key, so no in-process counter can guarantee global ordering anyway. A `UnixNano` nonce is strictly increasing for sequential calls; the rare out-of-order `EAPI:Invalid nonce` (concurrent calls / cross-pod races) is classified **retriable with backoff** (mapped to `ErrStatusCodeTooEarly`, not fatal) so the next attempt sends a fresh, higher nonce. The backoff matters because Kraken temporarily locks a key after too many invalid-nonce errors. For production HA the real fix (per Kraken) is a **dedicated API key per worker** or a configured **nonce window** on the key.

### Public endpoints

`/0/public/Time`, `/0/public/Assets`, `/0/public/AssetPairs` require no signing. They are invoked over the same HTTP client so the metrics + otel transport instruments them uniformly.

---

## 3. Workflow & pagination

`workflow()` declares the task tree:

```text
fetch_accounts          (periodic root)
fetch_balances          (periodic root; reads BalanceEx, no parent context needed)
fetch_payments          (periodic root)
fetch_orders            (periodic root)
fetch_conversions       (periodic root)
```

All tasks are independent periodic roots. `fetch_orders` is **not** nested under `fetch_accounts`: Kraken cannot filter orders by account, so nesting would make the engine fan out one identical full-orders fetch per account. Orders (and payments/conversions) resolve their account references from each row's own raw Kraken code — for orders, the pair's `base`/`quote` codes; for payments/conversions, the ledger entry's `asset` — which is exactly the per-variant reference `fetch_accounts` emits. So there is no in-process wallet cache, no DB `AccountLookup`, and no accounts dependency.

### Pagination invariants — frozen-end + ofs window

Every history-bearing Ledgers/ClosedOrders stream uses one shared algorithm
(`ledgerWindow` in [state.go](state.go)). Kraken returns the newest `PAGE_SIZE`
rows by default and `start` is a strict lower bound, so a naive `start=watermark`
forward poll **skips rows** whenever more than one page arrives between cycles
(it advances the watermark to the newest row but only saw the newest 50). A naive
`ofs` backfill **drifts** when new rows arrive mid-walk. We fence each drain:

1. **Freeze `End`** at the wall-clock instant the drain starts, persisted in
   state. Rows arriving mid-drain (`time > End`) fall outside the window and so
   cannot shift `ofs` positions.
2. **Page `ofs = 0, PAGE_SIZE, 2·PAGE_SIZE, …`** within `(Watermark, End]` until a
   short page (the window is drained). `HasMore=true` between pages so Temporal
   keeps dispatching until the window empties.
3. **Promote `Watermark = End`** only once the window fully drains, then reset.

This unifies first-install backfill (`Watermark=0`) with incremental polling.
Correctness is **positional**: `ofs` strictly increases over a finite frozen
window, covering every row exactly once — so it terminates, never skips, and is
immune to equal-timestamp pages (it indexes position, not time or ID).

**Why `ofs` and not an ID cursor.** `ofs` is marked `deprecated` in the spec and
`start`/`end` accept an ordered ledger/order-tx-ID boundary, which would be the
modern choice — *but* Kraken's Ledgers/ClosedOrders responses are **unordered
JSON maps**, so we cannot determine a page's boundary ID to drive an ID cursor
(and a timestamp cursor loops on equal-timestamp full pages). `ofs` is the only
positional primitive that works with the actual response shape; the frozen `End`
makes it stable.

| Capability | Pagination |
|---|---|
| Accounts | n/a — BalanceEx single-shot. State tracks `AccountAssetsImportedAt: map[ref]rfc3339` so cycles are idempotent. |
| Balances | n/a — BalanceEx single-shot, no state. |
| Payments | frozen-end + ofs window on `/0/private/Ledgers`, classifying each row via `mappers.ClassifyLedgerType`. |
| Conversions | same window on `/0/private/Ledgers`, separate state struct; half-paired refids carry over via `Pending`, pruned once the watermark passes their time. |
| Orders | frozen-end + ofs window on `/0/private/ClosedOrders` (`closetime: "close"`). OpenOrders is not polled (unbounded page — see §8). |

`without_count: true` is always set for speed.

### History horizon

Kraken does not document a hard horizon cut-off for `Ledgers` / `TradesHistory`. The response carries a `count` field with the total available entries; the connector walks until offset >= count (signalled by an empty page). Practical limits:

- Page size: 50 default (Kraken doesn't honour a request to raise it beyond this on Spot REST).
- Rate limit: per-tier REST counter (Tier 2/3 ≈ 1 call/sec sustained). For a 100k-entry account, backfill takes ~2000 calls per orchestrator at 50/page → ~30 min wall-clock at 1 call/sec. Temporal handles this transparently via repeated `HasMore=true` activity invocations.
- Memory: each page is small (~5-50 KB JSON); no streaming required.

---

## 4. Asset normalisation

Kraken's asset codes carry legacy class prefixes and staking/earn suffix families. The connector emits a single canonical code per underlying asset so balances/payments don't fan out into five rows for `ADA` + `ADA.S` + `ADA.M` etc.

Normalisation is two steps in [`mappers/assets.go`](mappers/assets.go) (`NormalizeAsset`):

1. **Strip the suffix family** (`.S` `.M` `.B` `.F` `.P` `.T` `.HOLD` `.BASE`).
2. **Apply the explicit alias map** (`assetAliases`). This mirrors ccxt's `commonCurrencies` table and is *authoritative* — we deliberately do **not** algorithmically strip a leading `X`/`Z`, because:
   - some codes don't strip cleanly: `XXDG → DOGE` (not `XDG`), `XXLM → XLM`;
   - a blind strip mangles legitimate tickers that start with X/Z (`XCN`, `ZETA`, `ZRO`) — a real over-strip bug class that bit ccxt.

   Kraken stopped minting X/Z-prefixed codes years ago, so the alias set is effectively closed. Any unmapped code passes through unchanged (graceful degradation — still internally consistent because `/Assets` keys are normalised through the same function).

| Input | Output | Reason |
|---|---|---|
| `XXBT`, `XBT`, `XBT.M`, `XBT.F` | `BTC` | alias `XXBT/XBT → BTC` (after suffix strip) |
| `XXDG`, `XDG` | `DOGE` | alias — the old prefix-strip heuristic wrongly produced `XDG` |
| `ZUSD` / `ZEUR` / `ZGBP` / `ZCAD` / `ZJPY` / `ZAUD` | `USD` / … | fiat class-prefix aliases |
| `XETH` `XXRP` `XXLM` `XXMR` `XLTC` `XETC` `XZEC` `XMLN` `XREP` | `ETH` `XRP` `XLM` `XMR` `LTC` `ETC` `ZEC` `MLN` `REP` | crypto class-prefix aliases |
| `ADA.S`, `EUR.HOLD`, `BTC.B/.F/.M/.P/.T` | `ADA`, `EUR`, `BTC` | suffix families (B = yield, F = Earn auto, M = opt-in, P = parachain, T = tokenised, S = legacy staked, HOLD = pending) |
| `ZETA`, `XCN`, `ZRO` | unchanged | not legacy-prefixed — over-strip guard |

Observed in UAT probe (see `probe-transcript.md`): 828 spot assets, 27 `.S`, 7 `.HOLD`, 5 `.M`, 2 `.P`, 1 `.BASE`.

### Asset precision: `decimals` vs `display_decimals` (lossless by design)

`/0/public/Assets` returns **two** precisions per asset, and we deliberately key `currencies[symbol]` off `decimals` (the internal/ledger precision), not `display_decimals` (the UI rounding). `FormatAsset` then renders `SYMBOL/decimals`, so the precision in `DefaultAsset`, balance `Asset`, payment `Asset`, and order/conversion `SourceAsset`/`DestinationAsset` is Kraken's `decimals`.

This is why the emitted precision runs finer than the common market conventions:

| Asset (raw) | Kraken `decimals` | Kraken `display_decimals` | Emitted | Common convention |
|---|---:|---:|---|---|
| `XXBT` (BTC) | 10 | 5 | `BTC/10` | BTC/8 |
| `ZUSD` (USD) | 4 | 2 | `USD/4` | USD/2 |
| `USDC` | 8 | 4 | `USDC/8` | USDC/6 |

It is **not** a fixed `+2` we inject — each value is that asset's `decimals` field, which happens to sit a couple of digits finer than the usual unit (satoshi / cent / token-decimals).

**Why `decimals` (the trade-off):**

- **Lossless (the deciding factor).** Kraken reports amounts — balances, `vol_exec`, `cost`, fees, ledger amounts — carrying up to `decimals` significant digits (e.g. a real UAT balance `0.6645856520`, 10 dp). `ParseDecimalAmount` converts to minor units at the asset precision; keying off `decimals` guarantees every value round-trips exactly. Using `display_decimals` (BTC/5, USD/2) would **truncate/round** any amount finer than the UI precision → silent precision loss → reconciliation drift.
- **Cost — cross-connector consistency.** Formance encodes precision into the asset identifier (`USD/4`), and other connectors may emit a different precision for the same symbol (coinbaseprime uses ISO4217 for fiat → `USD/2`, API precision for crypto). So aggregating one symbol across connectors must be precision-aware; krakenpro's fiat precision (`USD/4`) will not string-match another connector's `USD/2`.

**Considered alternatives (not adopted):**

- *`display_decimals` everywhere* — matches the conventional table exactly but is lossy on high-precision Kraken amounts. Rejected.
- *Hybrid (ISO4217 for fiat + `decimals` for crypto)* — would align fiat with the platform (`USD/2`) while keeping crypto lossless. Viable if cross-connector fiat consistency becomes a requirement; deferred because it trades a (small) fiat-precision loss for consistency, and v1 favours faithful representation.

### Per-asset-class accounts (no aggregation)

Each Kraken asset class is its own account — the coinbaseprime wallet-per-asset model (TRADING/VAULT/ONCHAIN), here keyed by asset class. There is no balance aggregation; the normalised symbol survives as `DefaultAsset` / `Asset` while the account identity is the raw code:

- **Reference** = the raw Kraken code (`XXBT`, `XBT.M`, `ZUSD`, `ADA.S`) — Kraken's own stable per-variant id.
- **`wallet_type`** metadata = the class: `spot` (suffix-free, the trading class), `staked` (.S), `rewards` (.M), `yield` (.B), `earn` (.F), `parachain` (.P), `tokenised` (.T), `hold` (.HOLD), `margin` (.BASE).
- **`kraken_asset`** metadata = the raw code (equals Reference) — keeps spot/earn provenance explicit next to the normalised `DefaultAsset`.

Because distinct variants have distinct account references, the engine never sees a duplicate `(account, asset)` tuple, so balances report the **real per-variant amount** with no summing.

Orders, conversions and payments reference each leg's own raw Kraken code (the per-variant account `fetch_accounts` emits): payments/conversions use the ledger entry's `asset`, orders use the pair's `base`/`quote` codes. No in-process cache or account lookup is needed.

Payments/conversions also keep the precise raw asset in metadata (`kraken_asset`, plus `kraken_source_asset` / `kraken_destination_asset` for conversions).

---

## 5. Capability: `FETCH_ACCOUNTS`

**Endpoint:** `POST /0/private/BalanceEx` — request body `{"nonce": "…"}`.

**Response shape** (per [Kraken docs](https://docs.kraken.com/api/docs/rest-api/get-extended-balance)):

```json
{
  "error": [],
  "result": {
    "ZUSD":   { "balance": "171288.6158", "hold_trade": "8861.7898" },
    "XXBT":   { "balance": "1011.1908877900", "hold_trade": "0.0000000000" },
    "ADA.S":  { "balance": "457.1234", "hold_trade": "0.0000" }
  }
}
```

Each raw BalanceEx key → one PSPAccount keyed by the raw code.

| K | F field | Notes |
|---|---|---|
| (asset key) | `Reference` | raw Kraken code, e.g. `XXBT`, `XBT.M`, `ZUSD` |
| symbol + class | `Name` | human label, e.g. `BTC Spot`, `BTC Rewards` |
| n/a — stable sentinel | `CreatedAt` | `2011-08-01T00:00:00Z` (Kraken genesis) |
| `NormalizeAsset` + precision | `DefaultAsset` | `<SYMBOL>/<precision>` via `currency.FormatAsset` |
| (raw row) | `Raw` | the contributing `{code, entry}` |
| see §9 | `Metadata` | namespaced `com.krakenpro.spec/*` (`wallet_type` only — the raw code is already the `Reference`) |

Every BalanceEx variant that normalizes to a known symbol produces an account — **zero balances are not filtered** (Kraken only returns a row for an asset the account holds or has held). `fetchNextAccounts` and `fetchNextBalances` share one inclusion predicate (`mappers.IncludeBalanceEntry`) so a balance can never reference an account that was not emitted. Accounts mirror BalanceEx exactly: no synthetic spot account is generated. If value sits only in an earn variant there is no spot account, so an order referencing that pair's spot code may point at a not-yet-held account — order refs are optional/best-effort.

Cross-cycle de-dup: `accountsState.AccountAssetsImportedAt[reference]` records when an account (raw code) was first emitted; subsequent cycles skip already-seen accounts.

---

## 6. Capability: `FETCH_BALANCES`

Re-reads BalanceEx (cheap single-shot call) and emits one PSPBalance per **raw variant**, each keyed to its own per-class account. No aggregation: distinct account references (`XXBT`, `XBT.M`) mean the engine never sees a duplicate `(account, asset)` tuple, so each variant reports its real balance against its own account while the `Asset` field stays the normalised symbol.

The emitted balance is the **available** amount per Kraken's BalanceEx docs:
`balance + credit − credit_used − hold_trade` (clamped to ≥ 0). `credit` /
`credit_used` are populated on VIP/Pro accounts with a credit line and default to
zero on spot-only accounts. Fully-empty rows (balance, hold_trade and credit all
zero) are skipped.

| K | F field | Notes |
|---|---|---|
| (asset key) | `AccountReference` | raw Kraken code (matches the PSPAccount Reference) |
| `NormalizeAsset` + precision | `Asset` | `<SYMBOL>/<precision>` |
| `balance - hold_trade` | `Amount` | `*big.Int` minor units |
| orchestrator `now()` | `CreatedAt` | balances are snapshots; engine namespaces by ConnectorID |

---

## 7. Capability: `FETCH_PAYMENTS`

**Endpoint:** `POST /0/private/Ledgers` — request body filters: `start = LastLedgerTime`, `without_count: true`, no `type` (we filter client-side). Fixed PAGE_SIZE per cycle.

**Response shape**:

```json
{
  "error": [],
  "result": {
    "ledger": {
      "L4UESK-KG3EQ-UFO4T5": {
        "refid":   "TYH2WW-WHIOM-TFFLE6",
        "time":    1688019200.1234,
        "type":    "trade",
        "subtype": "",
        "aclass":  "currency",
        "asset":   "ZEUR",
        "amount":  "100.0000",
        "fee":     "0.4000",
        "balance": "1234.5600"
      }
    },
    "count": 18432
  }
}
```

**Type → PSPPayment.Type mapping** (the K `type` enum is closed in the OpenAPI spec):

| K `type` | F `Payment.Type` | Notes |
|---|---|---|
| `deposit` | `PAYMENT_TYPE_PAYIN` | external funding incoming |
| `withdrawal` | `PAYMENT_TYPE_PAYOUT` | external funding outgoing |
| `transfer`, `custodytransfer` | `PAYMENT_TYPE_TRANSFER` | internal movement (spot<->futures, subaccount, spot<->staking allocation; often carries a `subtype`). The spot leg is attributed by amount sign (negative=source, positive=destination); the counterparty wallet isn't tracked |
| `staking`, `reward`, `dividend`, `credit`, `nft_rebate` | `PAYMENT_TYPE_PAYIN` (positive amount) | rewards & rebates (staking REWARD income; the staking *allocation* move is a `transfer` above) |
| `adjustment`, `rollover`, `settled`, `reserve`, `reserved_fee`, `ic_settlement` | `PAYMENT_TYPE_OTHER` | bookkeeping / system entries |
| `nftcreatorfee` | `PAYMENT_TYPE_PAYOUT` | NFT creator fee outflow |
| `trade`, `eqtrade` | **skipped** (handled by FETCH_ORDERS) | sign-of-life log only |
| `nfttrade` | `PAYMENT_TYPE_OTHER` with metadata flag | spot-only stance: NFTs not first-class |
| `conversion`, `sale`, `marginconversion`, `margin_conversion` | **skipped** (handled by FETCH_CONVERSIONS) | |
| unknown future value | `PAYMENT_TYPE_OTHER` + warn-log with the row id | catalogue rule L8 |

**Field mapping** (post type-classification):

| K | F field | Notes |
|---|---|---|
| (map key — the ledger ID) | `Reference` | e.g. `L4UESK-KG3EQ-UFO4T5`; **not** `refid` which groups multi-leg events |
| `time` × 1e9 → `time.Time` | `CreatedAt` | float epoch seconds → UTC; strict monotonic per row |
| classified type (above) | `Type` | |
| `abs(amount)` × 10^precision | `Amount` | gross — `fee` is reported separately on the ledger (different row for trade-related fees) |
| `asset → NormalizeAsset` + precision | `Asset` | `<SYMBOL>/<precision>` |
| `models.PAYMENT_SCHEME_OTHER` | `Scheme` | Kraken doesn't expose card / SEPA / ACH per row in this endpoint |
| `models.PAYMENT_STATUS_SUCCEEDED` | `Status` | ledger entries are only written on settlement — there's no pending state at this layer |
| see §9 | `Metadata` | includes `refid`, `subtype`, `aclass`, `balance_after`, `kraken_type`, `kraken_asset` (the ledger id is the `Reference`, not duplicated in metadata) |
| (full row) | `Raw` | for replay / debugging |

`fee` from the ledger row is logged in metadata (`com.krakenpro.spec/fee`) but **not** subtracted from `amount`. For payments the fee is recorded only when material; for trade-related fees, see §8.

---

## 8. Capability: `FETCH_ORDERS`

**One endpoint:**

- `POST /0/private/ClosedOrders` — historical/closed orders. No cursor support (spec-confirmed); paged through the shared frozen-end + ofs window (§3) on close time (`closetime: "close"`, so a newly-closed order with an ancient opentm still surfaces).

**Why not OpenOrders.** Kraken's `OpenOrders` accepts no page-size limit, so a single drain can return an unbounded set (up to thousands of rows) and exceed Temporal's max activity payload. We therefore fetch **only** the page-bounded ClosedOrders. A closed order still carries its per-fill txids (`trades: true`), so fill traceability is preserved; the only thing lost is the in-flight OPEN/PARTIALLY_FILLED interim snapshot, which is deferred until Kraken adds an open-orders page limit (see §8.5 and the deferred-items table).

**Account references.** Order source/destination resolve to each leg's raw Kraken code (the pair's `base`/`quote` codes, e.g. `XXBT`/`ZUSD`) — the per-variant account reference `fetch_accounts` emits. No in-process cache or DB lookup. If the spot account isn't currently held the reference can point at a not-yet-emitted account; `PSPOrder.Validate()` permits this and refs are best-effort, so it never blocks the stream.

**`cl_ord_id`.** When an order carries a client-assigned id it maps to
`PSPOrder.ClientOrderID` and `metadata."com.krakenpro.spec/cl_ord_id"`.

ClosedOrders always passes `trades: true` so each row carries its per-fill txid list inline — **no extra `QueryTrades` call**, audit-grade traceability preserved.

**Why this is different from a fills-aggregation source** (e.g. TradesHistory): each row already carries the order's cumulative `vol_exec` / `cost` / `fee` / `status`. We don't aggregate across pages, so the emitted `PSPOrder.baseQuantityFilled` never bounces; the engine's adjustment trail collapses to one entry per real status change.

> **Adjustment granularity is per polling cycle, not per fill** — see §8.5 below for the full design rationale.

### Response shape (closed order, abridged)

```json
{
  "error": [],
  "result": {
    "closed": {
      "OQCLML-BW3P3-BUCMWZ": {
        "status":   "closed",
        "opentm":   1688665400.0,
        "closetm":  1688667626.5567,
        "descr":    { "pair": "XXBTZUSD", "type": "buy", "ordertype": "limit", "price": "27500.0" },
        "vol":      "1.00000000",
        "vol_exec": "1.00000000",
        "cost":     "27500.00",
        "fee":      "73.70",
        "price":    "27500.0",
        "trades":   ["TCWJEG-FL4SZ-3FKGH6", "TKH2SE-M7IF5-CFI7LT"]
      }
    },
    "count": 9876
  }
}
```

### Field mapping (per order)

| K | F field | Notes |
|---|---|---|
| (map key) | `Reference` | order id; engine namespaces by ConnectorID |
| `opentm` | `CreatedAt` | always the open time, so the open→closed upsert keeps a stable creation timestamp; `closetm` (when set) is preserved as the `close_time` metadata key |
| `descr.pair` → `base` / `quote` (cached AssetPairs) | source/destination asset (see direction below) | normalised via `NormalizeAsset` |
| `descr.type` (buy/sell) | `Direction` | `buy → ORDER_DIRECTION_BUY`, `sell → ORDER_DIRECTION_SELL` |
| `descr.ordertype` | `Type` | mapped via `MapOrderType` |
| `vol` at base precision | `BaseQuantityOrdered` | total ordered |
| `vol_exec` at base precision | `BaseQuantityFilled` | cumulative — no per-page aggregation |
| `cost` at quote precision | `QuoteAmount` | cumulative |
| `fee` at quote precision | `Fee` | cumulative |
| quote symbol + precision | `QuoteAsset`, `FeeAsset` | |
| quote symbol + dynamic price precision | `PriceAsset` | dynamic = max decimals across `descr.price`/`descr.price2`/`price`, capped at 10 |
| `price` (top-level avg fill price) | `AverageFillPrice` | parsed once; not derived from fills |
| `descr.price` / `descr.price2` | `LimitPrice` / `StopPrice` | order-type dependent — see price-mapping table below |
| `status` + (vol_exec vs vol) | `Status` | via `mapKrakenOrderStatus` (table below) |
| `expiretm` (when > 0) | `ExpiresAt` + `TimeInForce` | non-zero → `TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME` with `ExpiresAt` set to that instant; zero/absent → default `TIME_IN_FORCE_GOOD_UNTIL_CANCELLED`, `ExpiresAt` nil |
| pair `base`/`quote` raw codes | `SourceAccountReference`, `DestinationAccountReference` | BUY: src=quote code, dst=base code; SELL: inverted |
| `trades: [...]` | `metadata."com.krakenpro.spec/fills"` | comma-joined trade txids (free of charge — same response, no extra call) |

### Price mapping (`descr.price` / `descr.price2`)

Kraken overloads the two price fields by order type, so the mapping is not positional. `parseOrderAmounts` branches on `descr.ordertype`:

| `descr.ordertype` | `descr.price` → | `descr.price2` → | Notes |
|---|---|---|---|
| `limit`, `limit-maker`, `market`, `iceberg`, `settle-position` | `LimitPrice` | `StopPrice` | default branch; `LimitPrice` nil for market orders |
| `stop-loss`, `take-profit` | `StopPrice` (trigger) | — | `descr.price` is the trigger price; no limit leg |
| `stop-loss-limit`, `take-profit-limit` | `StopPrice` (trigger) | `LimitPrice` | `descr.price` trigger, `descr.price2` limit |
| `trailing-stop`, `trailing-stop-limit` | — | — | both left nil — `descr.price`/`price2` are **relative offsets** (signed/percent), not absolute prices; the resolved absolute trigger/limit live in Kraken's top-level `stopprice`/`limitprice`, which the connector does not currently decode |

### Status mapping (`mapKrakenOrderStatus`)

Mirrors `coinbaseprime/orders.go::mapCoinbaseStatus`. The Kraken `status` enum is small (pending/open/closed/canceled/expired) and the FILLED vs PARTIALLY_FILLED distinction is derived from the `(vol_exec, vol)` pair via `*big.Float` comparison at 256-bit precision.

| Kraken `status` | `vol_exec` vs `vol` | Formance `OrderStatus` |
|---|---|---|
| `pending` | n/a | `PENDING` |
| `open` | 0 | `OPEN` |
| `open` | 0 < exec < vol | `PARTIALLY_FILLED` |
| `closed` | exec >= vol | `FILLED` |
| `closed` | 0 < exec < vol | `PARTIALLY_FILLED` |
| `closed` | 0 | `CANCELLED` (closed without filling) |
| `canceled` / `cancelled` | exec > 0 | `PARTIALLY_FILLED` |
| `canceled` / `cancelled` | 0 | `CANCELLED` |
| `expired` | n/a | `EXPIRED` |
| unknown future | n/a | `PENDING` + warn-log |

### Order type mapping (`MapOrderType`)

| Kraken | Formance |
|---|---|
| `market` | `ORDER_TYPE_MARKET` |
| `limit` | `ORDER_TYPE_LIMIT` |
| `stop-loss` | `ORDER_TYPE_STOP` |
| `stop-loss-limit` | `ORDER_TYPE_STOP_LIMIT` |
| `take-profit` | `ORDER_TYPE_TAKE_PROFIT` |
| `take-profit-limit` | `ORDER_TYPE_TAKE_PROFIT_LIMIT` |
| `trailing-stop` | `ORDER_TYPE_TRAILING_STOP` |
| `trailing-stop-limit` | `ORDER_TYPE_TRAILING_STOP_LIMIT` |
| `limit-maker` | `ORDER_TYPE_LIMIT_MAKER` |
| `iceberg`, `settle-position` | `ORDER_TYPE_MARKET` (closest match) |
| unknown | `ORDER_TYPE_UNKNOWN` + warn |

### Wallet resolution

Source/destination account references are the pair's raw Kraken `base`/`quote` codes (e.g. `XXBT`, `ZUSD`) — exactly the per-variant reference `fetch_accounts` emits. This needs no in-process asset cache, no DB `AccountLookup`, and no per-account fan-out, which is why `fetch_orders` can be a standalone root. If the spot account isn't currently held the reference can dangle; refs are optional/best-effort.

### 8.5 Adjustment granularity — design choice

> **TL;DR — adjustments are recorded per polling-cycle state change, not per individual fill.** An order whose execution touches 1000 trades will typically land in Formance as 1–3 adjustments, not 1000. The per-fill txid list is preserved on every adjustment via `metadata."com.krakenpro.spec/fills"` and the verbatim Kraken payload (including `trades: [...]`) is on `adjustment.raw`. This matches `coinbaseprime` and `bitstamp` and is what the engine's adjustment dedup contract is built for.

#### Why polling-cycle granularity

The engine creates an `OrderAdjustment` per *new* `(reference, status, baseQuantityFilled, fee)` tuple it observes ([`internal/models/orders.go::OrderAdjustmentID`](../../../internal/models/orders.go)). Our connector emits the **cumulative** order state at each polling cycle (`ClosedOrders` already aggregates fills server-side). Because we only poll ClosedOrders, an order is observed once it closes; intermediate OPEN/PARTIALLY_FILLED snapshots are not captured (deferred — see below). So:

| Order lifecycle observed via polling | Adjustments produced |
|---|---|
| Order created + fully filled inside a single cycle window | **1** (only the terminal state is observed) |
| PENDING → OPEN observed → FILLED observed across 3 cycles | **3** |
| OPEN with `vol_exec` growing 0.1 → 0.4 → 1.0 across 3 cycles | **3** |
| 50 fills land in one 20-minute cycle window | **1** (cumulative; intermediate fills never observed) |

This is a deliberate match with the rest of the Formance crypto-exchange connector family (`coinbaseprime`, `bitstamp`) — they all use cumulative state from a list-orders endpoint, not per-fill replay.

#### Why we don't synthesise per-fill adjustments

A connector that wanted one adjustment per fill would have to:

1. Keep using `OpenOrders` / `ClosedOrders` for order discovery + defining attributes (pair, ordertype, descr.price, …).
2. For every order, additionally walk `TradesHistory` (or `QueryOrders` with `trades: true`) chronologically.
3. Maintain per-order seen-fills state to avoid re-emitting fills already observed.
4. Emit one PSPOrder per fill carrying the cumulative state at that point.

Trade-offs we walked away from:

| Concern | Today | Per-fill mode |
|---|---|---|
| Adjustments per high-frequency order (OX3SMX = 1025 fills) | 1 | 1025 |
| Per-order state | none beyond a single watermark | needs persisted seen-fills set, grows unboundedly |
| Engine outbox / workflow events | one per state change | one per fill — 1000× volume on busy orders |
| Coinbaseprime / bitstamp parity | matches | diverges |
| Per-fill price / vol / fee detail | recoverable on demand via `QueryTrades` against the txid list in metadata | always materialised in storage |
| Use case fit | reporting + reconciliation by order | audit-grade per-fill ledger inside Formance |

The per-fill detail is **never lost** — `adjustment.metadata."com.krakenpro.spec/fills"` always holds the full txid list and `adjustment.raw.trades` carries the same list verbatim from Kraken. Drilling down into a specific fill is a single `QueryTrades` call (50 txids per request) away. We chose not to pre-materialise that into storage.

#### Future: WebSocket streaming will change this

When this connector graduates from REST polling to Kraken's Spot WebSocket `executions` channel ([docs.kraken.com](https://docs.kraken.com/api/docs/websocket-v2/executions/)), each individual fill arrives as a discrete event in real time. The orchestrator will naturally emit one PSPOrder per event — the engine's dedup then records one adjustment per fill, because each event carries a distinct cumulative `(status, baseFilled, fee)` tuple. No further refactor on the engine side, no compromise on the mapper: the WS feed makes per-fill adjustments the by-default behaviour, with the polling-cycle approximation becoming a fallback for environments where WS isn't available.

Until that day, the recipe to recover per-fill detail from the current shape is:

```bash
# Pull the txid list out of an adjustment's metadata
TXIDS=$(curl -sS "http://localhost:8080/v3/orders/$OID" \
        | jq -r '.data.adjustments[-1].metadata."com.krakenpro.spec/fills"')

# Resolve them via Kraken (50 at a time)
/tmp/kraken.sh private QueryTrades "{\"txid\":\"$TXIDS\"}"
```

---

## 9. Capability: `FETCH_CONVERSIONS`

Same `POST /0/private/Ledgers` stream as §7, filtered to the full conversion-type set. Spot accounts on Kraken use `conversion` (Instant Buy/Sell) and `sale`; margin accounts can also surface `marginconversion`, `margin_conversion`, and the derivatives-specific variants below. The classifier is exhaustive across all 8 conversion types observed in Kraken's OpenAPI spec, even though spot-only accounts (the design target per ticket EN-1014) will only encounter the first four:

| K `type` | Where it occurs |
|---|---|
| `conversion` | Pro Convert / Mobile Buy / Sell (asset → asset, off-orderbook) |
| `sale` | Sale of staked rewards / NFT proceeds |
| `marginconversion`, `margin_conversion` | Margin position close-out conversion |
| `derivativesflexconversion`, `derivativestaxconversion`, `derivativesconversioncredit`, `collateralconversion` | Derivatives / futures lifecycle conversions — out of scope for spot-only, but classified for exhaustiveness so a margin-enabled account won't fall through to `PAYMENT_TYPE_OTHER` |

A conversion is a **pair** of ledger rows sharing one `refid`: one negative-amount leg (source asset) and one positive-amount leg (destination asset). A pre-pass filters the page to known-asset conversion rows — if any asset is missing it forces one cache refresh, then drops rows still unknown so they never enter the persisted `Pending` map. The pairing step then buffers unmatched (known-asset) legs by `refid`.

The two legs are written for the same conversion event: their `time` values are effectively identical (measured on UAT: max delta ~1µs across paired ledger rows), which is why `CreatedAt` takes `max(time)` — a defensive tie-break for that sub-microsecond jitter. Because the legs' times are essentially equal, they fall in the same frozen pagination window `(Watermark, End]` and the ofs drain surfaces both before the watermark promotes. Pruning correctness therefore rests on **same-window membership**, not on byte-identical timestamps. A buffered leg is pruned only once the watermark has moved a full `pendingPruneGrace` (300s) beyond its time — far more than the observed delta, so a delayed partner still pairs, while `Pending` stays bounded.

**Field mapping** (per resolved pair):

| K | F field | Notes |
|---|---|---|
| `refid` | `Reference` | shared by both legs |
| `max(time over the two legs)` | `CreatedAt` | UTC |
| negative leg `asset → NormalizeAsset` + precision | `SourceAsset` | |
| positive leg `asset → NormalizeAsset` + precision | `DestinationAsset` | |
| `abs(negative leg amount)` × 10^src_precision | `SourceAmount` | gross |
| `positive leg amount` × 10^dst_precision | `DestinationAmount` | gross |
| `sum(fee over both legs)` at dest precision | `Fee` | conversions accrue fees on either leg |
| dest asset | `FeeAsset` | |
| `models.PAYMENT_STATUS_SUCCEEDED` | `Status` | ledger entries are settled |
| see §9 (metadata) | `Metadata` | both ledger ids, `kraken_type`, `subtype` |
| both rows marshalled together | `Raw` | for replay |

If three or more ledger rows arrive sharing the same `refid` (e.g. an extra fee row), the resolver requires exactly one negative + one positive and tags surplus rows in the warn-log + metadata; this is not expected on spot-only.

---

## 10. Metadata

All metadata keys are namespaced `com.krakenpro.spec/`. Per-primitive:

| Key | Source | Capabilities |
|---|---|---|
| `source_ledger_id`, `destination_ledger_id` | row map keys | conversions (payments use the ledger id as the `Reference`, not metadata) |
| `refid` | `refid` | payments, conversions |
| `kraken_type` | `type` | payments, conversions |
| `subtype` | `subtype` | payments, conversions |
| `aclass` | `aclass` | payments, conversions |
| `balance_after` | `balance` | payments, conversions |
| `fee` | `fee` | payments |
| `wallet_type` | class label: `spot` / `staked` / `rewards` / … | accounts |
| `pair` | `pair` | orders |
| `ws_name` | `wsname` (from AssetPairs cache) | orders |
| `fills` | comma-separated list of fill txids | orders |
| `ordertype` | `ordertype` | orders |
| `price_asset` | `<QUOTE>/<dynamicPrecision>` | orders |

The orchestrator passes the raw envelope into `PSPAccount.Raw` / `PSPPayment.Raw` / `PSPOrder.Raw` / `PSPConversion.Raw` for downstream debugging.

---

## 11. Error policy

| Kraken error | Action | Severity | Notes |
|---|---|---|---|
| `EAPI:Invalid key`, `EAPI:Invalid signature`, `EAPI:Bad request`, `EGeneral:Permission denied`, `EGeneral:Unknown method` | wrap `models.ErrInvalidRequest` so Temporal stops retrying (`client.IsFatalAuthError` in [`client/error.go`](client/error.go)) | fatal | a revoked key / permission change can't self-heal, so retrying only wastes cycles |
| `EAPI:Invalid nonce` | wrapped as `httpwrapper.ErrStatusCodeTooEarly` → retry with backoff (**not** fatal) | — | stateless `UnixNano` nonces can momentarily collide under concurrency / cross-pod; the next attempt sends a fresh higher nonce (see the "Nonce & multiple workers" note). The backoff matters because too many invalid-nonce errors temporarily lock the key |
| `EAPI:Rate limit exceeded`, `EService:Throttled[: ts]` | wrapped as `httpwrapper.ErrStatusCodeTooManyRequests` → `plugins.ErrUpstreamRatelimit` | — | Kraken signals rate limits in the error array (often on HTTP 200), so `client.apiError` detects the code and maps it to the platform rate-limit/retry path rather than a generic retry |
| `EOrder:*`, `EQuery:Unknown asset pair` | log + skip the offending row | info | per-row error doesn't fail the cycle |
| HTTP 429 / 5xx | bubble up retryable | warn | `httpwrapper` already retries with backoff |

`error: []` with a non-empty `result` is the success case. `error: [...]` overrides the response, even when 200.

---

## 12. Install behaviour

`Install(ctx, _)` does **no network I/O** — it only registers the periodic workflow, so install stays fast. Validation is deferred:

1. `New` already rejects malformed config (missing fields, non-base64 secret).
2. The asset caches (`/0/public/Assets`, `/0/public/AssetPairs`, 24h TTL) lazy-load on the first fetch via `ensureAssets`; a stale cache that misses a newly-listed asset is force-refreshed and retried by the ledger orchestrators before the watermark advances (see §6/§9).
3. A bad-but-well-formed key surfaces on the first poll as a fatal-auth error, which the `FetchNext*` wrappers map to a non-retryable `models.ErrInvalidRequest` (`EAPI:Invalid nonce` stays retriable — see §2).

---

## 13. CE / EE separation

- Directory: `ee/plugins/krakenpro/`.
- Build: `-tags ee` (every test file must be importable under `-tags ee`).
- Registered in `internal/connectors/plugins/registry/enterprise_ce.go` → `EnterpriseOnlyPlugins["krakenpro"] = struct{}{}`.
- CE binaries return `ErrPluginEnterpriseOnly` for `krakenpro`.

---

## 14. Deferred / future work

| Item | Why deferred | Resolution path |
|---|---|---|
| Per-tier maker/taker fee schedule application | Not needed for read-only sync (fee is already per-row on ledger / cumulative on order) | n/a |
| Open / partially-filled order snapshots | `OpenOrders` accepts no page-size limit, so a drain could exceed Temporal's max activity payload. We poll only the page-bounded `ClosedOrders`, so an order is observed once it closes (final cumulative state); in-flight OPEN/PARTIALLY_FILLED states aren't captured. | If Kraken adds an open-orders page limit, reintroduce a bounded OpenOrders drain; or pick it up via the WebSocket `executions` migration below. |
| Per-fill `OrderAdjustment` granularity | Today we emit one adjustment per polling-cycle state change (§8.5). Materialising 1 adjustment per fill would 1000× the workflow-event volume on high-frequency orders and diverges from coinbaseprime / bitstamp parity. Per-fill detail is still recoverable via `metadata.com.krakenpro.spec/fills` + `QueryTrades`. | Naturally resolved by the WebSocket migration below — each `executions` event becomes one PSPOrder emission, the engine then records one adjustment per fill with no orchestrator-side bookkeeping. |
| **Spot WebSocket `executions` channel** ([docs](https://docs.kraken.com/api/docs/websocket-v2/executions/)) | REST polling is sufficient for the read-only spot use case in EN-1014 and matches the rest of the connector family. WS streaming would obsolete the polling-cycle approximation, surface PENDING → OPEN → PARTIALLY_FILLED → FILLED transitions in real time, and naturally give per-fill adjustment granularity. | Future ticket — the orchestrator would split into a streaming consumer (for live execution events) and the current REST path (for backfill + reconciliation). The mapper layer in `mappers/order.go` already produces the right PSPOrder shape per event, so the change is concentrated in the orchestrator. |
| Trailing-stop absolute trigger / limit prices | For `trailing-stop` / `trailing-stop-limit`, `descr.price`/`descr.price2` are relative offsets (signed/percent), so they're not mapped to `StopPrice`/`LimitPrice` (both left nil) to avoid emitting an offset as an absolute price. The resolved absolute values live in Kraken's top-level `stopprice`/`limitprice` fields, which `OrderEntry` does not currently decode. | Add `stopprice`/`limitprice` to `client.OrderEntry` and map them to `StopPrice`/`LimitPrice` for the trailing variants (and optionally as the source of truth for the fixed stop/take-profit variants). |
| Webhook signature scheme | Out of scope per epic EN-715 (read-only) | future ticket |
| Per-fill `price`/`vol`/`fee` detail | The `trades: [...]` array on each order is txids only. Full fill-row detail needs a separate `QueryTrades` call (50 txids per request) | On-demand via the operator using the txid list in `adjustment.metadata."com.krakenpro.spec/fills"`. Could be batched into the connector in a follow-up if a consumer needs it pre-materialised. |
