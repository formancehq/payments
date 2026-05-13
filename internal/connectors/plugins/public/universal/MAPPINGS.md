# Universal Connector — PSP ↔ Formance Mappings

Co-located mapping reference per the Formance connector skill (Phase 4
checkpoint). The Universal Connector is unique in that the "PSP" is itself
the contract at [`contract/universal-openapi.yaml`](contract/universal-openapi.yaml);
each row below maps a contract field to the Formance PSP type the engine
actually consumes.

## Config

| Config field           | Type          | Required | Notes                                                                                                                              |
|------------------------|---------------|----------|------------------------------------------------------------------------------------------------------------------------------------|
| `endpoint`             | string (URL)  | yes      | Base URL of the counterparty implementing universal-openapi.yaml. Must be HTTPS in production.                                     |
| `apiKey`               | string        | yes      | Bearer token. Sent on every request as `Authorization: Bearer <key>`. Never logged.                                                |
| `webhookSharedSecret`  | string        | optional | HMAC-SHA256 secret for verifying inbound webhooks. **Required** when the counterparty advertises `features.webhookSignature == "hmac-sha256"`. |
| `pollingPeriod`        | duration str. | optional | Defaults to `sharedconfig.DefaultPollingPeriod` (30m). Floor `sharedconfig.MinimumPollingPeriod` (20m).                             |
| `capabilityOverrides`  | string array  | optional | Allow-list to **narrow** the counterparty-declared set. Validated against the `Capability` enum.                                   |

## Pagination

The counterparty announces its strategy in `GET /v1/capabilities`'s
`features.pagination`:

| Mode     | State key sent on next call           | Plugin behaviour                                              |
|----------|----------------------------------------|---------------------------------------------------------------|
| `cursor` | `nextCursor` from previous response    | Sends `?cursor=…` until `hasMore=false`.                      |
| `page`   | `pageNumber` (1-based, local)          | Sends `?page=N&pageSize=…`.                                   |
| `none`   | n/a                                    | Single fetch per poll cycle; engine dedups via PSP references. |

`updatedAtFrom` is **always** sent when state has it, regardless of the
chosen strategy — so cursor/page can be combined with incremental polling.

## Capability discovery

| /v1/capabilities `supported[]`   | Engine `Capability`                            | Schedules                       |
|----------------------------------|------------------------------------------------|---------------------------------|
| `FETCH_ACCOUNTS`                 | `CAPABILITY_FETCH_ACCOUNTS`                    | `TASK_FETCH_ACCOUNTS` periodic  |
| `FETCH_BALANCES`                 | `CAPABILITY_FETCH_BALANCES`                    | `TASK_FETCH_BALANCES` periodic (chained under accounts) |
| `FETCH_EXTERNAL_ACCOUNTS`        | `CAPABILITY_FETCH_EXTERNAL_ACCOUNTS`           | `TASK_FETCH_EXTERNAL_ACCOUNTS` periodic |
| `FETCH_PAYMENTS`                 | `CAPABILITY_FETCH_PAYMENTS`                    | `TASK_FETCH_PAYMENTS` periodic  |
| `FETCH_OTHERS`                   | `CAPABILITY_FETCH_OTHERS`                      | `TASK_FETCH_OTHERS` periodic    |
| `FETCH_ORDERS`                   | `CAPABILITY_FETCH_ORDERS`                      | `TASK_FETCH_ORDERS` periodic; bootstraps accounts first |
| `FETCH_CONVERSIONS`              | `CAPABILITY_FETCH_CONVERSIONS`                 | `TASK_FETCH_CONVERSIONS` periodic; bootstraps accounts first |
| `CREATE_WEBHOOKS`                | `CAPABILITY_CREATE_WEBHOOKS`                   | `TASK_CREATE_WEBHOOKS` one-shot at install |
| `TRANSLATE_WEBHOOKS`             | `CAPABILITY_TRANSLATE_WEBHOOKS`                | Enables `TranslateWebhook` runtime guard |
| `CREATE_BANK_ACCOUNT`            | `CAPABILITY_CREATE_BANK_ACCOUNT`               | Enables `CreateBankAccount` runtime guard |
| `CREATE_TRANSFER`                | `CAPABILITY_CREATE_TRANSFER`                   | Enables `CreateTransfer` / poll / reverse |
| `CREATE_PAYOUT`                  | `CAPABILITY_CREATE_PAYOUT`                     | Enables `CreatePayout` / poll / reverse |
| `ALLOW_FORMANCE_ACCOUNT_CREATION`| `CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION`   | Engine permits accounts created in Formance UI/API |
| `ALLOW_FORMANCE_PAYMENT_CREATION`| `CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION`   | Engine permits payments created in Formance UI/API |

