#!/usr/bin/env bash
# Scenario 02 — WebSocket happy path (the core new scenario).
#
# Install with eventStream=wss, verify the supervisor connects, evolve
# 10 records, assert the engine ingests them AND the mock log shows the
# WS broadcast path was exercised INSTEAD OF the HTTP webhook callback.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/../lib.sh"

CID="$(install_connector mode=wss)"
[ -n "${CID}" ] || { log_err "install returned no connector id"; exit 1; }

log_info "waiting for stream supervisor to connect"
wait_until "ws handshake accepted on mock" \
  "docker logs --since 2m \"$(container_name universal-mock)\" 2>&1 | grep -F 'ws handshake accepted' >/dev/null" \
  30

log_info "driving 10 evolutions"
out=$(evolve 10)
echo "${out}" | jq .

# Universal payments should land via the WS-loopback path.
assert_universal_payments_count_at_least 5 60

# Stream MUST have been the transport for at least some events; the
# HTTP webhook fallback for the same install MUST NOT have fired.
assert_log_contains "$(container_name universal-mock)" "stream broadcast" 2m
assert_log_does_not_contain "$(container_name universal-mock)" "auto-emit delivered" 30s
