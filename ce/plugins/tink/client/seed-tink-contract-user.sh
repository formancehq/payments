#!/usr/bin/env bash
# One-time seeding helper for the Tink contract test (see
# .omc/specs/contract-tests-tink.md). Creates a persistent Tink user and opens
# the Tink Link flow so a human can connect the Tink Demo Bank to it. The
# resulting external_user_id becomes TINK_CONTRACT_SEEDED_USER_ID.
#
# Usage:
#   TINK_CONTRACT_CLIENT_ID=... TINK_CONTRACT_CLIENT_SECRET=... \
#     ./seed-tink-contract-user.sh [external_user_id]
#
# The user is created only if missing, so re-running is safe (e.g. to re-link
# after a seed decays). NEVER delete this user — it IS the seed.
set -euo pipefail

API="https://api.tink.com"
EXT_ID="${1:-formance-contract-seed}"
MARKET="FR"
LOCALE="en_US"
REDIRECT_URI="https://google.com"

: "${TINK_CONTRACT_CLIENT_ID:?TINK_CONTRACT_CLIENT_ID must be set}"
: "${TINK_CONTRACT_CLIENT_SECRET:?TINK_CONTRACT_CLIENT_SECRET must be set}"
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

fail() { # fail <step> <response>
  echo "FAILED at $1 — response was:" >&2
  echo "$2" >&2
  exit 1
}

echo "==> 1/3 app access token (client credentials)"
TOKEN_RESP=$(curl -sS "$API/api/v1/oauth/token" \
  -d client_id="$TINK_CONTRACT_CLIENT_ID" \
  -d client_secret="$TINK_CONTRACT_CLIENT_SECRET" \
  -d grant_type=client_credentials \
  -d scope=user:create,authorization:grant)
TOKEN=$(jq -r '.access_token // empty' <<<"$TOKEN_RESP")
[ -n "$TOKEN" ] || fail "token fetch (check creds + user:create/authorization:grant scopes on the Console app)" "$TOKEN_RESP"

echo "==> 2/3 create user external_user_id=$EXT_ID (skipped if it already exists)"
CREATE_RESP=$(curl -sS "$API/api/v1/user/create" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"external_user_id\":\"$EXT_ID\",\"market\":\"$MARKET\",\"locale\":\"$LOCALE\"}")
if jq -e '.user_id' <<<"$CREATE_RESP" >/dev/null 2>&1; then
  echo "    created: $(jq -r '.user_id' <<<"$CREATE_RESP")"
elif grep -qi 'exist' <<<"$CREATE_RESP"; then
  echo "    already exists, reusing"
else
  fail "user create" "$CREATE_RESP"
fi

echo "==> 3/3 delegated authorization code for Tink Link"
# actor_client_id is Tink's fixed constant for Tink Link, the same one the
# connector hardcodes in client/create_temporary_auth_code.go.
CODE_RESP=$(curl -sS "$API/api/v1/oauth/authorization-grant/delegate" \
  -H "Authorization: Bearer $TOKEN" \
  -d external_user_id="$EXT_ID" \
  -d id_hint=contract-seed \
  -d actor_client_id=df05e4b379934cd09963197cc855bfe9 \
  -d 'scope=authorization:read,authorization:grant,credentials:refresh,credentials:read,credentials:write,providers:read,user:read')
CODE=$(jq -r '.code // empty' <<<"$CODE_RESP")
[ -n "$CODE" ] || fail "delegate code" "$CODE_RESP"

URL="https://link.tink.com/1.0/transactions/connect-accounts"
URL+="?client_id=$TINK_CONTRACT_CLIENT_ID"
URL+="&state=seed"
URL+="&authorization_code=$CODE"
URL+="&market=$MARKET"
URL+="&locale=$LOCALE"
URL+="&test=true"
URL+="&redirect_uri=$REDIRECT_URI"

echo
echo "Opening Tink Link (code is short-lived — finish the flow promptly):"
echo "$URL"
echo
echo "In the browser: pick the Tink Demo Bank test provider and log in with"
echo "its demo credentials. After the redirect, the connection is linked."
echo
echo "Then export for the contract test:"
echo "  TINK_CONTRACT_SEEDED_USER_ID=$EXT_ID"

if command -v open >/dev/null; then open "$URL"; fi
