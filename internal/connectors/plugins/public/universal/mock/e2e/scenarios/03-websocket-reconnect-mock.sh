#!/usr/bin/env bash
# Scenario 03 — WS reconnect on counterparty restart.
#
# Install wss; evolve; assert delivery. Restart the universal-mock
# container; assert the plugin reconnects within 60s with a fresh nonce
# (signed handshake survives restart). Continue evolving; assert
# cumulative delivery.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/../lib.sh"

CID="$(install_connector mode=wss)"
wait_until "first ws handshake" \
  "docker logs --since 2m \"$(container_name universal-mock)\" 2>&1 | grep -F 'ws handshake accepted' >/dev/null" 30
evolve 5 >/dev/null
assert_universal_payments_count_at_least 3 60

log_info "restarting universal-mock container"
docker restart "$(container_name universal-mock)" >/dev/null
wait_until "universal-mock healthy after restart" \
  "curl -sSf -H 'Authorization: Bearer ${MOCK_API_KEY}' '${MOCK_EXTERNAL_URL}/v1/capabilities' >/dev/null" 60

# The plugin's supervisor backs off then reconnects; the new handshake
# carries a fresh nonce.
wait_until "post-restart ws handshake" \
  "docker logs --since 1m \"$(container_name universal-mock)\" 2>&1 | grep -F 'ws handshake accepted' >/dev/null" 60

log_info "driving 5 more evolutions"
evolve 5 >/dev/null
assert_universal_payments_count_at_least 6 60
