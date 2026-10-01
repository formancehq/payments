#!/usr/bin/env bash
set -euo pipefail

[[ "$#" -eq 1 ]] || {
  printf 'usage: %s <imports-file>\n' "$0" >&2
  exit 2
}

readonly required_imports=(
  'wasi:clocks/wall-clock@0.2.12'
  'wasi:random/random@0.2.12'
  'wasi:cli/environment@0.2.12'
  'wasi:io/poll@0.2.12'
  'wasi:clocks/monotonic-clock@0.2.12'
)

import_count=0
while IFS= read -r imported_interface; do
  ((import_count += 1))
  case "$imported_interface" in
    wasi:clocks/wall-clock@0.2.12|wasi:random/random@0.2.12|wasi:cli/environment@0.2.12|wasi:io/poll@0.2.12|wasi:clocks/monotonic-clock@0.2.12)
      ;;
    *)
      printf 'component imports non-allowlisted interface: %s\n' "$imported_interface" >&2
      exit 1
      ;;
  esac
done < "$1"

for required_import in "${required_imports[@]}"; do
  grep -Fqx -- "$required_import" "$1" || {
    printf 'component is missing required import: %s\n' "$required_import" >&2
    exit 1
  }
done

[[ "$import_count" -eq "${#required_imports[@]}" ]] || {
  printf 'component must import exactly %d interfaces, found %d\n' \
    "${#required_imports[@]}" "$import_count" >&2
  exit 1
}
