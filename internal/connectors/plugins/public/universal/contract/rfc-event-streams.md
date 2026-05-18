# RFC — Event Streams as a first-class Plugin extension

**Status**: draft
**Owner**: Connectivity / Universal team
**Targets**: payments worker, plugin interface, Temporal usage
**Phase-1 reference**: this RFC's predecessor lives in the Universal
Connector at `client/stream.go`, `stream.go`, and `mock/stream.go`.
Phase 1 ships HTTP-loopback dispatch; this RFC defines the engine-level
extension that obsoletes the loopback and unlocks other connectors.

---

## Why now

Phase 1 added WebSocket streaming to the Universal Connector by
treating each WS frame as a synthetic HTTP webhook delivery — the
supervisor signs the body and POSTs to the connector's own webhook URL
through the local gateway. This reuses every line of the existing
webhook pipeline at zero engine cost.

The downsides surface once another connector adopts the pattern:

1. **Loopback HTTP hop per event** — fine at 20 k tx/day, painful at
   sustained ≥ 100 / s.
2. **Code duplication** — every connector that wants WS would
   reimplement supervisor + HMAC + URL building.
3. **Operator surface** — `STACK_PUBLIC_URL` becomes load-bearing for
   correctness (Phase 1 already depends on it for pod-restart recovery).
4. **No cost levers** — Phase 1 inherits today's per-event Temporal
   workflow cost. Cheaper alternatives (batching, fast-path) require a
   contract surface the loopback path can't deliver.

The right answer is a small, opinionated extension to the Plugin
interface so the engine wires a direct in-process dispatcher and other
connectors get the supervisor for free.

## Why NOT a long-running Temporal activity

The original sketch proposed a `TASK_STREAM_EVENTS` task type backed by
a long-running activity with heartbeats. After a closer look this is a
known anti-pattern for persistent inbound streams:

- **Worker slot starvation.** A worker has a fixed
  `MaxConcurrentActivityExecutionSize`. A WS connection that never
  returns pins one slot forever. With N stream-enabled connectors,
  worker sizing scales linearly with the number of installs.
- **Heartbeats are paid actions doing no useful work.** Temporal Cloud
  bills per action; at 20 s heartbeat × N connections × M tenants this
  is a steady cost floor that buys nothing the WS lib's TCP keepalive
  doesn't already provide. 100 connections ≈ 13 k actions / day of
  pure overhead.
- **Failure semantics are wrong.** Activity timeout → workflow retries
  the activity → fresh dial → connection state lost. There is no
  "reattach to the existing connection", which is exactly what you
  want for a WS supervisor.
- **Determinism boundary.** Workflow code is deterministic; once an
  activity returns you cross the boundary. A WS reader is fundamentally
  non-deterministic event-driven code that spends the whole activity
  body doing things Temporal isn't trying to help with.

Temporal is great for orchestrating durable mutations triggered BY an
inbound event (`RunHandleWebhooks` is the right shape). It is wrong
for holding the socket that delivers the event.

## Design

### New optional Plugin interface

```go
// internal/models/plugin.go
type PluginWithEventStream interface {
    // OpenEventStream is called once per (plugin, pod) on engine start
    // for any installed connector whose Plugin satisfies the interface.
    // The plugin owns the goroutine, the reconnect loop, and the
    // lifetime; it calls dispatcher.Dispatch for each inbound event.
    // Returns when ctx is cancelled or the plugin gives up.
    OpenEventStream(ctx context.Context, dispatcher EventDispatcher) error

    // CloseEventStream is called on Uninstall and on graceful shutdown.
    // Best-effort drain + clean WS close (status 1001).
    CloseEventStream(ctx context.Context) error
}

type EventDispatcher interface {
    // Dispatch runs the event through VerifyWebhook + TranslateWebhook +
    // persist + emit synchronously, in-process. Returns the error so
    // the plugin can apply backpressure or NACK to the counterparty.
    Dispatch(ctx context.Context, connectorID models.ConnectorID, eventName string, body []byte) error

    // DispatchBatch is the cost-reducing variant: one engine pipeline
    // run per batch. Plugins SHOULD prefer it when the underlying
    // transport delivers frames in groups (most WS PSPs do).
    DispatchBatch(ctx context.Context, connectorID models.ConnectorID, frames []DispatchFrame) error
}

type DispatchFrame struct {
    EventName string
    Body      []byte
}
```

### Engine wiring

- `engine.OnStart` iterates installed connectors as today. For each
  whose `Plugin` satisfies `PluginWithEventStream`, the engine spawns
  one supervisor goroutine that calls `OpenEventStream(ctx, dispatcher)`.
- `dispatcher` re-enters `engine.HandleWebhook`'s inner logic minus the
  HTTP routing layer. The current `handler_connectors_webhooks.go`
  becomes one of two callers of the same internal dispatcher; the WS
  supervisor is the other. Zero pipeline duplication.
