#!/usr/bin/env bash
set -euo pipefail

script_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
checker="$script_root/check-imports.sh"
fixture_root="$(mktemp -d "${TMPDIR:-/tmp}/payments-imports-test.XXXXXX")"
cleanup() { rm -rf "$fixture_root"; }
trap cleanup EXIT

write_fixture() {
  local name="$1"
  shift
  printf '%s\n' "$@" > "$fixture_root/$name"
}

expect_success() {
  local name="$1"
  "$checker" "$fixture_root/$name"
}

expect_failure() {
  local name="$1"
  if "$checker" "$fixture_root/$name" >/dev/null 2>&1; then
    printf 'import check unexpectedly accepted fixture: %s\n' "$name" >&2
    exit 1
  fi
}

write_fixture exact \
  'wasi:clocks/wall-clock@0.2.12' \
  'wasi:random/random@0.2.12' \
  'wasi:cli/environment@0.2.12' \
  'wasi:io/poll@0.2.12' \
  'wasi:clocks/monotonic-clock@0.2.12'
: > "$fixture_root/empty"
write_fixture subset 'wasi:io/poll@0.2.12'
write_fixture foreign \
  'wasi:clocks/wall-clock@0.2.12' \
  'wasi:random/random@0.2.12' \
  'wasi:cli/environment@0.2.12' \
  'wasi:io/poll@0.2.12' \
  'wasi:clocks/monotonic-clock@0.2.12' \
  'wasi:filesystem/types@0.2.12'

expect_success exact
expect_failure empty
expect_failure subset
expect_failure foreign
