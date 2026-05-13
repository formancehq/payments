# Universal Connector — State Machines (v1)

> Companion: [`data-model.md`](data-model.md),
> [`adjustments.md`](adjustments.md),
> [`universal-events.md`](universal-events.md).

Every stateful primitive in the contract has two execution paths:

- **Polling** — engine-pulled via the periodic Temporal workflows
  registered at install time.
- **Webhook** — counterparty-pushed via the events catalogued in
  [`universal-events.md`](universal-events.md).

Both paths converge on the same engine state: a webhook is a low-latency
shortcut for what the next poll would have surfaced anyway. Adjustment
dedup (see [`adjustments.md`](adjustments.md)) makes both paths
idempotent — the engine never double-counts.

## Install / bootstrap sequence

```mermaid
sequenceDiagram
  participant U as User
  participant API as Payments v3 API
  participant W as Temporal worker
  participant P as Universal plugin
  participant C as Counterparty

  U->>API: POST /v3/connectors/install/Universal {config}
  API->>W: schedule InstallConnector workflow
  W->>P: Install(req)
  P->>C: GET /v1/capabilities
  C-->>P: {supported, features}
  P-->>W: InstallResponse{Workflow: tree}

  alt FETCH_ORDERS or FETCH_CONVERSIONS declared
    W->>W: BootstrapOnInstall = [TASK_FETCH_ACCOUNTS]
    W->>P: FetchNextAccounts(...) until HasMore=false
  end

  W->>W: register periodic schedules per workflow tree
  W->>W: register one-shot CREATE_WEBHOOKS task (if declared)
  W-->>API: connector ready
```

The bootstrap branch matches the canonical pattern in
[`internal/connectors/engine/workflow/install_connector.go`](../../../../engine/workflow/install_connector.go)
lines 116–158. We opt into it only when `FETCH_ORDERS` or
`FETCH_CONVERSIONS` is declared, because those primitives reference
accounts at runtime via `AccountLookup`; we don't want the first poll to
race an empty accounts table.

## Payment lifecycle

`PSPPayment.Status` enum from
[`internal/models/payment_status.go`](../../../../models/payment_status.go).

```mermaid
stateDiagram-v2
  [*] --> PENDING
  PENDING --> AUTHORISATION : auth flow
  PENDING --> SUCCEEDED     : direct success
  PENDING --> FAILED
  PENDING --> CANCELLED
  PENDING --> EXPIRED
  AUTHORISATION --> CAPTURE
  CAPTURE --> SUCCEEDED
  CAPTURE --> CAPTURE_FAILED
  SUCCEEDED --> REFUNDED
  SUCCEEDED --> DISPUTE
  REFUNDED --> REFUND_REVERSED
  REFUNDED --> REFUNDED_FAILURE
  DISPUTE --> DISPUTE_WON
  DISPUTE --> DISPUTE_LOST
  SUCCEEDED --> [*]
  FAILED --> [*]
  CANCELLED --> [*]
  EXPIRED --> [*]
  CAPTURE_FAILED --> [*]
  REFUND_REVERSED --> [*]
  REFUNDED_FAILURE --> [*]
  DISPUTE_WON --> [*]
  DISPUTE_LOST --> [*]
```

Terminal states do not need to disappear from `GET /v1/payments` — the
counterparty SHOULD continue to serve them, just with a frozen
`updatedAt`. The engine's `updatedAtFrom` cursor will skip them on
subsequent polls.

Every transition emits a [`PaymentAdjustment`](../../../../models/payment_adjustments.go).

## Payout / Transfer initiation lifecycle

Engine-side states from
[`PaymentInitiationAdjustmentStatus`](../../../../models/payment_initiation_adjustments_status.go).
The counterparty does not see these directly; they are derived from the
PSP `PaymentStatus` returned on each poll.

```mermaid
stateDiagram-v2
  [*] --> WAITING_FOR_VALIDATION
  WAITING_FOR_VALIDATION --> SCHEDULED_FOR_PROCESSING
  SCHEDULED_FOR_PROCESSING --> PROCESSING
  PROCESSING --> PROCESSED
  PROCESSING --> FAILED
  PROCESSING --> REJECTED
  PROCESSED --> REVERSE_PROCESSING : on POST /v1/.../reverse
  REVERSE_PROCESSING --> REVERSED
  REVERSE_PROCESSING --> REVERSE_FAILED
  PROCESSED --> [*]
  FAILED --> [*]
  REJECTED --> [*]
  REVERSED --> [*]
  REVERSE_FAILED --> [*]
```

