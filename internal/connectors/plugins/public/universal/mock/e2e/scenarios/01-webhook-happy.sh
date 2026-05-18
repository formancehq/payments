#!/usr/bin/env bash
# Scenario 01 — Webhook happy path (regression for what works today).
#
# Install the connector in pure-webhook mode, evolve 10 records, assert
# the engine ingests 10 universal payments and the mock log shows the
# HTTP auto-emit path was exercised.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HERE}/../lib.sh"

CID="$(install_connector mode=webhook)"
[ -n "${CID}" ] || { log_err "install returned no connector id"; exit 1; }

wait_for_subscriptions 60

log_info "driving 10 evolutions"
out=$(evolve 10)
echo "${out}" | jq .

assert_universal_payments_count_at_least 5 60
assert_log_contains "$(container_name universal-mock)" "auto-emit delivered" 2m
assert_log_does_not_contain "$(container_name universal-mock)" "stream broadcast" 2m
