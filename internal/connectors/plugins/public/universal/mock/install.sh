#!/usr/bin/env bash
# Installs the Universal CE Connector against the docker-compose-bundled mock
# counterparty. Idempotent — re-running uninstalls + reinstalls cleanly.
#
# Run from the repo root after:
#   docker compose \
#     -f docker-compose.yml \
#     -f docker-compose.dev-override.yml \
#     -f docker-compose.universal-mock.yml \
#     up -d --build

set -euo pipefail

# Default to the docker-compose gateway URL (Caddy on host port 8092).
# The gateway strips the /api/payments prefix before reverse-proxying to
# the payments service, so all v3 API calls go via the gateway.
PAYMENTS_URL="${PAYMENTS_URL:-http://localhost:8092/api/payments}"
PAYMENTS_HEALTHCHECK_URL="${PAYMENTS_HEALTHCHECK_URL:-http://localhost:8080/_healthcheck}"
MOCK_INTERNAL_URL="${MOCK_INTERNAL_URL:-http://universal-mock:8080}"
MOCK_EXTERNAL_URL="${MOCK_EXTERNAL_URL:-http://localhost:8090}"
CONNECTOR_NAME="${CONNECTOR_NAME:-universal-mock}"

echo "==> Sanity-checking that everything is reachable"
curl -sSf "${PAYMENTS_HEALTHCHECK_URL}" >/dev/null || {
  echo "ERR: payments service unreachable at ${PAYMENTS_HEALTHCHECK_URL}"
  echo "     start it with:"
  echo "     docker compose -f docker-compose.yml -f docker-compose.dev-override.yml -f docker-compose.universal-mock.yml up -d --build"
  exit 1
}
curl -sSf -H 'Authorization: Bearer dev-key' "${MOCK_EXTERNAL_URL}/v1/capabilities" >/dev/null || {
  echo "ERR: universal-mock unreachable at ${MOCK_EXTERNAL_URL}"
  exit 1
}
echo "    payments OK"
echo "    universal-mock OK"

echo
echo "==> Cleaning up any previous install named ${CONNECTOR_NAME}"
existing=$(curl -sS "${PAYMENTS_URL}/v3/connectors" | jq -r --arg name "${CONNECTOR_NAME}" '.cursor.data[] | select(.name==$name) | .id' || echo "")
if [ -n "${existing}" ]; then
  echo "    found existing ${existing}, uninstalling"
  curl -sS -X DELETE "${PAYMENTS_URL}/v3/connectors/${existing}" >/dev/null
  # Engine uninstall is async; poll until gone.
  for i in $(seq 1 30); do
    still=$(curl -sS "${PAYMENTS_URL}/v3/connectors" | jq -r --arg name "${CONNECTOR_NAME}" '.cursor.data[] | select(.name==$name) | .id' || echo "")
    [ -z "${still}" ] && break
    sleep 1
  done
fi

echo
echo "==> Installing Universal connector against ${MOCK_INTERNAL_URL}"
install_resp=$(curl -sS -X POST \
  -H 'Content-Type: application/json' \
  -d "$(cat <<EOF
{
  "name":                "${CONNECTOR_NAME}",
  "endpoint":            "${MOCK_INTERNAL_URL}",
  "apiKey":              "dev-key",
  "webhookSharedSecret": "dev-secret",
  "pollingPeriod":       "20m"
}
EOF
)" \
  "${PAYMENTS_URL}/v3/connectors/install/Universal")

connector_id=$(echo "${install_resp}" | jq -r '.data')
if [ -z "${connector_id}" ] || [ "${connector_id}" = "null" ]; then
  echo "ERR: install failed:"
  echo "${install_resp}" | jq .
  exit 1
fi
echo "    installed connector_id=${connector_id}"

echo
echo "==> Sleeping 10s so the install workflow runs (FetchAccounts bootstrap + first poll)"
sleep 10

echo
echo "==> Status snapshot (filtered to provider=universal — other connectors may be present)"
for ep in accounts payments; do
  echo "--- ${ep} ---"
  curl -sS "${PAYMENTS_URL}/v3/${ep}?pageSize=200" | \
    jq --arg name "${CONNECTOR_NAME}" \
      '{universal: [.cursor.data[] | select(.provider=="universal")] | length}'
done

echo
echo "==> Useful follow-ups"
cat <<EOF

  # Watch the dataset evolve + auto-emit webhooks to the engine:
  curl -sS -X POST -H 'Authorization: Bearer dev-key' \\
    '${MOCK_EXTERNAL_URL}/_admin/evolve?n=100'
  # → {"advanced":100,"webhooksDelivered":<N>}

  # Push a single synthetic payment.updated webhook into Formance:
  curl -sS -X POST -H 'Authorization: Bearer dev-key' \\
    '${MOCK_EXTERNAL_URL}/_admin/trigger-webhook?name=payment.updated'

  # Find a universal payment and inspect its adjustment history:
  REF=\$(curl -sS '${PAYMENTS_URL}/v3/payments?pageSize=200' | \\
    jq -r '.cursor.data[] | select(.provider=="universal") | .id' | head -1)
  curl -sS "${PAYMENTS_URL}/v3/payments/\${REF}" | \\
    jq '{reference,status,adjustments:.adjustments|map({status,amount})}'

  # Tail logs:
  docker logs -f payments-universal-mock-1
  docker logs -f payments-payments-worker-1   # Temporal activities
  docker logs -f payments-payments-1          # HTTP requests + webhook verification

  # Temporal UI (workflow inspector):
  open http://localhost:8081

  # Console (Formance UI):
  open http://localhost:3000

EOF
