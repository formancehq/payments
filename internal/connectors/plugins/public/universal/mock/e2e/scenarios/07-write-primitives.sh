#!/usr/bin/env bash
# Scenario 07 — Write primitives end-to-end.
#
# Exercises the engine's full write-side pipeline through the
# universal connector — PAYOUT + TRANSFER — via the
# terminal-on-first-call mock path.
#
# Coverage notes (intentional gaps):
#   - Polling path (mode="polling" + repeat GetPayout until
#     terminal): covered by Go unit tests in payouts_test.go
#     (PollPayoutStatus). Skipped here because the engine's
#     `MinimumPollingPeriod = 20m` makes multi-poll scenarios
#     infeasible inside an interactive E2E budget.
#   - Reversal: covered by the plugin's ReversePayout / ReverseTransfer
#     Go tests + the mock's reversalResponse handler. Not driven
#     here because the engine constructs reverse workflow IDs by
#     concatenating multiple base64-encoded connector + initiation
#     IDs and runs into Temporal's 1000-char WorkflowID limit on
#     the universal connector's nested ID encoding (see
#     payments.log ERR "WorkflowId length exceeds limit"). That
#     limit is an engine-side concern and not something the
#     universal plugin can mitigate.
#
# Steps:
#   1) Install fresh in webhook mode + wait for FetchAccounts /
#      FetchExternalAccounts to populate the engine DB.
#   2) Pick one universal internal account (source) and one external
#      account (destination) from the engine's /v3/accounts.
#   3) POST /v3/payment-initiations of type=PAYOUT, amount=5000 (≤
#      €100 → mock returns mode=terminal Payment in one shot).
#   4) Assert initiation reaches PROCESSED + payment lands in
#      /v3/payments with matching reference.
#   5) Same again for TRANSFER (amount=7500).
#
# Also asserts that /v3/orders is non-empty — orders are pull-only
# (no webhook), populated by FetchNextOrders at install time, and
# this is the first scenario that verifies they reached the engine
# DB through the connector's mapper.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/../lib.sh"

CID="$(install_connector mode=webhook name=universal-e2e-write)"
[ -n "${CID}" ] || { log_err "install returned no connector id"; exit 1; }

wait_for_subscriptions 60

# The engine populates /v3/accounts via the bootstrap FetchAccounts
# task. Wait for both internal and external to land before we try to
# pick a source/destination.
log_info "waiting for universal accounts + external accounts to populate"
wait_until "at least 1 universal internal account" \
  "[ \$(curl -sS '${PAYMENTS_URL}/v3/accounts?pageSize=200' | jq '[.cursor.data[]? | select(.provider==\"universal\" and .type==\"INTERNAL\")] | length') -ge 1 ]" 60
wait_until "at least 1 universal external account" \
  "[ \$(curl -sS '${PAYMENTS_URL}/v3/accounts?pageSize=200' | jq '[.cursor.data[]? | select(.provider==\"universal\" and .type==\"EXTERNAL\")] | length') -ge 1 ]" 60

src=$(curl -sS "${PAYMENTS_URL}/v3/accounts?pageSize=200" \
  | jq -r '.cursor.data[]? | select(.provider=="universal" and .type=="INTERNAL") | .id' | head -1)
dst=$(curl -sS "${PAYMENTS_URL}/v3/accounts?pageSize=200" \
  | jq -r '.cursor.data[]? | select(.provider=="universal" and .type=="EXTERNAL") | .id' | head -1)
[ -n "${src}" ] && [ -n "${dst}" ] || { log_err "could not pick source/destination ids (src=${src} dst=${dst})"; exit 1; }
log_info "picked source=${src:0:24}... destination=${dst:0:24}..."

# initiate_and_wait <ref> <type> <amount> — drives /v3/payment-initiations
# with noValidation=true (skip the manual approval gate), polls for the
# initiation to reach PROCESSED, asserts the resulting payment lands
# in /v3/payments with a matching reference. Echoes the
# paymentInitiationID so the caller can chain a /reverse.
initiate_and_wait() {
  local ref="$1" type="$2" amount="$3"
  local sched
  sched="$(date -u -v+5S +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '+5 seconds' +%Y-%m-%dT%H:%M:%SZ)"
  local body
  body=$(jq -n \
    --arg ref "${ref}" --arg sched "${sched}" --arg cid "${CID}" \
    --arg src "${src}" --arg dst "${dst}" \
    --arg type "${type}" --argjson amount "${amount}" \
    '{reference:$ref, scheduledAt:$sched, connectorID:$cid, description:"e2e \($type)", type:$type, amount:$amount, asset:"EUR/2", sourceAccountID:$src, destinationAccountID:$dst}')

  log_info "initiating ${type} amount=${amount} reference=${ref}"
  local resp
  resp=$(curl -sS -X POST -H 'Content-Type: application/json' -d "${body}" \
    "${PAYMENTS_URL}/v3/payment-initiations?noValidation=true")
  local pid
  pid=$(echo "${resp}" | jq -r '.data.paymentInitiationID // empty')
  if [ -z "${pid}" ]; then
    log_err "/v3/payment-initiations failed: $(echo "${resp}" | jq -c .)"
    return 1
  fi
  log_info "${type} accepted, paymentInitiationID=${pid:0:24}..."

  wait_until "${type} ${ref} reaches PROCESSED" \
    "[ \"\$(curl -sS '${PAYMENTS_URL}/v3/payment-initiations/${pid}' | jq -r '.data.status // \"\"')\" = \"PROCESSED\" ]" 60

  wait_until "${type} ${ref} payment visible in /v3/payments" \
    "[ \$(curl -sS '${PAYMENTS_URL}/v3/payments?pageSize=200' | jq --arg ref '${ref}' '[.cursor.data[]? | select(.reference==\$ref)] | length') -ge 1 ]" 60

  log_info "${type} ${ref}: PROCESSED + payment ingested"
  echo "${pid}"
}

PAYOUT_REF="e2e-payout-$(date +%s)"
initiate_and_wait "${PAYOUT_REF}" PAYOUT 5000 >/dev/null

TRANSFER_REF="e2e-transfer-$(date +%s)"
initiate_and_wait "${TRANSFER_REF}" TRANSFER 7500 >/dev/null

# Pull primitives — orders are populated by FetchNextOrders at
# install time (no webhook event); first scenario that confirms they
# reach the engine DB end-to-end.
log_info "verifying /v3/orders contains universal orders"
wait_until "at least 1 universal order" \
  "[ \$(curl -sS '${PAYMENTS_URL}/v3/orders?pageSize=200' | jq '[.cursor.data[]? | select(.provider==\"universal\")] | length') -ge 1 ]" 60
