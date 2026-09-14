# Payments → fctl command inventory

Status: **implemented in source; local unit contract verified.** This document
retains the reproducible source inventory behind the 44-command Payments v3 catalogue. The core adapter,
portable descriptor, lifecycle entry point, deterministic component build and
focused tests live alongside it. This is not installed or live execution
acceptance; §9 distinguishes the local unit contract from external release
gates.

Audit refreshed: 2026-09-12. Every count in §3 and every table in
`v3-operations.generated.md` is derived by `plugins/fctl/cmd/specaudit` from
this repository's own committed OpenAPI document and pinned by a golden test —
none is transcribed by hand.

## 1. Pinned revisions and authoritative sources

| Source | Revision | Role |
|---|---|---|
| `formancehq/payments` (this repo) | `6066bb9296157465fbe11ea7f347f6b46f27a550`, live `origin/main` HEAD verified 2026-09-12 | Authoritative: `openapi.yaml`, `openapi/v3/*`, server routes (`internal/api/`), storage (`internal/storage/`), generated client (`pkg/client/`) |
| historical fctl baseline | `693c58e27865f83332e6c3199d61fed81b742f41` (read-only source revision) | The prior command surface used to measure compatibility (`cmd/payments/**`) |
| fctl-v2 SDK | `e9b1395f46f3100b381dbe00f5213de28e6df0e1`, SDK NAR `sha256-DnTiEFya3R9KCYmgv5SO/1StKTCPmndObQrrVHf79Xk=`, WIT SHA-256 `38fdf377264eeada82b23fef153e6bf106ed0624e8916ff62cabdf210d6255f5` | Public plugin SDK, `producthttp` adapter, RFC 0011 and portable lifecycle contract used by this module |
| `go-libs` | `v5.6.1` (this repo's `go.mod`) | Authentication middleware behaviour behind §6 |

`generated-client-provenance.json` records the exact product revision, Git
object IDs and OpenAPI SHA-256, plus the validated RFC 0011 digest and exact
fctl SDK source lock used here. `with-fctl-sdk.sh` accepts the checkout location
through `FCTL_SDK_ROOT`. For a Git source it validates repository/commit and
exports the locked SDK and WIT from that commit before checking the NAR content
and WIT digest; dirty checkout files are excluded by construction. It then
creates a disposable Go workspace. No workstation path is committed.

Provenance note, stated rather than hidden: an earlier fctl-v2 audit pinned
Payments at `faac9792eb45fcc092e76adfb4a3d56b9b171575` (2026-09-02). That head
has moved and **none of its per-commit findings are reused here**. The
pre-existing iCloud-backed checkout at
`…/projects/formance/payments` sits at `4b026b2ad1b0b69f73f654e3a3b32fa4d30eb795`
and could not be used: `git status` did not complete within two minutes there.
It was left untouched.

## 2. Method

- Operations are read from the **merged** `openapi.yaml` at the repository
  root, which is the document `just openapi` produces from `openapi/common`,
  `openapi/v1-2` and `openapi/v3/*`. That is the same document the Speakeasy
  client in `pkg/client/` is generated from, so spec and client cannot disagree
  about which operations exist.
- Nothing is inferred from names. Scopes come from each operation's own
  `security` block; pagination requires both `cursor` and `pageSize` to be
  declared; request and response models are the resolved component schema
  names; the family table is an explicit per-operation map, so a new spec
  operation fails a test instead of being absorbed by a prefix rule.
- The absence of a declared `security` block is recorded as a distinct, weaker
  fact than a declared empty scope array. It is never rounded up to a guess.
- The legacy baseline was transcribed once, from
  `cmd/payments/**` at the pinned fctl revision, and is then held in place by
  tests: every legacy and /v3 operationId it names must exist in the current
  document.
- Server behaviour claims (§6) come from reading the routers and storage layer,
  not from running a server. No Payments server was started, and no live API
  was called.

Regenerate and verify:

```sh
just fctl-audit         # rewrite the committed artefacts
just fctl-audit-check   # fail if they drifted from openapi.yaml
```

## 3. Denominators

| Count | Value |
|---|---:|
| Operations in `openapi.yaml`, all tags | **109** |
| Unique `operationId`s | **109** (bijective with operations) |
| `payments.v1` operations | 45 |
| `payments.v3` operations | **64** ← the plugin denominator |
| Operations marked `deprecated: true` | 5 (all `payments.v1`) |
| Legacy fctl `payments` commands, executable leaves | **45** |
| Legacy commands mapped onto /v3 | **44** |
| Legacy commands excluded with evidence | **1** |
| /v3 operations reached by the legacy baseline | **44** |
| /v3 operations with no legacy precedent | **20** |
| /v3 operations carrying an admission blocker | **0** |
| /v3 operations with no admission blocker | **64** |
| /v3 executable operations carrying a release gap | **3** |

Cross-check against the generated client, which is independent of this audit's
parser: `pkg/client/v3.go` declares exactly **64** operation methods and
`pkg/client/v1.go` exactly **45**, matching the per-tag spec counts.

The historical "Payments has 45 operations" figure is confirmed, and its
meaning is now exact: it is the number of **executable legacy fctl `payments`
commands** at fctl `693c58e2`. Counting rule — grouping-only cobra commands
(`payments`, `payments pools`, `payments connectors schedules`, …) and shared
helpers (`cmd/payments/versions`, `cmd/payments/connectors/views`,
`cmd/payments/connectors/internal`) are not commands and are excluded. It
coincides numerically with the 45 `payments.v1` spec operations; the two are
different sets and the coincidence carries no meaning.

Note that 45 legacy commands map onto only 44 distinct /v3 operations, and
several commands issue more than one request (§5). Command count and operation
count are tracked separately throughout; neither is used as a proxy for the
other.

## 4. Frozen operation families

The executable compatibility surface covers four operation families. The
generated tables list every operation per family.

| Family | /v3 operations | Reached by legacy baseline |
|---|---:|---:|
| `connectors/schedules` | 12 | 9 |
| `payments/payment-initiations` | 14 | 12 |
| `accounts/bank-accounts` | 9 | 9 |
| `pools/orders/conversions/tasks` | 14 | 14 |
| **frozen total** | **49** | **44** |
| `payment-service-users` (recorded, **not** frozen) | 15 | 0 |

The 15 `payment-service-user` operations are the open-banking surface. They
exist in /v3, have no legacy fctl precedent, and are deliberately left outside
the frozen tranche rather than silently dropped: admitting them is a product
scope decision, not an audit finding. They are fully inventoried so that
decision can be made from facts.

`getServerInfo` (`GET /_info`) is classified separately as `server-probe`. It
is the only operation fctl needs before it knows the product major, it is
declared under the `payments.v1` tag while being served unprefixed, and it is
host-owned rather than part of any product family.

## 5. Legacy baseline mapping

The full 45-row mapping is in `v3-operations.generated.md` §"Legacy baseline
mapping". Summary of the non-trivial cases:

**Renamed domain.** The legacy `transfer_initiation` commands map onto the /v3
`payment-initiations` operations: `create` → `v3InitiatePayment`, `list` →
`v3ListPaymentInitiations`, `get` → `v3GetPaymentInitiation`, `delete` →
`v3DeletePaymentInitiation`, `retry` → `v3RetryPaymentInitiation`, `reverse` →
`v3ReversePaymentInitiation`. `approve` and `reject` already call the /v3
operations at the pinned fctl revision.

**Multi-request commands.** Three legacy commands issue more than one request
and must bind authorisation per actual request, never per command:

| Command | /v3 operations, in issue order |
|---|---|
| `payments connectors get-config` | `v3ListConnectors`, then `v3GetConnectorConfig` |
| `payments connectors install` | `v3ListConnectorConfigs`, then `v3InstallConnector` |
| `payments connectors update-config` | `v3ListConnectorConfigs`, then `v3UpdateConnectorConfig` |

**Legacy version branching, dropped.** Twelve legacy commands name
operationIds from both tags. Ten of them branch on a probed server version
between a `payments.v1` and a `payments.v3` form of the same behaviour. The
remaining two — `payments connectors install` and `payments connectors
update-config` — read the connector configuration schema from /v3 while
mutating through v1. The plugin declares product major 3 only, so in every case
only the /v3 operation is carried; the v1 branch is not a fallback to preserve.

**Shape changes to carry deliberately.** `v3AddAccountToPool` takes the account
in the path (`POST /v3/pools/{poolID}/accounts/{accountID}`) where the legacy
v1 operation took it in the body; `payments pools balances <poolID> <at>` maps
to `v3GetPoolBalances`, whose `at` is a query parameter.

### 5.1 The single exclusion

`payments transfer_initiation update_status <transferID> <status>` —
**excluded**, with no capability loss.

- Its request schema accepts exactly two values:
  `components.schemas.UpdateTransferInitiationStatusRequest.status` is
  `enum: [REJECTED, VALIDATED]`.
- /v3 splits precisely those two transitions into dedicated operations,
  `v3ApprovePaymentInitiation` (VALIDATED) and `v3RejectPaymentInitiation`
  (REJECTED), both already carried by `transfer_initiation approve` and
  `transfer_initiation reject`.
- The legacy command labels itself "deprecated in >= v3.0.0"
  (`cmd/payments/transferinitiation/update_status.go:47`).
- Re-adding a status verb on /v3 would mean choosing an operation per status
  value, which is exactly what the two dedicated commands already express.

No other legacy behaviour is excluded. The five deprecated `payments.v1`
operations are not excluded *commands*: they are v1 connector paths superseded
by `connectorId`-qualified v1 forms and by /v3 operations, and are recorded as
divergence D3.

## 6. Authorisation, exactly

Declared scopes, read per operation from the document: 55 operations declare
`payments:read`, 51 declare `payments:write`, and 3 declare nothing. Every
declaring operation declares exactly one scope under the single
`Authorization` scheme — there are no unions, no alternatives, and no
operation with a declared-but-empty scope array.

Two facts constrain how far that can be trusted:

- **The `payments` service does not enforce scopes.** `internal/api/v3/router.go`
  wraps its authenticated group in `jwt.Middleware(a)`, which in `go-libs`
  v5.6.1 (`pkg/authn/jwt/middleware.go`) only calls `Authenticate`. Scope
  enforcement (`ErrMissingScope`, `CheckEndpointSpecificScopesClaim`) lives in
  `ControlPlaneMiddleware`, which this router does not use, and the repository
  contains no occurrence of `payments:read` or `payments:write` in Go sources.
  The scope arrays are therefore a **declared contract enforced upstream**, and
  a per-operation scope catalogue is provable from the document but not
  verifiable against this service's own behaviour. See divergence D2.
- **Three operations declare no scopes at all** while the server does require
  authentication for them. They remain executable with the explicit empty
  exact-scope representation; see release gap G1.

`GET /_info` needs its own statement: the document declares it under
`payments:read`, but the server registers it on the root router outside every
authenticated group (`internal/api/router.go`), so it answers unauthenticated.
fctl must read it unauthenticated to learn the major before selecting a
provider. See divergence D1.

## 7. Risks, per operation

The generated tables carry a risk cell per operation. The classifications and
their evidence:

**`secret` — credentials cross the boundary.** Three operations, and the
direction matters.

- Inbound: `v3InstallConnector`, `v3UpdateConnectorConfig`. Both request bodies
  resolve to `V3ConnectorConfig`.
- Outbound: `v3GetConnectorConfig`. `V3GetConnectorConfigResponse.data` also
  resolves to `V3ConnectorConfig`.

`V3ConnectorConfig` has 23 provider variants
(`openapi/v3/v3-connectors-config.yaml`); 22 of them declare credential fields
— `apiKey`, `apiSecret`, `clientSecret`, `privateKey`, `password`,
`passphrase`, `accessKey`, `secret`, `userCertificate`, `userCertificateKey`,
`configurationToken`, `stagingToken`, `webhookPassword`,
`webhookSharedSecret`. Only `V3DummypayConfig` has none. Two name-similar
fields are **not** credentials and are excluded from that list:
`V3BankingcircleConfig.authorizationEndpoint` (a URL) and
`V3WiseConfig.webhookPublicKey` (a public key).

The read path is not redacted. `internal/api/services/connector_configs.go`
`ConnectorsConfig` unmarshals the stored config, injects `provider`, and
re-marshals it unchanged; `internal/storage/connectors.go` `ConnectorsGet`
selects `pgp_sym_decrypt(config, …) AS decrypted_config`. So
`v3GetConnectorConfig` returns decrypted PSP credentials verbatim. At legacy
fctl `693c58e2` the per-connector views printed them directly — for example
`cmd/payments/connectors/views/stripe.go` renders `config.APIKey` — and the
legacy tree contains no redaction or masking helper anywhere. Redacting in the
plugin is therefore a **deliberate behaviour change** relative to the baseline.
Historical blocker B3 recorded the decision; the generated-DTO redaction closes
it before emission.

**`destructive` — removes or resets server state.** 8 operations: the 7 /v3
`DELETE` operations, plus `v3ResetConnector`, which is a `POST` and is listed
explicitly for that reason.

**`display_once` — one-shot value in the success body.** 2 operations,
`v3CreateLinkForPaymentServiceUser` and
`v3UpdateLinkForPaymentServiceUserOnConnector`. Both mint a provider
authorisation URL per link attempt
(`V3PaymentServiceUserCreateLinkResponse.link`,
`V3PaymentServiceUserUpdateLinkResponse.link`). Neither may be persisted, cached
or re-rendered from a stored result. Both sit in the non-frozen
`payment-service-users` family.

**Idempotence — there is no idempotency key.** The document declares no
`Idempotency-Key` or equivalent parameter on **any** operation, in any tag
(asserted by `TestNoIdempotencyKeyIsDeclared`). Consequence: a retried `POST`
is a second effect, so no automatic retry may be applied to
`v3InitiatePayment`, `v3CreatePayment`, `v3CreateAccount`,
`v3CreateBankAccount`, `v3CreatePool`, `v3InstallConnector`,
`v3ResetConnector`, or any other `POST`. `GET`, `PUT` and `DELETE` are
HTTP-idempotent and marked replay-safe. `PATCH` is never assumed replay-safe
without operation-specific proof, which this document does not provide. The one dedup-adjacent field in the
document, `clientOrderID`, explicitly documents itself as "stored for
traceability only — Formance does NOT dedup on this field".

**Pagination — cursor, on 17 operations.** Those 17 declare both `cursor` and
`pageSize` as query parameters. No operation in the document is a stream: every
success response is a single JSON body.

**`get-with-body` — 15 paginated `GET`s carry a JSON request body.** The
generated-client adapter preserves the body through `Host.Request`, which
keeps native semantics exact. Browser `fetch` rejects `GET` requests with a
body, so compatibility gap C1 remains open for browser execution.

## 8. Admission blockers, release gaps and divergences

Kept strictly separate from the verified facts above. An admission blocker
stops the operations it names. A release gap does not stop local execution but
must close before release evidence can be claimed.

### Admission blockers

None for source implementation. All 44 compatibility operations are represented
by the local unit contract; installed execution acceptance remains a separate
gate.

### Release gaps

**G1 — undeclared scopes.** `v3GetBankAccount`,
`v3UpdateBankAccountMetadata`, `v3ForwardBankAccount`.
`openapi/v3/v3-api.yaml` gives these three no `security:` key, unlike every
other /v3 operation, while `internal/api/v3/router.go` registers all three
inside the `jwt.Middleware(a)` group, so the server does reject
unauthenticated calls. A named scope for them would have to be invented. The
plugin admits the three baseline commands with the SDK's explicit empty exact
scope set for protected bearer-only operations. The missing scope names remain
an upstream source-contract gap; the fix belongs in this repository, in
`openapi/v3/v3-api.yaml`.

**C1 — browser `GET` with a request body (compatibility gap).** 15 operations, all the paginated /v3
listings: `v3ListAccounts`, `v3ListBankAccounts`, `v3ListConnectorSchedules`,
`v3ListConnectors`, `v3ListConversions`, `v3ListOrders`,
`v3ListPaymentInitiationAdjustments`,
`v3ListPaymentInitiationRelatedPayments`, `v3ListPaymentInitiations`,
`v3ListPaymentServiceUserConnections`,
`v3ListPaymentServiceUserConnectionsFromConnectorID`,
`v3ListPaymentServiceUserLinkAttemptsFromConnectorID`,
`v3ListPaymentServiceUsers`, `v3ListPayments`, `v3ListPools`. Each declares
`requestBody.content.application/json.schema: V3QueryBuilder` alongside method
`GET`, where `V3QueryBuilder` is `type: object, additionalProperties: true`.
The generated client matches: `pkg/client/v3.go` `ListAccounts(ctx, pageSize,
cursor, requestBody map[string]any, …)`. The original admission risk was that
a host transport that drops `GET` bodies degrades a filtered query into an
**unfiltered listing** — a wrong answer returned successfully, not an error.
The `producthttp` boundary preserves these bodies and the Payments adapter
tests the mapping through `Host.Request`, so native execution is admitted
without rewriting the query semantics. Browser `fetch`, however, rejects a
body on `GET`; the exact operation set is pinned as compatibility gap
`C1-browser-get-with-body`, and browser acceptance remains blocked. Two of the
17 paginated operations
(`v3GetAccountBalances`, `v3ListConnectorScheduleInstances`) declare no body
and are unaffected.

**B3 — unredacted connector config (historical, closed in the adapter).**
`v3GetConnectorConfig` recursively redacts the credential keys enumerated in
§7 on the generated response DTO before emitting the public result, while
preserving non-secret fields and exact `int64` values outside JavaScript's safe
integer range. This is an intentional safety improvement over the historical
command, and B3 is absent from the machine-readable blocker report.

### Divergences

Recorded because they change what fctl may assume; they do not by themselves
block an operation.

**D1 — `/_info` auth.** Document says `payments:read`; server serves it
unauthenticated on the root router. fctl must rely on the server behaviour,
because it needs the major before it can authenticate against the right
provider. The **document** is the side to fix.

**D2 — scopes not enforced in this service.** §6. The named scope contract is
enforced upstream; the three omitted declarations remain represented as an
explicit empty exact scope set rather than guessed from neighbouring routes.

**D3 — deprecated v1 connector paths.** `uninstallConnector`,
`readConnectorConfig`, `resetConnector`, `listConnectorTasks`,
`getConnectorTask` are marked deprecated in the document. Legacy fctl still
references two of them (`cmd/payments/connectors/uninstall.go`,
`cmd/payments/connectors/configs/getconfig.go`) as version-dependent
fallbacks. A major-3-only plugin carries none of them.

**D4 — one missing SDK name override.** 63 of the 64 /v3 operations carry
`x-speakeasy-name-override`; `v3UpdateConnectorConfig` does not. Its generated
Go method is therefore `V3.V3UpdateConnectorConfig`, not
`V3.UpdateConnectorConfig`. Cosmetic in the document, load-bearing in an
adapter: any code that derives the client method from the operationId by
convention gets exactly this one wrong. Fixable in
`openapi/v3/v3-api.yaml` in this repository.

## 9. Current implementation and remaining gates

**Done in this tranche.** Reproducible operation extraction from the pinned
current source; the 109/45/64 denominators cross-checked against the generated
client; the 45-command legacy baseline mapped 44/1 with the one exclusion
evidenced; the four families frozen and the open-banking surface recorded;
per-operation scopes, request and response models, and risks read from source;
0 admission-blocked operations, 3 executable operations carrying release gap
G1, browser compatibility gap C1, 1 closed historical blocker and 4 divergences recorded and separated from
facts;
determinism held by a golden report plus invariant tests over operationId
uniqueness, tag partition, family completeness, baseline consistency, blocker
coverage, credential classification, pagination, and the idempotency finding.

**Implemented in source; local unit contract verified.** The product-owned
module contains the 44-command v3 catalogue and its 47 ordered request policies (including the three composite
connector commands), exact operation policies, the Payments-generated v3 client over the
host-only `producthttp` endpoint/auth transport, host-owned JSON inputs, opaque
cursor traversal, connector-secret redaction, portable descriptor and WIT
lifecycle, and a two-lane deterministic component build with validation and a
16 MiB ceiling. Static tests compare every carried method, path, body,
pagination, scope and mutation classification with the current OpenAPI
document; the race-enabled module gate enforces at least 80% statement
coverage. Generated-client ambient authentication and retries are
disabled because the host owns both concerns.

Remaining external or upstream gates:

1. **G1, in this repository** — declare the three missing `security` blocks in
   `openapi/v3/v3-api.yaml`, regenerate, and re-run `just fctl-audit`. This is
   the remaining source-contract gap. Until then the catalogue uses the
   explicit empty exact-scope set permitted for protected bearer-only routes;
   it does not invent scope names.
2. **D1** — fix the `/_info` security declaration in the document, and confirm
   fctl-v2's unauthenticated major probe against it.
3. **Product-major selection** — prove that a `/_info` response reporting a
   non-3 major yields `plugin_incompatible` before any Payments request is
   issued in an installed-host scenario.
4. **Open-banking scope decision** — accept or defer the 15
   `payment-service-user` operations, with the two `display_once` link
   operations handled explicitly if accepted.
5. **Browser `GET`-body compatibility** — provide a proven body-free Payments
   API alternative for C1 before claiming browser acceptance; never drop the
   query body and return an unfiltered listing.
6. **Release evidence** — build the component in the declared authoring shell,
   install it through the supported OCI path, and run the real Payments
   scenarios on both native and browser hosts. No live service or publication
   is claimed by the local unit evidence.
