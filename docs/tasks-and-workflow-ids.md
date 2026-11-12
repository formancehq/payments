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
| Operator action | None required. See [Rollout recommendation](#rollout-recommendation). |

### Existing state

- Task IDs already returned to clients keep resolving, because the stored reference is never rewritten.
- Running workflows (including scheduled payouts and transfers sleeping until `ScheduledAt`)
  receive their `TaskID` as workflow input and complete normally. No code rebuilds a workflow
  ID from an object ID. Connector-wide termination looks up workflows by search attributes.

### N / N-1 coexistence

Both versions read all existing state correctly. The one gap is duplicate-execution protection:
v3.4.9 and this release compute different workflow IDs for the same object and attempt,
so Temporal's `REJECT_DUPLICATE` no longer deduplicates across versions.

- **Create transfer / payout.** The workflow ID is the only guard. The workflows add their
  `PROCESSING` adjustment unconditionally. If two create, approve or retry calls for the
  same payment initiation land on different versions before the first workflow has written its
  adjustment, both pass the API-level status check and both reach the PSP.
- **Reverse transfer / payout.** `validateOnlyReverse` rejects a second reversal while one is in
  progress, and `validateReverseAmount` caps the total reversed amount. A partial reversal that
  already completed on one version and is submitted again on the other version starts a
  new workflow and can be executed a second time, as long as the cap allows it.
- **Forward bank account.** A duplicate forward creates the bank account at the PSP a second time.

Within a single version, the guard works as before. Across any version, it holds only for the
Temporal namespace retention period, because after that the workflow ID can be reused.

The same window applies during a rollback from this release to v3.4.9.

### Rollout recommendation

To remove the coexistence window, make sure the two versions never serve the API at the same time
(for example a `Recreate` strategy, or `maxSurge: 0` with a single replica), or deploy while no
payment initiation, reversal or bank account forwarding traffic is in flight.
If neither is possible, the risk is limited to concurrent or retried calls for the same object
during the rollout.

### Validation

| Path | Evidence |
| --- | --- |
| Fresh install | e2e suite (`just tests`). `test/e2e/api_bank_accounts_test.go` and `test/e2e/api_payment_service_users_test.go` assert that the reference contains the hashed ID. |
| Upgrade from v3.4.9 | No schema change. Existing tasks are read by ID and are not parsed. |
| Running workflows across the upgrade | Established by code review: workflows use the `TaskID` from their input. Not covered by an automated test. |
| Rollback to v3.4.9 | v3.4.9 reads tasks by stored ID and does not parse references. Not covered by an automated test. |
| Duplicate execution | Within one version: unchanged (`REJECT_DUPLICATE`). Across versions: not guarded, see above. |
