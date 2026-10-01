#!/usr/bin/env bash
set -euo pipefail

version="${1-}"
core_pattern='(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)'
identifier_pattern='[0-9A-Za-z-]+'
if [[ ! "$version" =~ ^${core_pattern}(-${identifier_pattern}(\.${identifier_pattern})*)?(\+${identifier_pattern}(\.${identifier_pattern})*)?$ ]]; then
  printf 'FCTL_PLUGIN_VERSION is not a SemVer 2.0.0 version: %s\n' "$version" >&2
  exit 2
fi

# Numeric prerelease identifiers must not contain leading zeroes. Build
# identifiers deliberately have no such restriction in SemVer 2.0.0.
without_build="${version%%+*}"
if [[ "$without_build" == *-* ]]; then
  prerelease="${without_build#*-}"
  IFS='.' read -r -a identifiers <<< "$prerelease"
  for identifier in "${identifiers[@]}"; do
    if [[ "$identifier" =~ ^[0-9]+$ && "$identifier" != "0" && "$identifier" == 0* ]]; then
      printf 'FCTL_PLUGIN_VERSION is not a SemVer 2.0.0 version: %s\n' "$version" >&2
      exit 2
    fi
  done
fi
