#!/usr/bin/env bash
# Scenario 05 — Replay protection.
#
# Build a signed hello frame in Python (websocket-client is broadly
# available), send it once (should succeed), then replay it on a fresh
# connection (should be closed 1008 by the counterparty's nonce cache).
# Asserts that the mock log records the rejection.
#
# Requires python3 + the websocket-client module to be available on the
# host (pip install websocket-client). Skipped gracefully when missing.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/../lib.sh"

# Make sure the mock is in wss mode (whichever scenario ran last may
# have left it in webhook mode). Must propagate MOCK_LOG_LEVEL so the
# log assertions further down see the debug-level "ws handshake
# rejected" line.
MOCK_LOG_LEVEL=debug MOCK_EVENT_STREAM=wss \
  docker compose "${COMPOSE_FILES[@]}" up -d --no-deps --force-recreate universal-mock >/dev/null
wait_until "mock wss healthy" \
  "curl -sSf -H 'Authorization: Bearer ${MOCK_API_KEY}' '${MOCK_EXTERNAL_URL}/v1/capabilities' | jq -e '.features.eventStream == \"wss\"' >/dev/null" 30

# Pick a python interpreter that has websocket-client. The Nix dev
# shell ships its own python which doesn't see user-installed pip
# packages, so we fall back to the system interpreter when available.
PY=""
for cand in /usr/bin/python3 /opt/homebrew/bin/python3 python3; do
  if command -v "${cand}" >/dev/null 2>&1 && "${cand}" -c 'import websocket' 2>/dev/null; then
    PY="${cand}"
    break
  fi
done
if [ -z "${PY}" ]; then
  log_warn "websocket-client not found on any python3 (tried /usr/bin/python3, /opt/homebrew/bin/python3, python3); skipping (pip install websocket-client to enable)"
  exit 0
fi
log_info "using ${PY} for replay test"

WS_URL="ws://localhost:8090/v1/stream"
NONCE="replay-nonce-$(date +%s)"
TS=$("${PY}" -c 'import datetime; print(datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"))')
SIG=$("${PY}" -c "
import hmac, hashlib, json
secret = '${MOCK_WEBHOOK_SECRET}'.encode()
ts = '${TS}'
nonce = '${NONCE}'
events = json.dumps(['payment.updated'], separators=(',', ':'))
payload = f'{ts}.{nonce}.{events}'.encode()
print(hmac.new(secret, payload, hashlib.sha256).hexdigest())
")
HELLO=$("${PY}" -c "
import json
print(json.dumps({
  'type': 'hello',
  'apiKey': '${MOCK_API_KEY}',
  'timestamp': '${TS}',
  'nonce': '${NONCE}',
  'events': ['payment.updated'],
  'signature': '${SIG}',
}))
")

log_info "first connect — must succeed"
HELLO_PY="${HELLO}" WS_URL="${WS_URL}" "${PY}" - <<'PY'
import json, os, websocket
hello = os.environ["HELLO_PY"]
ws = websocket.create_connection(os.environ["WS_URL"], subprotocols=["formance-universal-v1"], timeout=5)
ws.send(hello)
ack = json.loads(ws.recv())
assert ack.get("type") == "hello-ack", f"unexpected ack: {ack}"
ws.close()
print("first connect: accepted, ack=", ack)
PY

log_info "second connect with same nonce — must be rejected (1008 close or empty frame after close)"
HELLO_PY="${HELLO}" WS_URL="${WS_URL}" "${PY}" - <<'PY'
import os, websocket
ws = websocket.create_connection(os.environ["WS_URL"], subprotocols=["formance-universal-v1"], timeout=5)
ws.send(os.environ["HELLO_PY"])
try:
    raw = ws.recv()
    # websocket-client v1.x returns '' on a server-initiated close;
    # any non-empty payload would be a legitimate frame, which means
    # the replay was NOT rejected — that's the failure mode.
    assert raw == "", f"expected close frame, got payload: {raw!r}"
    print("second connect: server closed cleanly (empty recv) — rejection inferred")
except websocket.WebSocketException as e:
    msg = str(e).lower()
    assert "1008" in msg or "policy" in msg or "nonce" in msg, f"expected 1008/policy/nonce in error, got: {e}"
    print("second connect: rejected with WS error ->", e)
finally:
    try:
        ws.close()
    except Exception:
        pass
PY

# Mock-side proof of intent — independent of how the client surfaced
# the close (the mock SHOULD log "ws handshake rejected" with reason
# "nonce already seen" regardless).
assert_log_contains "$(container_name universal-mock)" "ws handshake rejected" 1m
