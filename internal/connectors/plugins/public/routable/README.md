# Routable Connector (Design Draft)

This connector would integrate the Routable payments platform with the Formance Payments service, enabling vendor onboarding, bill payables, and outbound payments via Routable's API.

References:
- Routable product overview: `https://routable.com/`
- Routable developer docs: `https://developers.routable.com/`

## Capabilities (proposed)

- Fetch external accounts (vendors + payment methods)
- Fetch payments (payables/payments state)
- Create payouts (initiate outbound payments)
- Webhooks: create/verify/translate (payment and vendor events)
- Optionally: Fetch accounts and balances (Routable Balance)

## Configuration (proposed)

- `apiToken` (string, required): Routable API token (sandbox or production)
- `baseURL` (string, required): API base URL (e.g., `https://api.sandbox.routable.com`)
- `webhookSharedSecret` (string, optional): For webhook signature verification
- `accountingIntegration` (string, optional): If leveraging accounting sync metadata (QBO, NetSuite, etc.)

## Resource Mapping

- Formance External Accounts ↔ Routable Vendors/Contacts + Payment Methods
  - Create external account: create/update vendor, attach bank account/payment method
  - Fetch external accounts: list vendors and their default payment methods
- Formance Payments ↔ Routable Payables/Payments
  - Create payout: create `Payable` with proper details, set `ready_to_send` or schedule (`send_on`), handle currency and amount
  - Fetch payments: list payments and statuses; map to Formance payment states
- Accounts/Balances (optional) ↔ Routable Balance
  - If the tenant uses Routable Balance, expose balance as an account with periodic balance fetch

## Payment Methods & Currencies

- Routable supports ACH, wires, checks, international payments in 140+ currencies to 220+ countries
- Formance amounts must be sent in currency's smallest unit; ensure conversion where needed

## Webhooks (proposed)

- Configure webhooks for payment lifecycle events and vendor updates
- Verify signatures using `webhookSharedSecret`
- Translate events to Formance webhook responses: payments created/updated, vendor created/updated, failed payments, etc.

## Idempotency, Pagination, Rate Limits

- Set `Idempotency-Key` per create operations (mapped from Formance idempotency key)
- Respect Routable pagination and rate limit (default 600 req/min)
- Implement retries with backoff on 429/5xx

## Error Handling

- Map Routable error codes to Formance `UNAUTHORIZED`, `BAD_REQUEST`, `NOT_FOUND`, `CONFLICT`, `UNAVAILABLE`

## Initial Implementation Plan

1) Capabilities and config scaffolding
2) Client and auth wiring (bearer `apiToken`, base URL)
3) External accounts: create/fetch vendors + payment methods
4) Payouts: create payable and trigger/schedule payment
5) Payments: fetch list/status + webhooks translation
6) Optional: balance account exposure
7) Reliability: idempotency, retries, rate limiting
8) Tests + dev server wiring

See the Increase connector README for formatting and examples of usage flows.