- Graceful shutdown cancels the supervisor ctx; the plugin's
  `CloseEventStream` returns within a 30 s deadline before the engine
  forces.

### TaskType + workflow tree

Add `TASK_STREAM_EVENTS` to [`internal/models/connector_tasks_tree.go`](../../../../models/connector_tasks_tree.go).
Workflow.go in any consumer plugin adds the node when the relevant
capability is declared. The task is **lifecycle-only** — it carries no
Temporal-scheduled activities. Its presence in the tree tells the
engine: "this connector wants the supervisor running while installed".
The engine implements lifecycle outside the Temporal scheduler.

### Plugin migration (Universal first)

Universal moves from Phase-1 HTTP loopback to the in-process
dispatcher under the new interface. Contract (`features.eventStream`,
`features.streamEvents`, frame shape, signed handshake) is unchanged.
Mock is unchanged. The migration is:

1. Implement `OpenEventStream` / `CloseEventStream` (largely a port of
   the existing `streamSupervisor`).
2. Replace `httpDispatcher` with `engineDispatcher` (in-process call).
3. Delete the `maybeStartStream` / `ensureStreamRunning` HTTP-loopback
   plumbing.
4. Mark `TASK_STREAM_EVENTS` in `workflow.go` when the per-install
   set has `CAPABILITY_TRANSLATE_WEBHOOKS` AND `features.eventStream==
   "wss"`. (Capability conjunction is the same install-time check
   Phase 1 enforces.)

Other connectors adopt the same interface; the implementation surface
is tiny because the engine owns the dispatcher.

## Lifecycle, pod failure & recovery

(Same model as Phase 1 — the WS connection lives in the worker pod,
just under a more disciplined contract.)

**State that lives in the pod (and nowhere else):**

- The WS socket file descriptor + TLS session.
- A small bounded dispatch buffer (default 64 frames).
- A reconnect-attempt counter for backoff.

Everything else (connector config, webhook subs, idempotency state) is
in Postgres.

**Graceful shutdown (SIGTERM, rolling deploy):**

1. Engine receives SIGTERM, propagates ctx cancellation to every
   `OpenEventStream`.
2. Each plugin stops reading, drains the in-flight buffer (bounded;
   < 1 s at typical sizes), sends WS close 1001.
3. Counterparty marks the subscription as gracefully disconnected; the
   next pod's `OpenEventStream` reconnects within seconds.
4. **No events lost** in this path.

**Crash (SIGKILL, OOM, kernel panic):**

1. TCP RST or keepalive timeout on the counterparty side; events
   buffered for the dead pod are dropped server-side.
2. **In-flight buffer is lost.** Recovery depends on the counterparty:
   - **(a) `?since=<cursor>` on reconnect** — cleanest. RFC recommends
     Universal mandate this.
   - **(b) No replay** — falls back to the periodic `FetchNext*` poll
     as the safety net.
   - **(c) Outbox / ack** (Stripe-style) — out of scope for v1; v2
     enhancement.

**Reconnect storm protection:**

- Exponential backoff with **full jitter** (cap 60 s), already in the
  Phase 1 supervisor — port unchanged.
- Counterparty SHOULD enforce a per-installation connect rate limit
  (e.g. 1 / 10 s).

**Pod startup & re-establishment:**

- `engine.OnStart` walks every installed connector and spawns the
  supervisor goroutine when the plugin satisfies the interface. No
  lazy-on-FetchNext* recovery path needed — the engine owns the
  lifecycle, not the inbound poll.

**Multi-pod coordination (no leader election in v1):**

- N replicas → N WS connections → counterparty fans out → engine
  receives each event N× → `WebhookIdempotencyKey` dedup absorbs
  duplicates. Cost: up to N× the per-event Temporal cost.
- **v1.5 cost optimisation:** add a Postgres advisory lock keyed by
  `(connectorID, stream)`. Cheaper than Temporal heartbeats; no extra
  infra; lock released on pod death by Postgres itself.

**Temporal worker lifecycle interaction:**

- Supervisor and Temporal worker share the process; both die on the
  same SIGTERM. No `select`-on-Temporal-shutdown plumbing needed.

**At-least-once vs at-most-once:**

- v1 is **at-least-once** end-to-end. Counterparty must emit a stable
  `id`; engine dedups via `WebhookIdempotencyKey`; in-process
  dispatcher inherits the property because dedup is downstream of
  dispatch.

## Cost levers committed in this RFC

1. **`DispatchBatch`** as a first-class dispatcher method — plugins
   buffer frames for up to 100 ms (or 50 events, whichever comes
   first) and dispatch in a single workflow run. Engine pipeline
   iterates the batch and persists each event; one workflow per batch
   instead of one per event. Concrete saving at 100 / s: 99 % action
   reduction.