### Polling path

```mermaid
sequenceDiagram
  participant E as Engine (Temporal)
  participant P as Universal plugin
  participant C as Counterparty

  E->>P: CreatePayout(initiation)
  P->>C: POST /v1/payouts (Idempotency-Key=initiation.reference)
  C-->>P: {mode: "polling", pollingID: "ext_xyz"}
  P-->>E: CreatePayoutResponse{PollingPayoutID: "ext_xyz"}
  E->>E: schedule PollPayout workflow

  loop until terminal
    E->>P: PollPayoutStatus(pollingID)
    P->>C: GET /v1/payouts/ext_xyz
    C-->>P: {payment: {status: "PENDING", ...}}
    P-->>E: {Payment: nil, Error: nil} (keep polling)
  end

  C-->>P: {payment: {status: "SUCCEEDED", ...}} (eventually)
  P-->>E: {Payment: <terminal>, Error: nil}
  E->>E: write PaymentInitiationAdjustment(PROCESSED)<br/>delete poll schedule
```

Synchronous failure path: any `GET /v1/payouts/{id}` may return `error: "..."`
to drop the poll and write a `FAILED` adjustment. The polling workflow itself
is in [`internal/connectors/engine/workflow/poll_payout.go`](../../../../engine/workflow/poll_payout.go).

### Webhook path (equivalent)

```mermaid
sequenceDiagram
  participant E as Engine (Temporal)
  participant P as Universal plugin
  participant C as Counterparty

  C->>P: POST /webhook/payment/updated (signed)
  P->>P: VerifyWebhook (HMAC-SHA256, ConstantTimeCompare)
  P->>P: TranslateWebhook → WebhookResponse{Payment: <terminal>}
  P-->>E: ack
  E->>E: write PaymentInitiationAdjustment<br/>delete poll schedule (if any)
```

Both paths converge on the same `(reference, status)` adjustment dedup,
so receiving both a webhook AND a successful poll for the same transition
is harmless.

## Order lifecycle

`PSPOrder.Status` enum from
[`internal/models/order_status.go`](../../../../models/order_status.go):

```mermaid
stateDiagram-v2
  [*] --> PENDING
  PENDING --> OPEN
  PENDING --> FAILED
  OPEN --> PARTIALLY_FILLED
  OPEN --> FILLED
  OPEN --> CANCELLED
  OPEN --> EXPIRED
  PARTIALLY_FILLED --> PARTIALLY_FILLED : new fill
  PARTIALLY_FILLED --> FILLED
  PARTIALLY_FILLED --> CANCELLED
  PARTIALLY_FILLED --> EXPIRED
  FILLED --> [*]
  CANCELLED --> [*]
  FAILED --> [*]
  EXPIRED --> [*]
```

Note the self-loop on `PARTIALLY_FILLED`: the adjustment dedup key
includes `BaseQuantityFilled`, so each new fill while staying in the same
status produces a fresh adjustment.

## Conversion lifecycle

```mermaid
stateDiagram-v2
  [*] --> PENDING
  PENDING --> COMPLETED
  PENDING --> FAILED
  COMPLETED --> [*]
  FAILED --> [*]
```

No adjustment history (latest-wins). The engine refetches the same record
on every poll until the status is terminal.

## Replay / dedup invariants

These three rules let us run polling and webhooks side-by-side without
double-counting:

1. **Idempotency-Key on every POST**: counterparty MUST dedup. The plugin
   uses the entity's natural reference (initiation reference, bank-account
   UUID, event-name + connector) as the key.
2. **Adjustment dedup keys**:
   - `PaymentAdjustmentID` = `(payment.reference, status, amount)`
   - `OrderAdjustmentID` = `(order.reference, status, baseQuantityFilled, fee, feeAsset)`
   - `PaymentInitiationAdjustmentID` = `(initiation.reference, createdAt, status)`
3. **`updatedAt` strictly increasing per record**: the engine's
   `updatedAtFrom` cursor is a high-watermark; non-monotonic timestamps
   cause records to be silently skipped on the next pass.

If your counterparty satisfies all three, you can safely run both polling
and webhooks at the same time, retry deliveries indefinitely, and replay
historical events without corrupting state.