Anything not declared returns `plugins.ErrNotImplemented` from the
corresponding plugin method (engine maps to a non-retryable terminal
failure in the Temporal activity).

## Per-primitive field mapping

Every full schema is in [`contract/data-model.md`](contract/data-model.md);
the table below summarises the wire→PSP key fields and what's optional.

### Account → `models.PSPAccount`

| Wire field      | PSP field      | Notes                                                |
|-----------------|----------------|------------------------------------------------------|
| `reference`     | `Reference`    | Required.                                            |
| `createdAt`     | `CreatedAt`    | Required.                                            |
| `name`          | `Name`         | Optional.                                            |
| `defaultAsset`  | `DefaultAsset` | Optional. UMN string.                                |
| `metadata`      | `Metadata`     | Optional, namespace per `com.universal.spec/...`.    |
| (entire body)   | `Raw`          | JSON-marshalled wire object stored verbatim.         |

### Balance → `models.PSPBalance`

| Wire field          | PSP field          | Notes                              |
|---------------------|--------------------|------------------------------------|
| `accountReference`  | `AccountReference` | Required, matches an `Account.reference`. |
| `createdAt`         | `CreatedAt`        | Required.                          |
| `amount`            | `Amount`           | Decimal-string minor units; parsed to `*big.Int`. |
| `asset`             | `Asset`            | UMN string.                        |

### Payment → `models.PSPPayment`

| Wire field                    | PSP field                     | Notes                                                                                    |
|-------------------------------|-------------------------------|------------------------------------------------------------------------------------------|
| `reference`                   | `Reference`                   | Required.                                                                                |
| `parentReference`             | `ParentReference`             | Optional; refunds/captures point at the original payment.                                |
| `createdAt`                   | `CreatedAt`                   | Required.                                                                                |
| `updatedAt`                   | (drives engine adjustment poll) | Strictly monotonic per record — see [`adjustments.md`](contract/adjustments.md).         |
| `type`                        | `Type`                        | `PAYIN | PAYOUT | TRANSFER | OTHER`. Unknown → `OTHER`.                                  |
| `status`                      | `Status`                      | Enum from `models.PaymentStatus`. Unknown → `OTHER`.                                     |
| `scheme`                      | `Scheme`                      | Enum from `models.PaymentScheme`. Unknown / empty → `OTHER`.                             |
| `amount`                      | `Amount`                      | `*big.Int` minor units.                                                                  |
| `asset`                       | `Asset`                       | UMN string.                                                                              |
| `sourceAccountReference`      | `SourceAccountReference`      | Optional.                                                                                |
| `destinationAccountReference` | `DestinationAccountReference` | Optional.                                                                                |
| `metadata`                    | `Metadata`                    | Optional.                                                                                |
| (entire body)                 | `Raw`                         | JSON-marshalled wire object stored verbatim.                                             |

### Order → `models.PSPOrder`

| Wire field                    | PSP field                     | Notes                                                                |
|-------------------------------|-------------------------------|----------------------------------------------------------------------|
| `reference`                   | `Reference`                   | Required.                                                            |
| `clientOrderID`               | `ClientOrderID`               | Optional traceability key.                                           |
| `createdAt`                   | `CreatedAt`                   | Required.                                                            |
| `updatedAt`                   | (drives adjustment dedup)     | See [`adjustments.md`](contract/adjustments.md).                     |
| `direction`                   | `Direction`                   | `BUY | SELL`. Required.                                              |
| `type`                        | `Type`                        | `MARKET | LIMIT | STOP_LIMIT`. Required.                             |
| `status`                      | `Status`                      | Enum from `models.OrderStatus`.                                      |
| `sourceAsset`/`destinationAsset` | `SourceAsset`/`DestinationAsset` | UMN strings; meaning depends on `Direction`.                         |
| `baseQuantityOrdered`         | `BaseQuantityOrdered`         | Required, `*big.Int` base-asset minor units.                         |
| `baseQuantityFilled`          | `BaseQuantityFilled`          | Optional; bumping it generates a new adjustment.                     |
| `limitPrice`/`stopPrice`      | `LimitPrice`/`StopPrice`      | Optional, `*big.Int` quote-asset minor units.                        |
| `timeInForce`                 | `TimeInForce`                 | `GTC | IOC | FOK | GTD`.                                             |
| `expiresAt`                   | `ExpiresAt`                   | Optional, only meaningful for `GTD`.                                 |
| `quoteAmount`/`quoteAsset`    | `QuoteAmount`/`QuoteAsset`    | Optional, exposed for analytics.                                     |
| `fee`/`feeAsset`              | `Fee`/`FeeAsset`              | Optional, fee in quote currency by default.                          |
| `averageFillPrice`/`priceAsset`| `AverageFillPrice`/`PriceAsset` | Optional, analytics-only.                                          |
| `sourceAccountReference`/`destinationAccountReference` | `SourceAccountReference`/`DestinationAccountReference` | Optional, account links resolved by `AccountLookup`. |
| (entire body)                 | `Raw`                         | JSON-marshalled wire object stored verbatim.                         |