2. **Fast-path / slow-path split** — the engine factors the current
   `RunHandleWebhooks` into:
   - **Fast path** (in-process, no Temporal): pure
     `Verify → Translate → persist → emit`. Used by both the new
     dispatcher and the existing HTTP webhook handler.
   - **Slow path** (Temporal workflow): side-effecting or retry-
     requiring activities. Triggered only when fast-path persistence
     fails or downstream consumers need durable retry.
   Implementation note: this is the **biggest** engine refactor in the
   RFC and is the gate for sustained-throughput WS.

3. **Per-tenant WS connection multiplexing** — for the Universal
   connector specifically: when N installs of Universal point at the
   same `cfg.Endpoint`, open one TCP and demux per `connectorID`. The
   dispatcher interface already accepts `connectorID`, so the engine
   doesn't need to change. The supervisor needs a small registry keyed
   by endpoint.

## WS-to-WebhookEvent mapping (for non-Universal connectors)

Universal contracts the 1:1 envelope match by design (a feature, not a
limitation). Real PSPs differ on every dimension — see
[`webhooks.md`](webhooks.md) "WebSocket transport ↔ webhook" table.
The RFC commits the interface to NOT bake a Universal-only 1:1
assumption: `DispatchFrame.Body` is opaque bytes that go through the
plugin's own `TranslateWebhook`, so per-PSP envelope translation and
delta-to-snapshot materialization stay inside the connector where they
belong. Two patterns:

1. **In-connector mapper** — per-PSP `client.Read` does decode +
   materialization, then calls `dispatcher.Dispatch(ctx, id, name, body)`
   with a `body` that conforms to the engine-side webhook contract.
2. **Bridge service outside the connector** (recommended for messy
   PSPs) — consume the PSP's native WS, materialize snapshots and
   topic translation, re-emit in the engine-routable shape. Keeps the
   connector contract-pure; mirrors the Banking Bridge pattern.

## Observability

Required metrics (Prometheus, per `(connector, stream)`):

| Metric | Type | Notes |
|---|---|---|
| `connector_stream_connected` | gauge | 1 when supervisor holds an open connection |
| `connector_stream_reconnects_total` | counter | every successful reconnect |
| `connector_stream_dial_errors_total` | counter | label on error class |
| `connector_stream_events_received_total` | counter | per event name |
| `connector_stream_dispatch_errors_total` | counter | per event name |
| `connector_stream_dispatch_latency_seconds` | histogram | end-to-end |
| `connector_stream_buffer_depth` | gauge | tracks backpressure |

Log lines on connect / disconnect / reconnect-attempt include
`connector`, `endpoint`, `attempt`, `last_error_class`. Match the
existing webhook metric shape so Grafana dashboards generalise.

## Auth

The handshake auth scheme is Universal-specific (signed hello with
`timestamp.nonce.canonicalEvents` HMAC). The RFC interface
intentionally does NOT prescribe this — `OpenEventStream` is opaque to
the engine; the plugin handles its own auth. Universal's scheme stays
documented in [`webhooks.md`](webhooks.md) and reused by any future
plugin that wants the same primitive.

## Migration plan / sequencing

1. Land Phase 1 (HTTP-loopback supervisor) — DONE.
2. Implement the `PluginWithEventStream` + `EventDispatcher` interfaces.
3. Refactor `engine.HandleWebhook` to expose the inner pipeline as a
   reusable callable. No external behavior change.
4. Implement `engineDispatcher`. Wire `engine.OnStart` to spawn
   supervisors for installed connectors that satisfy the interface.
5. Migrate Universal: implement `OpenEventStream` / `CloseEventStream`,
   delete `httpDispatcher` + `ensureStreamRunning` + `maybeStartStream`,
   add `TASK_STREAM_EVENTS` to the workflow tree.
6. Add `DispatchBatch` and prove the cost win on a synthetic load
   test (100 events / s for 10 min; compare workflow action counts).
7. (v1.5) Postgres advisory lock for leader election.
8. (v2) Fast-path / slow-path split.

## Open questions

- WS for outbound commands too (engine → PSP) or strictly inbound?
- Hybrid mode where some events come over WS and others stay on HTTP
  webhooks (low-latency vs catch-all)?
- Per-tenant WS connection multiplexing — when and how?
- Backpressure policy: drop oldest, drop newest, NACK to counterparty,
  or block dispatcher?

## References

- Phase 1 implementation: [`../stream.go`](../stream.go),
  [`../client/stream.go`](../client/stream.go),
  [`../mock/stream.go`](../mock/stream.go).
- Webhook pipeline today:
  [`internal/api/v3/handler_connectors_webhooks.go`](../../../../../api/v3/handler_connectors_webhooks.go),
  [`internal/connectors/engine/engine.go`](../../../../engine/engine.go) (`HandleWebhook`),
  [`internal/connectors/engine/workflow/handle_webhooks.go`](../../../../engine/workflow/handle_webhooks.go).
- Capabilities + features wire shape: [`universal-openapi.yaml`](universal-openapi.yaml).
