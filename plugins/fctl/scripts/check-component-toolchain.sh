#!/usr/bin/env bash
set -euo pipefail

check_version() {
  local tool="$1"
  local expected="$2"
  local actual
  actual="$($tool --version)"
  if [[ "$actual" != "$expected" ]]; then
    printf '%s version mismatch: got %q, want %q\n' "$tool" "$actual" "$expected" >&2
    exit 1
  fi
}

check_version componentize-go "componentize-go 0.4.1"
check_version wasi-virt "wasi-virt 0.2.0"
check_version wasm-tools "wasm-tools 1.239.0"
check_version wasm-opt "wasm-opt version 124"
