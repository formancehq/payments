#!/usr/bin/env bash
###############################################################################
# Shared helpers for the Universal Connector E2E scenarios.
#
# Sourced by e2e.sh and every scenario in scenarios/. Defines a small set of
# colour-aware logging primitives plus the operations every scenario needs:
# stack up/down, connector install (webhook or wss mode), evolve, payment
# assertions, log assertions, artifact capture.
#
# Every operation is idempotent. The harness can be killed at any point and
# re-run safely.
###############################################################################
set -euo pipefail

REPO_ROOT="${REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../../../.." && pwd)}"
COMPOSE_FILES=(
  -f "${REPO_ROOT}/docker-compose.yml"
  -f "${REPO_ROOT}/docker-compose.dev-override.yml"
  -f "${REPO_ROOT}/docker-compose.universal-mock.yml"
)

# Defaults — overridable via env so CI can pin behaviour.
PAYMENTS_URL="${PAYMENTS_URL:-http://localhost:8092/api/payments}"
PAYMENTS_HEALTH_URL="${PAYMENTS_HEALTH_URL:-http://localhost:8080/_healthcheck}"
MOCK_EXTERNAL_URL="${MOCK_EXTERNAL_URL:-http://localhost:8090}"
MOCK_INTERNAL_URL="${MOCK_INTERNAL_URL:-http://universal-mock:8080}"
MOCK_API_KEY="${MOCK_API_KEY:-dev-key}"
MOCK_WEBHOOK_SECRET="${MOCK_WEBHOOK_SECRET:-dev-secret}"

# ANSI helpers — auto-disabled when stdout is not a TTY (CI logs).
if [ -t 1 ]; then
  C_RED=$'\033[0;31m'; C_GREEN=$'\033[0;32m'; C_YELLOW=$'\033[0;33m'; C_BLUE=$'\033[0;34m'; C_RESET=$'\033[0m'
else
  C_RED=""; C_GREEN=""; C_YELLOW=""; C_BLUE=""; C_RESET=""
fi

# log_pass / log_fail go to stdout so the harness summary picks them
# up. All other levels go to stderr so they NEVER contaminate a
# command-substitution capture — install_connector echoes the
# connector id on stdout and CID="$(install_connector ...)" would
# otherwise pick up info lines too.
log_info() { printf "%s[i]%s %s\n" "${C_BLUE}" "${C_RESET}" "$*" >&2; }
log_pass() { printf "%s[PASS]%s %s\n" "${C_GREEN}" "${C_RESET}" "$*"; }
log_fail() { printf "%s[FAIL]%s %s\n" "${C_RED}" "${C_RESET}" "$*"; }
log_warn() { printf "%s[w]%s %s\n" "${C_YELLOW}" "${C_RESET}" "$*" >&2; }
log_err()  { printf "%s[e]%s %s\n" "${C_RED}" "${C_RESET}" "$*" >&2; }

# wait_until polls a command until it returns 0, up to ${2:-60}s. Used
# instead of fixed sleeps so scenarios run as fast as the stack allows.
wait_until() {
  local desc="$1"; local cmd="$2"; local timeout="${3:-60}"
  local start
  start=$(date +%s)
  while true; do
    if eval "${cmd}" >/dev/null 2>&1; then
      return 0
    fi
    if [ $(( $(date +%s) - start )) -ge "${timeout}" ]; then
      log_err "timed out (${timeout}s) waiting: ${desc}"
      return 1
    fi
    sleep 1
  done
}

up_stack() {
  log_info "starting Formance + universal-mock stack (this can take a minute on first build)"
  # Debug-level mock logging so the in-scenario log assertions can
  # match the per-event "stream broadcast" / "auto-emit delivered"
  # lines (info-level only summarises). MOCK_EVENT_STREAM stays
  # blank here — scenarios opt in per-install via the env vars.
  MOCK_LOG_LEVEL=debug MOCK_EVENT_STREAM="${MOCK_EVENT_STREAM:-}" \
    docker compose "${COMPOSE_FILES[@]}" up -d --build >/dev/null
  wait_until "payments healthy" \
    "curl -sSf '${PAYMENTS_HEALTH_URL}' >/dev/null" 90
  wait_until "universal-mock healthy" \
    "curl -sSf -H 'Authorization: Bearer ${MOCK_API_KEY}' '${MOCK_EXTERNAL_URL}/v1/capabilities' >/dev/null" 60
  log_info "stack ready"
}

down_stack() {
  docker compose "${COMPOSE_FILES[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
}

