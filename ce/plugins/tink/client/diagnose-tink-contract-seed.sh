#!/usr/bin/env bash
# Diagnose the Tink contract "seeded-user read failed with a 4xx" CI failure by
# walking the exact call chain client.ListAccounts uses, printing the HTTP
# status of each step so we can tell WHICH of the two causes it is:
#
#   * MISMATCH  — step 2 (authorization-grant) 4xxs: TINK_CONTRACT_SEEDED_USER_ID
#                 does not exist under this Console app (wrong app, or user never
#                 created). Fix: seed the user under THESE credentials.
#   * DECAY     — steps 1-3 succeed but step 4 (GET /data/v2/accounts) 4xxs or
#                 returns zero accounts: the Tink Demo Bank consent has expired.
#                 Fix: re-run seed-tink-contract-user.sh and re-link the Demo Bank.
#
# Run with the SAME credentials CI uses (this is the whole point — it proves
# whether the CI secrets themselves work):
#
#   TINK_CONTRACT_CLIENT_ID=... TINK_CONTRACT_CLIENT_SECRET=... \
#     TINK_CONTRACT_SEEDED_USER_ID=... ./diagnose-tink-contract-seed.sh
set -euo pipefail

API="https://api.tink.com"
: "${TINK_CONTRACT_CLIENT_ID:?TINK_CONTRACT_CLIENT_ID must be set}"
: "${TINK_CONTRACT_CLIENT_SECRET:?TINK_CONTRACT_CLIENT_SECRET must be set}"
: "${TINK_CONTRACT_SEEDED_USER_ID:?TINK_CONTRACT_SEEDED_USER_ID must be set}"
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

EXT_ID="$TINK_CONTRACT_SEEDED_USER_ID"

# Mirrors the ListAccounts user-token scope set (user_access_token.go / accounts.go).
USER_SCOPES="accounts:read,transactions:read,user:read,credentials:read,providers:read"

hr() { printf '%s\n' "----------------------------------------------------------------"; }

echo "seeded external_user_id = $EXT_ID"
hr
echo "STEP 1  app access token (client_credentials, scope=authorization:grant)"
TOKEN_HTTP=$(curl -sS -o /tmp/tink_diag_1 -w '%{http_code}' "$API/api/v1/oauth/token" \
  -d client_id="$TINK_CONTRACT_CLIENT_ID" \
  -d client_secret="$TINK_CONTRACT_CLIENT_SECRET" \
  -d grant_type=client_credentials \
  -d scope=authorization:grant,authorization:read)
echo "  HTTP $TOKEN_HTTP"
if [ "$TOKEN_HTTP" != "200" ]; then
  echo "  -> app credentials themselves are rejected. The CI secrets"
  echo "     TINK_CONTRACT_CLIENT_ID/SECRET are wrong or lack authorization:grant."
  cat /tmp/tink_diag_1; echo; exit 0
fi
APP_TOKEN=$(jq -r '.access_token // empty' </tmp/tink_diag_1)
echo "  app token acquired"
hr

echo "STEP 2  delegated authorization-grant for external_user_id (the connector's"
echo "        getAuthorizationGrantCode) — 4xx here == SEED/APP MISMATCH"
GRANT_HTTP=$(curl -sS -o /tmp/tink_diag_2 -w '%{http_code}' "$API/api/v1/oauth/authorization-grant" \
  -H "Authorization: Bearer $APP_TOKEN" \
  --data-urlencode "external_user_id=$EXT_ID" \
  --data-urlencode "scope=$USER_SCOPES")
echo "  HTTP $GRANT_HTTP"
if [ "$GRANT_HTTP" != "200" ]; then
  echo "  -> DIAGNOSIS: MISMATCH. This external_user_id does not exist under this"
  echo "     Console app (or the scopes aren't granted). Response:"
  cat /tmp/tink_diag_2; echo
  echo "  FIX: run seed-tink-contract-user.sh with THESE credentials to create"
  echo "       and link the user, then set TINK_CONTRACT_SEEDED_USER_ID to it."
  exit 0
fi
CODE=$(jq -r '.code // empty' </tmp/tink_diag_2)
echo "  authorization code acquired (user exists under this app)"
hr

echo "STEP 3  token exchange (authorization_code -> user access token)"
UTOK_HTTP=$(curl -sS -o /tmp/tink_diag_3 -w '%{http_code}' "$API/api/v1/oauth/token" \
  -d client_id="$TINK_CONTRACT_CLIENT_ID" \
  -d client_secret="$TINK_CONTRACT_CLIENT_SECRET" \
  -d grant_type=authorization_code \
  -d code="$CODE")
echo "  HTTP $UTOK_HTTP"
if [ "$UTOK_HTTP" != "200" ]; then
  echo "  -> token exchange failed. Response:"; cat /tmp/tink_diag_3; echo; exit 0
fi
USER_TOKEN=$(jq -r '.access_token // empty' </tmp/tink_diag_3)
echo "  user access token acquired"
hr

echo "STEP 4  GET /data/v2/accounts (the connector's ListAccounts) — 4xx or zero"
echo "        accounts here == DEMO BANK CONSENT DECAYED"
ACC_HTTP=$(curl -sS -o /tmp/tink_diag_4 -w '%{http_code}' \
  -H "Authorization: Bearer $USER_TOKEN" \
  "$API/data/v2/accounts")
echo "  HTTP $ACC_HTTP"
if [ "$ACC_HTTP" != "200" ]; then
  echo "  -> DIAGNOSIS: DECAY (or account read scope missing). Response:"
  cat /tmp/tink_diag_4; echo
  echo "  FIX: re-run seed-tink-contract-user.sh and re-link the Tink Demo Bank."
  exit 0
fi
N=$(jq '.accounts | length' </tmp/tink_diag_4)
echo "  accounts returned: $N"
if [ "${N:-0}" -eq 0 ]; then
  echo "  -> DIAGNOSIS: DECAY. Auth chain is fine but the user has no linked"
  echo "     accounts — the Demo Bank connection lapsed."
  echo "  FIX: re-run seed-tink-contract-user.sh and re-link the Tink Demo Bank."
else
  echo "  -> All four steps succeed with $N account(s). The seed is healthy for"
  echo "     these credentials; if CI still fails, CI is using DIFFERENT secrets"
  echo "     than the ones you ran this with."
  jq -r '.accounts[] | "     account " + .id' </tmp/tink_diag_4
fi
