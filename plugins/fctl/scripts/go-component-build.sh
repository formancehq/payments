#!/usr/bin/env bash
set -euo pipefail

# A release build injects the published SemVer through the one build-time
# variable the plugin exposes. Unset means "development build": the compiled
# default in core.Version is kept and the produced bytes are unchanged.
version_ldflag=""
if [[ -n "${FCTL_PLUGIN_VERSION:-}" ]]; then
  version_ldflag=" -X github.com/formancehq/payments/plugins/fctl/core.Version=${FCTL_PLUGIN_VERSION}"
fi
readonly version_ldflag

arguments=()
output_path=""
previous=""
for argument in "$@"; do
  if [[ "$argument" == '-ldflags=-checklinkname=0' ]]; then
    argument="-ldflags=-checklinkname=0 -buildid= -s -w${version_ldflag}"
  fi
  if [[ "$previous" == "-o" ]]; then
    output_path="$argument"
  fi
  arguments+=("$argument")
  previous="$argument"
done
go "${arguments[@]}"
if [[ -n "$output_path" ]]; then
  wasm-opt -Oz --all-features "$output_path" --output "$output_path.optimized"
  mv "$output_path.optimized" "$output_path"
fi