# install_connector mode=<webhook|wss>
# Cleans any prior install, sets the mock's MOCK_EVENT_STREAM accordingly,
# and posts a fresh /v3/connectors/install/Universal. Echoes the connector id.
install_connector() {
  local mode="webhook"
  local name="universal-e2e"
  for arg in "$@"; do
    case "${arg}" in
      mode=*) mode="${arg#mode=}" ;;
      name=*) name="${arg#name=}" ;;
    esac
  done
  uninstall_connector_by_name "${name}"

  case "${mode}" in
    wss)
      log_info "switching mock to wss mode"
      MOCK_LOG_LEVEL=debug MOCK_EVENT_STREAM=wss \
        docker compose "${COMPOSE_FILES[@]}" up -d --no-deps --force-recreate universal-mock >/dev/null
      wait_until "universal-mock healthy (wss)" \
        "curl -sSf -H 'Authorization: Bearer ${MOCK_API_KEY}' '${MOCK_EXTERNAL_URL}/v1/capabilities' | jq -e '.features.eventStream == \"wss\"' >/dev/null" 60
      ;;
    webhook)
      log_info "switching mock to webhook-only mode"
      MOCK_LOG_LEVEL=debug MOCK_EVENT_STREAM="" \
        docker compose "${COMPOSE_FILES[@]}" up -d --no-deps --force-recreate universal-mock >/dev/null
      wait_until "universal-mock healthy (webhook)" \
        "curl -sSf -H 'Authorization: Bearer ${MOCK_API_KEY}' '${MOCK_EXTERNAL_URL}/v1/capabilities' >/dev/null" 60
      ;;
    *) log_err "unknown install mode: ${mode}"; return 2 ;;
  esac

  local body
  body=$(jq -n \
    --arg name "${name}" \
    --arg endpoint "${MOCK_INTERNAL_URL}" \
    '{name:$name, endpoint:$endpoint, apiKey:"dev-key", webhookSharedSecret:"dev-secret", pollingPeriod:"20m"}')
  local resp
  resp=$(curl -sS -X POST -H 'Content-Type: application/json' -d "${body}" \
    "${PAYMENTS_URL}/v3/connectors/install/Universal")
  local cid
  cid=$(echo "${resp}" | jq -r '.data // empty')
  if [ -z "${cid}" ]; then
    log_err "install failed: $(echo "${resp}" | jq -c .)"
    return 1
  fi
  log_info "installed connector_id=${cid} (mode=${mode})"
  echo "${cid}"
}

uninstall_connector_by_name() {
  local name="$1"
  local existing
  existing=$(curl -sS "${PAYMENTS_URL}/v3/connectors" | jq -r --arg name "${name}" '.cursor.data[]? | select(.name==$name) | .id' || true)
  if [ -n "${existing}" ]; then
    log_info "removing prior install ${existing}"
    curl -sS -X DELETE "${PAYMENTS_URL}/v3/connectors/${existing}" >/dev/null || true
    wait_until "connector ${name} removed" \
      "[ -z \"\$(curl -sS '${PAYMENTS_URL}/v3/connectors' | jq -r --arg name '${name}' '.cursor.data[]? | select(.name==\$name) | .id')\" ]" 30
  fi
}

# evolve N — drives the mock through N state transitions, returns the JSON
# response so the caller can assert on advanced/webhooksDelivered.
evolve() {
  local n="${1:-1}"
  curl -sS -X POST -H "Authorization: Bearer ${MOCK_API_KEY}" \
    "${MOCK_EXTERNAL_URL}/_admin/evolve?n=${n}"
}

# wait_for_subscriptions — block until ALL 8 webhook subscriptions
# have been POSTed to the mock AND the engine's storeWebhookConfig
# workflow activities have committed them to the DB. The latter has no
# direct external signal, so we wait for the count on the mock side
# then settle 3s for the workflow to finish persisting. Without this,
# the engine's HandleWebhook returns 500 ("webhook config not found")
# on inbound events that race the storage commit.
wait_for_subscriptions() {
  local timeout="${1:-60}"
  local want="${SUBSCRIPTION_COUNT:-8}"
  wait_until "engine registered ${want} webhook subscriptions on the mock" \
    "[ \$(docker logs --since 5m \"$(container_name universal-mock)\" 2>&1 | grep -cF 'webhook subscription created') -ge ${want} ]" \
    "${timeout}"
  # Storage-commit settle: workflow activities run sequentially after
  # the plugin returns; 3s is enough on a laptop and harmless on CI.
  sleep 3
}

# assert_universal_payments_count_at_least N [timeout=30] — polls /v3/payments
# filtered to provider=universal until the count >= N.
assert_universal_payments_count_at_least() {
  local want="$1"; local timeout="${2:-30}"
  wait_until "universal payments count >= ${want}" \
    "[ \$(curl -sS '${PAYMENTS_URL}/v3/payments?pageSize=200' | jq '[.cursor.data[]? | select(.provider==\"universal\")] | length') -ge ${want} ]" \
    "${timeout}"
}

# assert_log_contains <container> <pattern> [since=20s] — `docker logs` +
# grep -F. The since window keeps the search fast and avoids picking up
# matches from prior scenarios.
assert_log_contains() {
  local container="$1"; local pattern="$2"; local since="${3:-1m}"
  if docker logs --since "${since}" "${container}" 2>&1 | grep -F -- "${pattern}" >/dev/null; then
    return 0
  fi
  log_err "expected pattern not found in ${container} logs (since ${since}): ${pattern}"
  return 1
}

assert_log_does_not_contain() {
  local container="$1"; local pattern="$2"; local since="${3:-1m}"
  if docker logs --since "${since}" "${container}" 2>&1 | grep -F -- "${pattern}" >/dev/null; then
    log_err "unexpected pattern found in ${container} logs: ${pattern}"
    return 1
  fi
  return 0
}

# capture_logs <scenario_name> — on failure, dump every relevant
# container's logs into the artifacts dir for triage. Gateway and
# Temporal are included because webhook routing failures often surface
# at the gateway (404 on /api/payments/v3/connectors/webhooks/...) and
# CreateWebhooks workflow failures live in Temporal histories.
capture_logs() {
  local scenario="$1"
  local dir="${REPO_ROOT}/internal/connectors/plugins/public/universal/mock/e2e/artifacts/${scenario}"
  mkdir -p "${dir}"
  for svc in universal-mock payments payments-worker gateway temporal postgres; do
    container=$(docker compose "${COMPOSE_FILES[@]}" ps -q "${svc}" 2>/dev/null || true)
    if [ -n "${container}" ]; then
      docker logs "${container}" >"${dir}/${svc}.log" 2>&1 || true
    fi
  done
  log_warn "logs captured in ${dir}"
}

# container_name <service> — resolve a compose service to its container id.
container_name() {
  docker compose "${COMPOSE_FILES[@]}" ps -q "$1"
}
