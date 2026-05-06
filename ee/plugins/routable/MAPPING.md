# Routable ↔ Formance Payments — field mapping

Authoritative reference for how the dedicated Routable EE connector translates Routable API objects into Formance PSP types and back.

This document is the source of truth for connector reviewers, integrators, and operators tracing a Formance Payment back to its Routable origin (or vice-versa). It is co-located with the code so the mapping table and the implementation do not drift.

> Symbols used in this doc
>
> - **R** — a Routable API field (REST/JSON, `snake_case`)
> - **F** — a Formance PSP field (Go struct, `CamelCase`)
> - `→` — write direction (Routable → Formance, sync path)
> - `←` — write direction (Formance → Routable, payout/transfer initiation)

---

## 1. Connector configuration

Defined in [`config.go`](config.go), exposed through [`/openapi/v3/v3-connectors-config.yaml`](../../../openapi/v3/v3-connectors-config.yaml) as `V3RoutableConfig`.

| Field | Required | Default | Purpose |
|---|---|---|---|
| `apiKey` | yes | — | Routable Bearer token, sent as `Authorization: Bearer <apiKey>` on every request. |
| `endpoint` | no | `https://api.routable.com` | API root. Use `https://api.sandbox.routable.com` for the sandbox. |
| `actingTeamMember` | **no** | `""` | Default Routable team member ID for `POST /v1/payables`. Optional at the connector level: callers may override (or supply) it per-request via the metadata key [`com.routable.spec/acting_team_member`](#5-payment-initiation-metadata-keys-payouts--transfers). If neither config nor metadata sets it, payable creation fails with a clear validation error before the request is sent. |
| `pollingPeriod` | no | `30m` | Polling cadence for sync tasks (accounts, balances, external accounts, payments). Minimum 20 minutes. |

---

## 2. Capabilities and workflow

Declared in [`capabilities.go`](capabilities.go) and [`workflow.go`](workflow.go).

| Capability | Routable endpoint(s) | Triggered by |
|---|---|---|
| `CAPABILITY_FETCH_ACCOUNTS` | `GET /v1/settings/accounts` | Periodic `TASK_FETCH_ACCOUNTS` |
| `CAPABILITY_FETCH_BALANCES` | `GET /v1/settings/accounts/{id}` | Periodic `TASK_FETCH_BALANCES` (downstream of accounts) |
| `CAPABILITY_FETCH_EXTERNAL_ACCOUNTS` | `GET /v1/companies` | Periodic `TASK_FETCH_EXTERNAL_ACCOUNTS` |
| `CAPABILITY_FETCH_PAYMENTS` | `GET /v1/payables` + `GET /v1/receivables` | Periodic `TASK_FETCH_PAYMENTS` |
| `CAPABILITY_CREATE_TRANSFER` | `POST /v1/payables` | `CreateTransfer` engine workflow |
| `CAPABILITY_CREATE_PAYOUT` | `POST /v1/payables` | `CreatePayout` engine workflow |

Webhooks (`CREATE_WEBHOOKS`, `TRANSLATE_WEBHOOKS`) and `CreateBankAccount` / `ReverseTransfer` / `ReversePayout` are deliberately deferred to follow-up PRs and inherit `ErrNotImplemented` from [`base_plugin.go`](../../../internal/connectors/plugins/base_plugin.go).

---

## 3. Sync mappings (Routable → Formance)

### 3.1 Settings account → `PSPAccount` (internal)

Implemented in [`accounts.go`](accounts.go) (`settingsAccountToPSPAccount`).

| F — `models.PSPAccount` | R — `Account` | Notes |
|---|---|---|
| `Reference` | `id` | Stable Routable identifier (e.g. `acc_…`). |
| `Name` | `name` | Set when non-empty. |
| `CreatedAt` | `created_at` | Forwarded as-is (Routable always populates it on settings accounts). |
| `DefaultAsset` | `currency_code` | Formatted via [`go-libs/v3/currency.FormatAsset`](https://github.com/formancehq/go-libs) (e.g. `USD/2`). Omitted when Routable returns no currency. |
| `Metadata` | `object`, `type`, `is_valid`, `currency_code`, `type_details.{account_type,bank_name,account_number,routing_number}` | All keys namespaced under `com.routable.spec/`. See [`metadata.go`](metadata.go) → `settingsAccountMetadata`. |
| `Raw` | full JSON | Verbatim Routable response, kept for forensics. |

### 3.2 Settings account → `PSPBalance`

Implemented in [`balances.go`](balances.go) (`accountToBalance`).

| F — `models.PSPBalance` | R — `Account.type_details` | Notes |
|---|---|---|
| `AccountReference` | `id` | Same reference as the parent `PSPAccount`. |
| `Asset` | `currency_code` | Defaults to `USD` when absent (Routable historically omits it on balance-only accounts). |
| `Amount` | `available_amount` | Decimal string converted to minor units via [`amounts.go`](amounts.go) (`toMinorUnits`, half-up rounding). |
| `CreatedAt` | `time.Now().UTC()` | Routable does not expose a `balance_updated_at`; we stamp the read time. |

> Pending balances are not surfaced as a second `PSPBalance`. The Formance balance model represents one snapshot per `(account, asset)` pair, and Routable's `available_amount` is the canonical operational signal. If you need the pending figure, it is preserved in `PSPAccount.Metadata` under `com.routable.spec/...` keys.

### 3.3 Company → `PSPAccount` (external)

Implemented in [`external_accounts.go`](external_accounts.go) (`companyToPSPAccount`).

| F — `models.PSPAccount` | R — `Company` | Notes |
|---|---|---|
| `Reference` | `id` | Routable company ID (e.g. `co_…`). Used as the `DestinationAccountReference` on payables. |
| `Name` | `display_name` ?? `business_name` | Falls back to `business_name` when `display_name` is empty. |
| `CreatedAt` | `created_at` | Forwarded as-is. |
| `Metadata` | `object`, `type`, `status`, `country_code`, `is_vendor`, `is_customer`, `is_archived`, `external_id`, `business_name`, `display_name`, `registered_address.{line_1,line_2,city,state,postal_code,country}` | See [`metadata.go`](metadata.go) → `companyMetadata`. |
| `Raw` | full JSON | Verbatim Routable response. |

> **No N+1.** Unlike the legacy Generic-Connector adapter (`connector-routable`), we do **not** fan out to `GET /v1/companies/{id}/payment-methods` per row during list. Payment-method resolution happens on demand at payable-creation time only.

### 3.4 Payable → `PSPPayment` (PAYOUT)

Implemented in [`payments.go`](payments.go) (`payableToPSPPayment`).

| F — `models.PSPPayment` | R — `Payable` | Notes |
|---|---|---|
| `Reference` | `id` | Routable payable ID (e.g. `pa_…`). |
| `CreatedAt` | `created_at` | Forwarded as-is. |
| `Type` | constant `PAYMENT_TYPE_PAYOUT` | All Routable payables are money-out flows. Overridden to `PAYMENT_TYPE_TRANSFER` only when the row was created via `CreateTransfer` (see [`transfers.go`](transfers.go)). |
| `Amount` | `amount` × `precision(currency_code)` | Decimal → minor units. Unsupported currencies cause the row to be skipped with a log line. |
| `Asset` | `currency_code` | `USD/2`, `EUR/2`, `KWD/3`, … |
| `Scheme` | `delivery_method` | Mapped via [`scheme.go`](scheme.go) → `deliveryMethodToScheme` (`ach*` → `PAYMENT_SCHEME_ACH`, everything else → `PAYMENT_SCHEME_OTHER`). |
| `Status` | `status` | Mapped via [`status.go`](status.go) → `payableStatus`. See [§4](#4-status-mapping). |
| `SourceAccountReference` | `withdraw_from_account.id` | Routable settings account ID (matches a `PSPAccount` of internal type). |
| `DestinationAccountReference` | `pay_to_company.id` | Routable company ID (matches a `PSPAccount` of external type). |
| `Metadata` | `type`, `delivery_method`, `status`, `external_id`, `memo`, `reference` | See [`metadata.go`](metadata.go) → `payableMetadata`. |
| `Raw` | full JSON | Verbatim Routable response. |

### 3.5 Receivable → `PSPPayment` (PAYIN)

Implemented in [`payments.go`](payments.go) (`receivableToPSPPayment`). Mirror of §3.4 with source/destination flipped.

| F — `models.PSPPayment` | R — `Receivable` | Notes |
|---|---|---|
| `Type` | constant `PAYMENT_TYPE_PAYIN` | |
| `SourceAccountReference` | `pay_from_company.id` | Inbound counterparty (company ID). |
| `DestinationAccountReference` | `deposit_to_account.id` | Settings account ID. |
| All other fields | Same shape as §3.4 (`amount`, `currency_code`, `delivery_method`, `status`, `created_at`, …) | See `receivableMetadata` in [`metadata.go`](metadata.go) for the metadata key set. |

### 3.6 Pagination & state

Defined in [`state.go`](state.go) (`pageState`) and [`payments.go`](payments.go) (`paymentsState`). Persisted opaquely as JSON between fetch cycles.

| Resource | Cursor | Filters sent to Routable |
|---|---|---|
| Settings accounts | `{ page }` (1-indexed) | `page`, `page_size` |
| Companies | `{ page }` | `page`, `page_size` |
| Payments | `{ phase: "" \| "receivables", page, lastSeenAt }` | `page`, `page_size`, `status_changed_at.gte` |

The payments fetcher exhausts payables, then receivables, then advances the `lastSeenAt` watermark (the latest `status_changed_at` observed) and resets to a new cycle. This avoids re-pulling payables and receivables that have not changed since the previous cycle.

---

## 4. Status mapping

Implemented in [`status.go`](status.go) (`payableStatus`). Lifted from the rules already validated in `connector-routable/internal/mapper/status.go`, retargeted to the `models.PaymentStatus` enum.

| Routable `status` | Formance `models.PaymentStatus` |
|---|---|
| `draft`, `ready_to_send`, `pending`, `scheduled`, `initiated`, `processing`, `in_transit`, `awaiting_delivery` | `PAYMENT_STATUS_PENDING` |
| `completed`, `paid`, `delivered` | `PAYMENT_STATUS_SUCCEEDED` |
| `failed`, `returned`, `nsf` | `PAYMENT_STATUS_FAILED` |
| `stopped`, `canceled`, `cancelled`, `voided` | `PAYMENT_STATUS_CANCELLED` |
| `expired` | `PAYMENT_STATUS_EXPIRED` |
| anything else (or empty) | `PAYMENT_STATUS_UNKNOWN` |

Comparison is case-insensitive and trims whitespace.

`isTerminalStatus` in the same file controls when [`PollPayoutStatus`/`PollTransferStatus`](payouts.go) stop polling — it returns true for `SUCCEEDED`, `FAILED`, `CANCELLED`, `EXPIRED`, and `REFUNDED`.

---

## 5. Payment initiation metadata keys (payouts + transfers)

`CreatePayout` and `CreateTransfer` translate a `PSPPaymentInitiation` (defined in [`internal/models/payment_initiations.go`](../../../internal/models/payment_initiations.go)) into a Routable `POST /v1/payables` body. The implementation lives in [`payouts.go`](payouts.go) (`initiatePayable`).

Most of the request body is derived from the structured `PSPPaymentInitiation` fields. A small set of Routable-specific knobs is exposed via metadata; every key is namespaced under `com.routable.spec/`.

### 5.1 Routable-specific metadata keys

All keys are defined as constants in [`metadata.go`](metadata.go) so producers can reference them without string typos.

| Metadata key | Const | Required | Default | Maps to Routable field | Purpose |
|---|---|---|---|---|---|
| `com.routable.spec/type` | `MetadataKeyType` | no | `ach` | `type` | Payable rail (`ach`, `wire`, `check`, `international`, `external`, `vendor_choice`). |
| `com.routable.spec/delivery_method` | `MetadataKeyDeliveryMethod` | no | `ach_standard` | `delivery_method` | Specific delivery option (`ach_standard`, `ach_same_day`, `wire`, `check`, …). Must be compatible with `type`. |
| `com.routable.spec/acting_team_member` | `MetadataKeyActingTeamMember` | conditional¹ | config `actingTeamMember` | `acting_team_member` | Routable team member ID initiating the payable. |
| `com.routable.spec/external_id` | `MetadataKeyExternalID` | no | `""` | `external_id` | Caller-supplied external reference (idempotent lookup key on Routable's side). |
| `com.routable.spec/memo` | `MetadataKeyMemo` | no | `PSPPaymentInitiation.Description` | `memo` | Free-form note shown to the recipient. |
| `com.routable.spec/line_item_description` | `MetadataKeyLineDescription` | no | `PSPPaymentInitiation.Description` | `line_items[0].description` | Description on the auto-generated single-line item. |

¹ `acting_team_member` must be resolvable at request time — either from the connector config, this metadata key, or both. The client validates and returns `create payable: acting_team_member is required` before any HTTP call when neither is set.

### 5.2 Static body fields (always present)

| Routable field | Source |
|---|---|
| `pay_to_company` | `PSPPaymentInitiation.DestinationAccount.Reference` |
| `withdraw_from_account` | `PSPPaymentInitiation.SourceAccount.Reference` |
| `amount` | `PSPPaymentInitiation.Amount` (minor units) → decimal string via `fromMinorUnits` ([`amounts.go`](amounts.go)) |
| `currency_code` | `PSPPaymentInitiation.Asset` (e.g. `USD/2` → `USD`) |
| `line_items` | Single line item with `unit_price = amount = total`, `quantity = 1`, optional description from metadata |
| `reference` | `PSPPaymentInitiation.Reference` (also forwarded as the `Idempotency-Key` HTTP header) |

### 5.3 Idempotency

`PSPPaymentInitiation.Reference` is sent as the `Idempotency-Key` HTTP header to `POST /v1/payables`. Routable returns the original payable on retries with the same key, which is exactly the behaviour the engine's create-then-poll workflow relies on. Unlike the legacy Generic-Connector adapter, we **do not** strip the idempotency key from the response: native Formance plugin paths do not perform the `ParentReference` swap that caused duplicate `Payment` rows in the Generic flow.

### 5.4 Response handling

After `POST /v1/payables` returns:

| Routable response status | Engine response | Behaviour |
|---|---|---|
| Terminal (`completed`, `failed`, `cancelled`, `expired`) | `Payment` populated, no polling ID | Workflow ends immediately. |
| Non-terminal (`pending`, `processing`, …) | `PollingPayoutID` / `PollingTransferID` = Routable payable ID | Engine schedules `PollPayoutStatus`/`PollTransferStatus` against `GET /v1/payables/{id}` until terminal. |

`PollPayoutStatus` (and the shared `pollPayableStatus` it delegates to) treats `404 Not Found` as a transient state (eventual consistency after `202 Accepted`) and asks the engine to retry, instead of failing the workflow. See [`payouts.go`](payouts.go) and [`client/client.go`](client/client.go) (`ErrPayableNotFound`).

---

## 6. Quick-reference index

| Concern | File |
|---|---|
| Plugin shim, `ErrNotYetInstalled` guards | [`plugin.go`](plugin.go) |
| Capabilities matrix | [`capabilities.go`](capabilities.go) |
| Periodic task graph | [`workflow.go`](workflow.go) |
| Config schema (`apiKey`, `endpoint`, `actingTeamMember`, `pollingPeriod`) | [`config.go`](config.go) |
| State / cursors | [`state.go`](state.go), [`payments.go`](payments.go) |
| Settings accounts → `PSPAccount` (internal) | [`accounts.go`](accounts.go) |
| Settings account → `PSPBalance` | [`balances.go`](balances.go) |
| Companies → `PSPAccount` (external) | [`external_accounts.go`](external_accounts.go) |
| Payables/receivables → `PSPPayment` | [`payments.go`](payments.go) |
| Create payout (Routable payable) | [`payouts.go`](payouts.go) |
| Create transfer (Routable payable, type override) | [`transfers.go`](transfers.go) |
| Status mapping | [`status.go`](status.go) |
| Scheme mapping | [`scheme.go`](scheme.go) |
| Decimal ↔ minor units | [`amounts.go`](amounts.go) |
| Metadata key constants & helpers | [`metadata.go`](metadata.go) |
| Routable HTTP client (interface + impl) | [`client/client.go`](client/client.go) |
| Routable request/response shapes | [`client/types.go`](client/types.go) |
| Routable error envelope | [`client/error.go`](client/error.go) |

---

## 7. References

- Routable API docs: <https://developers.routable.com/reference>
- Routable idempotency: <https://developers.routable.com/docs/idempotency-keys>
- Formance PSP types: [`internal/models/`](../../../internal/models/)
  - [`accounts.go`](../../../internal/models/accounts.go) — `PSPAccount`
  - [`balances.go`](../../../internal/models/balances.go) — `PSPBalance`
  - [`payments.go`](../../../internal/models/payments.go) — `PSPPayment`
  - [`payment_initiations.go`](../../../internal/models/payment_initiations.go) — `PSPPaymentInitiation`
  - [`capabilities.go`](../../../internal/models/capabilities.go) — `Capability` enum
  - [`payment_scheme.go`](../../../internal/models/payment_scheme.go) — `PaymentScheme` enum
  - [`payment_status.go`](../../../internal/models/payment_status.go) — `PaymentStatus` enum
- Plugin contract: [`internal/connectors/plugins/plugin.go`](../../../internal/connectors/plugins/plugin.go), [`base_plugin.go`](../../../internal/connectors/plugins/base_plugin.go)
- Registry & EE/CE gating: [`internal/connectors/plugins/registry/`](../../../internal/connectors/plugins/registry/)
