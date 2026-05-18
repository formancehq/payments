#!/usr/bin/env bash
# Scenario 04 — WS reconnect on payments-worker pod restart.
#
# The engine's OnStart re-instantiates Plugin via New() on pod restart
# but does NOT replay CreateWebhooks. The plugin's lazy ensureStreamRunning()
# (called from every FetchNext*) is responsible for re-establishing the
# supervisor using STACK_PUBLIC_URL + connectorID. This scenario proves
# that path works end-to-end.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/../lib.sh"

CID="$(install_connector mode=wss)"
wait_until "first ws handshake" \
  "docker logs --since 2m \"$(container_name universal-mock)\" 2>&1 | grep -F 'ws handshake accepted' >/dev/null" 30
evolve 5 >/dev/null
assert_universal_payments_count_at_least 3 60

log_info "restarting payments-worker container"
docker restart "$(container_name payments-worker)" >/dev/null
wait_until "payments-worker healthy" \
  "curl -sSf '${PAYMENTS_HEALTH_URL}' >/dev/null" 60

# The supervisor restarts lazily on the first FetchNext* tick after
# worker startup. Polling period is 20m by default, so we wait up to
# 90s for the engine to fire the first poll (Temporal schedules can
# kick immediately on worker restart for resumed schedules).
wait_until "post-pod-restart ws handshake" \
  "docker logs --since 2m \"$(container_name universal-mock)\" 2>&1 | grep -c 'ws handshake accepted' | awk '{ exit !(\$1 >= 2) }'" 120

log_info "driving 5 more evolutions"
evolve 5 >/dev/null
assert_universal_payments_count_at_least 6 60
