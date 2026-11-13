# Task and Temporal workflow IDs

Some engine operations start a Temporal workflow and return a task to the caller.
For these operations, a single string is used as three things at once:

- the `Reference` of the persisted `models.TaskID` (stored in the `tasks` table via `TasksUpsert`),
- the Temporal workflow ID passed to `ExecuteWorkflow`,
- the duplicate-execution guard, through `WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE`.

The reference is built by `models.TaskIDReference` (`pkg/domain/models/task_id.go`):

```text
<prefix>-<stack>[-<attempt>]-<connector provider>-<connector reference>-<object id>
```

The task ID returned by the API is the base64url (no padding) encoding of the canonical JSON
`{"Reference": ..., "ConnectorID": ...}`.

## Object ID component

Starting with the release after v3.4.9, the following operations put the SHA-1 hex digest of
the object ID (`models.IdempotencyKey(<id>)`, 40 characters) in the reference.
Previous releases put `<id>.String()` there.

| Operation | Prefix | Hashed value |
| --- | --- | --- |
| `ForwardBankAccount` (bank account and payment service user bank account forwarding) | `create-bank-account-<stack>` | `BankAccount.ID` |
| `CreateTransfer` | `create-transfer-<stack>-<attempt>` | `PaymentInitiationID` |
| `CreatePayout` | `create-payout-<stack>-<attempt>` | `PaymentInitiationID` |
| `ReverseTransfer` | `reverse-transfer-<stack>` | `PaymentInitiationReversalID` |
| `ReversePayout` | `reverse-payout-<stack>` | `PaymentInitiationReversalID` |

Other task references (install, uninstall, reset, delete user/connection, complete user link) are unchanged.

### Why

`PaymentInitiationID.String()` and `PaymentInitiationReversalID.String()` are base64 blobs that
embed the connector ID again, even though `TaskIDReference` already writes the connector
provider and reference in plain text. With a long connector reference, the resulting workflow ID
exceeded the length limit of Temporal's PostgreSQL persistence (EN-1346 / EN-1347).
A fixed 40-character digest bounds the length of the object ID component.

### Properties

- **Deterministic.** The digest is computed from the canonical JSON of the ID, so every replica
  running this release derives the same reference for the same object and attempt.
  The digest changes if the JSON shape of the ID struct changes. Changing those structs
  is therefore an identifier change and must be handled as one.
- **Opaque.** The object ID cannot be recovered from the task reference.
  Callers must treat the task ID as an opaque handle and use it only with the tasks API.

## Behavior change for API consumers

Decoding a task ID returned by these endpoints no longer reveals the bank account,
payment initiation or reversal ID. A consumer that decoded the task ID and searched the
reference for the object ID must stop doing that. Use the object ID from the request or
response body instead. The task ID itself keeps the same encoding and keeps working with the tasks API.

## Migration (RFC 0014)

