#!/bin/sh

# Regenerates the build-tag-split plugin wiring files:
#   internal/connectors/plugins/registry/generated_ce.go  (//go:build !ee)
#   internal/connectors/plugins/registry/generated_ee.go  (//go:build ee)
#
# Run from the repository root (or any subdirectory — the script locates the root itself).

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

PUBLIC_DIR="$REPO_ROOT/internal/connectors/plugins/public"
EE_DIR="$REPO_ROOT/ee/plugins"
OUT_CE="$REPO_ROOT/internal/connectors/plugins/registry/generated_ce.go"
OUT_EE="$REPO_ROOT/internal/connectors/plugins/registry/generated_ee.go"
MOD="github.com/formancehq/payments"

# ── gather plugin lists ───────────────────────────────────────────────────────
ce_plugins=$(find "$PUBLIC_DIR" -mindepth 1 -maxdepth 1 -type d | sort | while read -r d; do basename "$d"; done)
ee_plugins=$(find "$EE_DIR"     -mindepth 1 -maxdepth 1 -type d | sort | while read -r d; do basename "$d"; done)

# ── helpers ───────────────────────────────────────────────────────────────────

# emit one import line per plugin name
# $1 = import path prefix (e.g. "github.com/formancehq/payments/internal/connectors/plugins/public")
# reads plugin names from stdin (one per line)
emit_imports() {
    prefix="$1"
    while IFS= read -r name; do
        printf '\t%s "%s/%s"\n' "$name" "$prefix" "$name"
    done
}

# emit one map entry per plugin name
# dummypay maps to DummyPSPName (a registry-package constant) rather than dummypay.ProviderName
# reads plugin names from stdin (one per line)
emit_entries() {
    while IFS= read -r name; do
        if [ "$name" = "dummypay" ]; then
            printf '\t\tDummyPSPName:               %s.Registration,\n' "$name"
        else
            printf '\t\t%s.ProviderName:\t%s.Registration,\n' "$name" "$name"
        fi
    done
}

# ── generated_ce.go ───────────────────────────────────────────────────────────
{
    printf '//go:build !ee\n\npackage registry\n\nimport (\n'
    printf '%s\n' "$ce_plugins" | emit_imports "$MOD/internal/connectors/plugins/public"
    printf '\tpkgplugins "%s/pkg/domain/plugins"\n)\n\nfunc init() {\n\tload(map[string]pkgplugins.Registration{\n' "$MOD"
    printf '%s\n' "$ce_plugins" | emit_entries
    printf '\t})\n}\n'
} > "$OUT_CE"

# ── generated_ee.go ───────────────────────────────────────────────────────────
{
    printf '//go:build ee\n\npackage registry\n\nimport (\n'
    printf '%s\n' "$ce_plugins" | emit_imports "$MOD/internal/connectors/plugins/public"
    printf '%s\n' "$ee_plugins" | emit_imports "$MOD/ee/plugins"
    printf '\tpkgplugins "%s/pkg/domain/plugins"\n)\n\nfunc init() {\n\tload(map[string]pkgplugins.Registration{\n' "$MOD"
    printf '%s\n' "$ce_plugins" | emit_entries
    printf '%s\n' "$ee_plugins" | emit_entries
    printf '\t})\n}\n'
} > "$OUT_EE"

gofmt -w "$OUT_CE" "$OUT_EE"

echo "Regenerated:"
echo "  $OUT_CE"
echo "  $OUT_EE"
