# Universal Connector — Webhook Event Catalogue (v1)

> This is the **catalogue / quick reference**. For the full
> "ins-and-outs" of the webhook subsystem — subscription protocol,
> signing recipe, verification rules, idempotency / retry semantics,
> error responses, polling-vs-webhook coexistence, and counterparty
> implementation recipes in Go / Python / Node — see the comprehensive
> guide in [`webhooks.md`](webhooks.md).

When a counterparty declares `CREATE_WEBHOOKS` (and optionally
`TRANSLATE_WEBHOOKS`) in `GET /v1/capabilities`, the Universal Connector
calls `POST /v1/webhooks` once per event type listed here at install. For
every subsequent event the counterparty MUST POST a `WebhookEvent` envelope
to the registered `callbackUrl`.

## Envelope

Every delivered body has this shape, regardless of event type:

```json
{
  "id":        "evt_01HV…",
  "type":      "payment.updated",
  "createdAt": "2026-05-13T12:34:56Z",
  "resource":  { /* one of the typed inline payloads below */ }
}
```

- `id` is **required** — it is used by the engine as the
  `WebhookIdempotencyKey`. Replay-safe: re-delivering the same `id` is a
  no-op.
- `type` MUST be one of the catalogue entries below.
- `createdAt` is RFC3339 UTC.
- `resource` is the typed payload (see "Resource shape" per event).

## Signing rules

The counterparty MUST sign every delivery with HMAC-SHA256 when (and only
when) it advertised `features.webhookSignature == "hmac-sha256"`.

The signature header is `X-Universal-Signature` and carries the lowercase
hex of `HMAC-SHA256(secret, "<timestamp>.<body>")`.
The timestamp header is `X-Universal-Timestamp` and carries an RFC3339 UTC
instant. Skew tolerance is **5 minutes** (matching every other Formance
connector, see [`internal/connectors/plugins/public/increase/webhooks.go`](../../increase/webhooks.go)).

The plugin verifies the signature with `subtle.ConstantTimeCompare`; any
mismatch yields a generic 401 — no detail is ever leaked back. See
[`webhooks.go`](../webhooks.go) `verifyHMACSHA256`.

## Catalogue

| Event type                 | When to deliver                                             | Required `resource` fields | Maps to                                       |
|----------------------------|-------------------------------------------------------------|----------------------------|-----------------------------------------------|
| `account.created`          | A new internal account exists on the counterparty           | `account`                  | `WebhookResponse.Account`                     |
| `account.updated`          | Account fields changed                                      | `account`                  | `WebhookResponse.Account`                     |
| `external_account.created` | Beneficiary appeared / was registered                       | `externalAccount`          | `WebhookResponse.ExternalAccount`             |
| `balance.updated`          | Account balance changed                                     | `balance`                  | `WebhookResponse.Balance`                     |
| `payment.created`          | Payment first observed (any source)                         | `payment`                  | `WebhookResponse.Payment`                     |
| `payment.updated`          | Status / amount / metadata changed                          | `payment`                  | `WebhookResponse.Payment`                     |
| `payment.deleted`          | Counterparty wants Formance to drop a record                | `paymentToDelete`          | `WebhookResponse.PaymentToDelete`             |
| `payment.cancelled`        | Counterparty wants Formance to mark cancelled               | `paymentToCancel`          | `WebhookResponse.PaymentToCancel`             |

### Not webhook-able: orders & conversions

`order.*` and `conversion.*` events are intentionally **not** in the
catalogue. The engine's
[`WebhookResponse` struct](../../../../models/plugin.go) exposes no
`Order` or `Conversion` field, so `TranslateWebhook` has no surface to
emit them. Subscribing the engine to those events at install would be
misleading — Formance would silently discard every delivery.

Orders and conversions are observed via the periodic
`FetchNextOrders` / `FetchNextConversions` polls (default
`pollingPeriod = 30m`, floor `20m`). Counterparties that need lower
latency for trading flows can shorten the connector's `pollingPeriod`
config to the 20-minute floor, or open an engine-side issue to extend
`WebhookResponse` with the missing fields.

If a counterparty pushes one of those event names anyway, the
plugin's `TranslateWebhook` returns a `400` with a clear error
("unsupported webhook event") so misuse is obvious.

## Resource shape

Each `resource.<field>` is the same wire schema served by the corresponding
`GET /v1/...` endpoint. See [`data-model.md`](data-model.md) for the
canonical field list per primitive.

## Failure modes

- Verification fails ⇒ 401 with no body. Counterparty SHOULD treat as
  permanent for that delivery and let the user investigate.
- Translation succeeds with empty `Responses[].XXX` ⇒ 200 OK; the engine
  treats the event as acknowledged but takes no action (used for `order.*` /
  `conversion.*`).
- Unknown event type ⇒ 400. Counterparty SHOULD stop sending it — the
  Universal Connector ignores unknowns at install too.
- Counterparty downtime / 5xx from Formance ⇒ counterparty MUST retry with
  the same `id`. The engine dedups on `id`, so delivery-at-least-once is
  safe.

See also:

- [`webhooks.md`](webhooks.md) — the in-depth guide (subscription
  protocol, signing recipe, verification rules, idempotency/retry,
  error semantics, implementation recipes).
- [`state-machines.md`](state-machines.md) — how webhooks slot into the
  polling and lifecycle flows.