| Item | Value |
| --- | --- |
| Class | Cutover (identifier format). There is no expand, backfill or contract step. |
| Destructive | No. Existing `tasks` rows and existing Temporal workflows are not modified. |
| Execution owner | None. No migration runs. New references are produced on new writes only. |
| Rollback version | v3.4.9 |
| Rollback window | Closes when the rollout is complete. No later contract release is required. |
| Operator action | No migration step. Retry restrictions apply, see [Retry restrictions](#retry-restrictions-after-upgrade-or-rollback). |

### Existing state

- Task IDs already returned to clients keep resolving, because the stored reference is never rewritten.
- Running workflows (including scheduled payouts and transfers sleeping until `ScheduledAt`)
  receive their `TaskID` as workflow input and complete normally. No code rebuilds a workflow
  ID from an object ID. Connector-wide termination looks up workflows by search attributes.

### Duplicate-execution protection across versions

Both versions read all existing state correctly. The gap is duplicate-execution protection:
v3.4.9 and this release compute different workflow IDs for the same object and attempt,
so Temporal's `REJECT_DUPLICATE` cannot find an execution started by the other version.
This shows up in two different ways.

**Concurrent execution.** Calls for the same object are served by both versions at the same time,
during a rolling update.

**Sequential replay.** A call for an object that was already accepted by one version is sent again
after the other version is running. This needs neither overlapping replicas nor in-flight traffic:
a request accepted before the upgrade and retried after it is enough. The same applies after a rollback.

Some operations have a guard that does not depend on the workflow ID:

| Operation | Guard outside the workflow ID | Concurrent execution | Sequential replay |
| --- | --- | --- | --- |
| Create payment initiation | `PaymentInitiationsInsert` has no `ON CONFLICT`: a second create with the same ID fails | Protected | Protected |
| Approve payment initiation | Last adjustment must be `WAITING_FOR_VALIDATION` | **Exposed** until the first workflow writes its adjustment | Protected once the first workflow has run |
| Retry payment initiation | Last adjustment must be `FAILED` or `NOT_INITIATED`; the attempt number comes from stored adjustments | **Exposed** until the first workflow writes its adjustment | Protected: each retry gets a new attempt number, so the workflow ID was never the guard |
| Reverse transfer / payout | `validateOnlyReverse` blocks a second reversal while one is in progress; `validateReverseAmount` caps the total reversed | Protected by `validateOnlyReverse` | **Exposed**: a completed partial reversal sent again starts a new workflow and runs again if the cap allows it |
| Forward bank account | None | **Exposed** | **Exposed**: a forward sent again creates the account at the PSP a second time |

Within a single version, the guard works as before. In every version, including v3.4.9, the
workflow ID only protects for the Temporal namespace retention period: once the earlier
execution is deleted, the same ID can start again. Sequential replay of bank account forwards and
completed partial reversals was therefore already possible after retention expiry. This release
also makes it possible across the upgrade, regardless of retention.

No durable guard (for example a stored per-object forward record or a reversal idempotency check)
is added by this change.

### Rollout recommendation

Making sure the two versions never serve the API at the same time (for example a `Recreate`
strategy, or `maxSurge: 0` with a single replica), or deploying while no payment initiation,
reversal or bank account forwarding traffic is in flight, prevents **concurrent execution** only.
It does not prevent sequential replay.

### Retry restrictions after upgrade or rollback

Until a durable guard exists, after upgrading to this release or rolling back to v3.4.9:

- Do not resend a bank account forward (`POST /v3/bank-accounts/{bankAccountID}/forward` or
  `POST /v3/payment-service-users/{paymentServiceUserID}/bank-accounts/{bankAccountID}/forward`) that was
  accepted by the other version. Check the original task's status, or the bank account's
  related accounts, instead.
- Do not resend a payment initiation reversal that was accepted by the other version.
  Check the original task's status, or the payment initiation's adjustments, instead.

Create, approve and retry of payment initiations can be resent: their state checks reject duplicates.

### Validation

| Path | Evidence |
| --- | --- |
| Fresh install | e2e suite (`just tests`). `test/e2e/api_bank_accounts_test.go` and `test/e2e/api_payment_service_users_test.go` assert that the reference contains the hashed ID. |
| Upgrade from v3.4.9 | No schema change. Existing tasks are read by ID and are not parsed. |
| Running workflows across the upgrade | Established by code review: workflows use the `TaskID` from their input. Not covered by an automated test. |
| Rollback to v3.4.9 | v3.4.9 reads tasks by stored ID and does not parse references. Not covered by an automated test. |
| Duplicate execution | Within one version: unchanged (`REJECT_DUPLICATE`). Across versions: see the protection table above. Established by code review, no automated test. |
| Production exposure | Grafana, `temporal_workflow_completed`, 90 days before this change: no create payout/transfer, reverse payout/transfer or create bank account workflows on any production stack. |
