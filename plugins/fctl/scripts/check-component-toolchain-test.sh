#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT

make_tool() {
  local name="$1"
  local version="$2"
  printf '#!/bin/sh\nprintf "%%s\\n" "%s"\n' "$version" > "$fixture/$name"
  chmod +x "$fixture/$name"
}

make_tool componentize-go "componentize-go 0.4.1"
make_tool wasi-virt "wasi-virt 0.2.0"
make_tool wasm-tools "wasm-tools 1.239.0"
make_tool wasm-opt "wasm-opt version 124"
PATH="$fixture:$PATH" "$script_dir/check-component-toolchain.sh"

make_tool wasm-opt "wasm-opt version 125"
if PATH="$fixture:$PATH" "$script_dir/check-component-toolchain.sh" >/dev/null 2>&1; then
  echo "toolchain guard accepted a mismatched wasm-opt" >&2
  exit 1
fi
