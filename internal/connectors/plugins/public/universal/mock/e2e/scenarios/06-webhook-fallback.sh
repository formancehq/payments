#!/usr/bin/env bash
# Scenario 06 — Webhook fallback when WS is not advertised.
#
# Install with the mock in webhook-only mode; assert the supervisor is
# NOT started AND deliveries land via the HTTP webhook callback.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/../lib.sh"

CID="$(install_connector mode=webhook)"
[ -n "${CID}" ] || { log_err "install returned no connector id"; exit 1; }

log_info "verifying capabilities advertise eventStream=\"\" (off)"
curl -sS -H "Authorization: Bearer ${MOCK_API_KEY}" "${MOCK_EXTERNAL_URL}/v1/capabilities" \
  | jq -e '(.features.eventStream // "") == ""' >/dev/null

wait_for_subscriptions 60
evolve 10 >/dev/null
assert_universal_payments_count_at_least 5 60

assert_log_contains "$(container_name universal-mock)" "auto-emit delivered" 2m
assert_log_does_not_contain "$(container_name universal-mock)" "ws handshake accepted" 30s
assert_log_does_not_contain "$(container_name universal-mock)" "stream broadcast" 30s