### Conversion → `models.PSPConversion`

| Wire field                    | PSP field                     | Notes                                                              |
|-------------------------------|-------------------------------|--------------------------------------------------------------------|
| `reference`                   | `Reference`                   | Required.                                                          |
| `createdAt`                   | `CreatedAt`                   | Required.                                                          |
| `status`                      | `Status`                      | `PENDING | COMPLETED | FAILED`.                                    |
| `sourceAsset`/`destinationAsset` | `SourceAsset`/`DestinationAsset` | UMN strings.                                                       |
| `sourceAmount`/`destinationAmount` | `SourceAmount`/`DestinationAmount` | `*big.Int` minor units.                                            |
| `fee`/`feeAsset`              | `Fee`/`FeeAsset`              | Optional.                                                          |
| `sourceAccountReference`/`destinationAccountReference` | `SourceAccountReference`/`DestinationAccountReference` | Optional.                                                          |
| (entire body)                 | `Raw`                         | JSON-marshalled wire object stored verbatim.                       |

### Other → `models.PSPOther`

| Wire field | PSP field | Notes                                                       |
|------------|-----------|-------------------------------------------------------------|
| `id`       | `ID`      | Required.                                                   |
| `data`     | `Other`   | Free-form JSON, marshalled to `json.RawMessage` untouched.  |

### Payout / Transfer initiation

Request body comes from `models.PSPPaymentInitiation`:

| Wire field                    | Source                                        |
|-------------------------------|-----------------------------------------------|
| `reference`                   | `pi.Reference`                                |
| `description`                 | `pi.Description`                              |
| `amount`                      | `pi.Amount.String()`                          |
| `asset`                       | `pi.Asset`                                    |
| `sourceAccountReference`      | `pi.SourceAccount.Reference`                  |
| `destinationAccountReference` | `pi.DestinationAccount.Reference`             |
| `metadata`                    | `pi.Metadata`                                 |

Idempotency-Key header MUST be `pi.Reference`.

Response interpretation:

| `mode`     | Plugin → engine                                                               |
|------------|-------------------------------------------------------------------------------|
| `terminal` | `Payment` translated via `toPSPPayment`, returned in `CreatePayoutResponse.Payment`. |
| `polling`  | `pollingID` returned in `CreatePayoutResponse.PollingPayoutID`; engine schedules `PollPayoutStatus`. |
| `error`    | Wrapped as `models.ErrInvalidRequest` and surfaced to the engine.             |

### Bank account creation

Request body fields from `models.BankAccount`:

| Wire field      | Source              |
|-----------------|---------------------|
| `id`            | `ba.ID.String()`    |
| `createdAt`     | `ba.CreatedAt`      |
| `name`          | `ba.Name`           |
| `accountNumber` | `ba.AccountNumber`  |
| `iban`          | `ba.IBAN`           |
| `swiftBicCode`  | `ba.SwiftBicCode`   |
| `country`       | `ba.Country`        |
| `metadata`      | `ba.Metadata`       |

Response `relatedAccount` is mapped via `toPSPAccount`.

### Webhooks

Subscription request fields per event type:

| Wire field    | Source                                                       |
|---------------|--------------------------------------------------------------|
| `name`        | one of `supportedWebhooks` keys (e.g. `payment.updated`)     |
| `callbackUrl` | `req.WebhookBaseUrl + supportedWebhooks[name]`               |

Idempotency-Key for subscription creation: `<connectorID>:<eventType>`.

Inbound delivery envelope ⇢ `WebhookResponse`: see
[`universal-events.md`](contract/universal-events.md).

## Metadata namespace

Plugin emits and accepts metadata under the `com.universal.spec/` namespace
to match the convention every other Formance connector follows.

## CLAUDE.md compliance check

- Amounts are gross (PSP fees not subtracted). The contract's `fee` field
  on Order/Conversion is informational only.
- IDs are raw — engine namespacing happens at the wrapper level, not in
  the plugin.
- Status enums map to canonical Formance constants; unknowns degrade to
  `OTHER`, never errors.
- Every PSP entity carries the wire payload in `Raw` for downstream
  audit / replay.
