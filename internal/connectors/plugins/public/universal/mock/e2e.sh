#!/usr/bin/env bash
###############################################################################
# Universal Connector — end-to-end harness.
#
# Stands up the full Formance local stack + the universal-mock counterparty,
# runs the requested scenarios sequentially, and reports PASS/FAIL per
# scenario. No Go test framework — curl + jq + docker compose only, so the
# script runs identically on a laptop and in CI.
#
# Usage:
#   ./internal/connectors/plugins/public/universal/mock/e2e.sh                       # all scenarios
#   ./internal/connectors/plugins/public/universal/mock/e2e.sh --scenarios=01,02    # subset
#   ./internal/connectors/plugins/public/universal/mock/e2e.sh --keep               # leave the stack up
#   ./internal/connectors/plugins/public/universal/mock/e2e.sh --verbose            # bash -x for the scenarios
#
# Run from the repository root.
###############################################################################
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${HERE}/../../../../../.." && pwd)"
cd "${REPO_ROOT}"

# shellcheck source=lib.sh
source "${HERE}/e2e/lib.sh"

KEEP=false
VERBOSE=false
SCENARIOS="01,02,03,04,05,06,07"

for arg in "$@"; do
  case "${arg}" in
    --keep)        KEEP=true ;;
    --verbose)     VERBOSE=true ;;
    --scenarios=*) SCENARIOS="${arg#--scenarios=}" ;;
    *) echo "unknown flag: ${arg}" >&2; exit 2 ;;
  esac
done
export VERBOSE

trap 'on_exit' EXIT
on_exit() {
  local rc=$?
  if [ "${KEEP}" = false ]; then
    log_info "tearing down stack"
    down_stack || true
  else
    log_info "leaving stack up (--keep)"
  fi
  # If a scenario failed, exit with the count (so CI gets a non-zero
  # signal). If only the trap fired (bootstrap / teardown error), keep
  # the original $? so infra failures aren't masked as success.
  if [ "${FAIL:-0}" -gt 0 ]; then
    exit "${FAIL}"
  fi
  exit "${rc}"
}

up_stack

ART_DIR="${REPO_ROOT}/internal/connectors/plugins/public/universal/mock/e2e/artifacts"
rm -rf "${ART_DIR}" && mkdir -p "${ART_DIR}"

PASS=0
FAIL=0
declare -a FAILED_SCENARIOS

IFS=',' read -ra SCEN_LIST <<< "${SCENARIOS}"
for id in "${SCEN_LIST[@]}"; do
  matches=("${HERE}"/e2e/scenarios/"${id}"-*.sh)
  if [ ! -e "${matches[0]}" ]; then
    log_err "scenario ${id} not found"
    FAIL=$((FAIL+1))
    FAILED_SCENARIOS+=("${id} (not found)")
    continue
  fi
  if [ "${#matches[@]}" -gt 1 ]; then
    log_err "scenario id ${id} matches multiple files: ${matches[*]}"
    FAIL=$((FAIL+1))
    FAILED_SCENARIOS+=("${id} (ambiguous)")
    continue
  fi
  scenario_file="${matches[0]}"
  scenario_name="$(basename "${scenario_file}" .sh)"
  log_info "──────── scenario ${scenario_name} ────────"
  runner=("bash")
  if [ "${VERBOSE}" = true ]; then
    runner=("bash" "-x")
  fi
  if "${runner[@]}" "${scenario_file}"; then
    log_pass "${scenario_name}"
    PASS=$((PASS+1))
  else
    capture_logs "${scenario_name}"
    log_fail "${scenario_name}"
    FAIL=$((FAIL+1))
    FAILED_SCENARIOS+=("${scenario_name}")
  fi
done

echo
echo "==============================================================="
echo "  Universal E2E summary: ${PASS} passed, ${FAIL} failed"
if [ "${FAIL}" -gt 0 ]; then
  printf "  Failed scenarios:\n"
  for f in "${FAILED_SCENARIOS[@]}"; do printf "    - %s\n" "${f}"; done
  printf "  Artifacts in: %s\n" "${ART_DIR}"
fi
echo "==============================================================="
# Exit handled by the EXIT trap so bootstrap and teardown failures
# also flow into the exit code.
