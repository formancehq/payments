#!/usr/bin/env bash
set -euo pipefail

readonly plugin_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
readonly wrapper="$plugin_root/scripts/with-fctl-sdk.sh"
readonly expected_nar_hash='sha256-DnTiEFya3R9KCYmgv5SO/1StKTCPmndObQrrVHf79Xk='
readonly expected_bundle_nar_hash='sha256-pDXwGgWba9XYBbkZfoYKTYC2D+3mDjDIw1e+gSHzpc8='
readonly expected_bundle_path='sdk/fctl-v2-poc'
readonly expected_wit_hash='38fdf377264eeada82b23fef153e6bf106ed0624e8916ff62cabdf210d6255f5'
readonly expected_commit='e9b1395f46f3100b381dbe00f5213de28e6df0e1'
readonly expected_repository='https://github.com/formancehq/fctl-v2-poc.git'

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

expect_failure() {
  local expected_message="$1"
  shift
  local stderr_path="$test_root/stderr"
  if "$@" >"$test_root/stdout" 2>"$stderr_path"; then
    fail "command unexpectedly succeeded: $*"
  fi
  grep -F "$expected_message" "$stderr_path" >/dev/null || {
    sed -n '1,120p' "$stderr_path" >&2
    fail "missing expected diagnostic: $expected_message"
  }
}

test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT
# The wrapper reports canonical paths; compare against canonical paths too.
test_root="$(cd "$test_root" && pwd -P)"
sdk_root="$test_root/fctl-source"
archive_root="$test_root/archive-source"
fake_bin="$test_root/bin"
mkdir -p \
  "$sdk_root/pkg/plugin" "$sdk_root/wit/formance/fctl/plugin/v1" \
  "$archive_root/pkg/plugin" "$archive_root/wit/formance/fctl/plugin/v1" \
  "$fake_bin"
sdk_root="$(cd "$sdk_root" && pwd -P)"
archive_root="$(cd "$archive_root" && pwd -P)"
printf 'module github.com/formancehq/fctl-v2-poc/pkg/plugin\n\ngo 1.25.0\n' >"$sdk_root/pkg/plugin/go.mod"
cp "$plugin_root/wit/plugin.wit" "$sdk_root/wit/formance/fctl/plugin/v1/plugin.wit"
cp "$sdk_root/pkg/plugin/go.mod" "$archive_root/pkg/plugin/go.mod"
cp "$plugin_root/wit/plugin.wit" "$archive_root/wit/formance/fctl/plugin/v1/plugin.wit"

cat >"$fake_bin/nix" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ "$#" -eq 8 ]] || { printf 'unexpected nix argument count: %s\n' "$#" >&2; exit 97; }
[[ "$1" == '--extra-experimental-features' && "$2" == 'nix-command' && "$3" == 'hash' && "$4" == 'path' && "$5" == '--type' && "$6" == 'sha256' && "$7" == '--sri' ]] || {
  printf 'unexpected nix arguments: %s\n' "$*" >&2
  exit 97
}
[[ "$8" == */pkg/plugin ]] || { printf 'nix hashed wrong path: %s\n' "$8" >&2; exit 97; }
if [[ -n "${REJECT_HASH_PATH:-}" && "$8" == "$REJECT_HASH_PATH" ]]; then
  printf 'nix hashed polluted checkout path: %s\n' "$8" >&2
  exit 97
fi
printf '%s\n' "${FAKE_NAR_HASH:?}"
EOF
chmod +x "$fake_bin/nix"

cat >"$fake_bin/git" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$3" == 'rev-parse' && "$4" == 'HEAD' ]]; then
  printf '%s\n' "${FAKE_GIT_HEAD:?}"
elif [[ "$3" == 'remote' && "$4" == 'get-url' && "$5" == 'origin' ]]; then
  printf '%s\n' "${FAKE_GIT_REMOTE:?}"
elif [[ "$3" == 'archive' ]]; then
  [[ "$4" == "${FAKE_GIT_HEAD:?}" ]] || { printf 'archived wrong commit: %s\n' "$4" >&2; exit 98; }
  [[ "$5" == 'pkg/plugin' && "$6" == 'wit/formance/fctl/plugin/v1/plugin.wit' ]] || {
    printf 'archived wrong paths: %s %s\n' "$5" "$6" >&2
    exit 98
  }
  tar -C "${FAKE_GIT_ARCHIVE_SOURCE:?}" -cf - "$5" "$6"
else
  printf 'unexpected git arguments: %s\n' "$*" >&2
  exit 98
fi
EOF
chmod +x "$fake_bin/git"

cat >"$test_root/inspect-workspace.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ -f "${GOWORK:?}" ]] || { printf 'missing temporary go.work\n' >&2; exit 91; }
[[ "${FCTL_SDK_ROOT:?}" != "$REJECTED_SDK_ROOT" ]] || { printf 'command received polluted checkout root\n' >&2; exit 91; }
resolved="$(go list -m -f '{{.Dir}}' github.com/formancehq/fctl-v2-poc/pkg/plugin)"
[[ "$resolved" != "$REJECTED_SDK_DIR" ]] || { printf 'workspace resolved polluted SDK directory\n' >&2; exit 91; }
[[ "$resolved" == "$FCTL_SDK_ROOT/pkg/plugin" ]] || { printf 'workspace and projected SDK root disagree\n' >&2; exit 91; }
[[ ! -e "$resolved/tampered.go" ]] || { printf 'workspace consumed dirty ignored source\n' >&2; exit 91; }
wit_hash="$(shasum -a 256 "$FCTL_SDK_ROOT/wit/formance/fctl/plugin/v1/plugin.wit" | awk '{print $1}')"
[[ "$wit_hash" == '38fdf377264eeada82b23fef153e6bf106ed0624e8916ff62cabdf210d6255f5' ]] || { printf 'command consumed dirty WIT source\n' >&2; exit 91; }
go list -m all >/dev/null
printf '%s\n' "$GOWORK" >"$CAPTURE_PATH"
printf '%s\n' "$resolved" >"$CAPTURE_SDK_PATH"
cp "$GOWORK" "$CAPTURE_CONTENT"
EOF
chmod +x "$test_root/inspect-workspace.sh"

cat >"$test_root/fail-after-capture.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "${GOWORK:?}" >"$CAPTURE_PATH"
exit 42
EOF
chmod +x "$test_root/fail-after-capture.sh"

cat >"$test_root/signal-wrapper.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "${GOWORK:?}" >"$CAPTURE_PATH"
kill -TERM "$PPID"
EOF
chmod +x "$test_root/signal-wrapper.sh"

[[ -x "$wrapper" ]] || fail "wrapper is missing or not executable: $wrapper"

# An unset FCTL_SDK_ROOT is not a failure: the wrapper falls back to the
# minimal SDK snapshot committed with this plugin, so a clean checkout with no
# credential and no fctl clone still gets an exact, hash-verified SDK. This
# case deliberately runs against the real nix so it proves the committed
# snapshot, not a fixture.
env -u FCTL_SDK_ROOT "$wrapper" true

# The snapshot is held to its own content hash. The upstream checkout hash
# covers the whole SDK module and must not unlock the measured subset.
expect_failure 'bundled fctl SDK content hash mismatch' env -u FCTL_SDK_ROOT \
  PATH="$fake_bin:$PATH" FAKE_NAR_HASH="$expected_nar_hash" "$wrapper" true

expect_failure 'bundled fctl SDK content hash mismatch' env -u FCTL_SDK_ROOT \
  PATH="$fake_bin:$PATH" FAKE_NAR_HASH='sha256-wrong' "$wrapper" true

expect_failure 'FCTL_SDK_ROOT is not a directory' env \
  PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$test_root/absent" FAKE_NAR_HASH="$expected_nar_hash" "$wrapper" true

expect_failure 'fctl SDK content hash mismatch' env \
  PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH='sha256-wrong' "$wrapper" true

printf '\n// wrong WIT\n' >>"$sdk_root/wit/formance/fctl/plugin/v1/plugin.wit"
expect_failure 'fctl SDK WIT hash mismatch' env \
  PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH="$expected_nar_hash" "$wrapper" true
cp "$plugin_root/wit/plugin.wit" "$sdk_root/wit/formance/fctl/plugin/v1/plugin.wit"

printf 'module example.invalid/wrong\n\ngo 1.25.0\n' >"$sdk_root/pkg/plugin/go.mod"
expect_failure 'fctl SDK module path mismatch' env \
  PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH="$expected_nar_hash" "$wrapper" true
printf 'module github.com/formancehq/fctl-v2-poc/pkg/plugin\n\ngo 1.25.0\n' >"$sdk_root/pkg/plugin/go.mod"

mkdir "$sdk_root/.git"
expect_failure 'fctl SDK checkout revision mismatch' env \
  PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH="$expected_nar_hash" \
  FAKE_GIT_HEAD='0000000000000000000000000000000000000000' FAKE_GIT_REMOTE="$expected_repository" \
  "$wrapper" true
expect_failure 'fctl SDK checkout remote mismatch' env \
  PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH="$expected_nar_hash" \
  FAKE_GIT_HEAD="$expected_commit" FAKE_GIT_REMOTE='https://example.invalid/fctl.git' \
  "$wrapper" true

# The checked-out tree is deliberately dirty and polluted. A Git source must
# project the locked commit, so none of this working-tree content is consumed.
printf 'module example.invalid/dirty-checkout\n\ngo 1.25.0\n' >"$sdk_root/pkg/plugin/go.mod"
printf 'package tampered\n' >"$sdk_root/pkg/plugin/tampered.go"
printf '\n// dirty checkout WIT\n' >>"$sdk_root/wit/formance/fctl/plugin/v1/plugin.wit"
capture_path="$test_root/workspace-path"
capture_sdk_path="$test_root/sdk-path"
capture_content="$test_root/workspace-content"
env PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH="$expected_nar_hash" \
  FAKE_GIT_HEAD="$expected_commit" FAKE_GIT_REMOTE="$expected_repository" \
  FAKE_GIT_ARCHIVE_SOURCE="$archive_root" REJECT_HASH_PATH="$sdk_root/pkg/plugin" \
  REJECTED_SDK_ROOT="$sdk_root" REJECTED_SDK_DIR="$sdk_root/pkg/plugin" CAPTURE_PATH="$capture_path" CAPTURE_SDK_PATH="$capture_sdk_path" CAPTURE_CONTENT="$capture_content" \
  "$wrapper" "$test_root/inspect-workspace.sh"
workspace_path="$(cat "$capture_path")"
projected_sdk_path="$(cat "$capture_sdk_path")"
[[ ! -e "$workspace_path" ]] || fail "temporary go.work was not cleaned up: $workspace_path"
[[ ! -e "$projected_sdk_path" ]] || fail "projected Git SDK was not cleaned up: $projected_sdk_path"
grep -F "$plugin_root" "$capture_content" >/dev/null || fail 'go.work omits the Payments plugin module'
grep -F "$projected_sdk_path" "$capture_content" >/dev/null || fail 'go.work omits the projected fctl SDK module'

if env PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH="$expected_nar_hash" \
  FAKE_GIT_HEAD="$expected_commit" FAKE_GIT_REMOTE="$expected_repository" \
  FAKE_GIT_ARCHIVE_SOURCE="$archive_root" REJECT_HASH_PATH="$sdk_root/pkg/plugin" \
  CAPTURE_PATH="$capture_path" "$wrapper" "$test_root/fail-after-capture.sh"; then
  fail 'wrapper swallowed the command failure'
fi
failed_workspace_path="$(cat "$capture_path")"
[[ ! -e "$failed_workspace_path" ]] || fail "failed command left temporary go.work behind: $failed_workspace_path"

if env PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH="$expected_nar_hash" \
  FAKE_GIT_HEAD="$expected_commit" FAKE_GIT_REMOTE="$expected_repository" \
  FAKE_GIT_ARCHIVE_SOURCE="$archive_root" REJECT_HASH_PATH="$sdk_root/pkg/plugin" \
  CAPTURE_PATH="$capture_path" "$wrapper" "$test_root/signal-wrapper.sh"; then
  fail 'wrapper survived SIGTERM from its command'
fi
signalled_workspace_path="$(cat "$capture_path")"
[[ ! -e "$signalled_workspace_path" ]] || fail "SIGTERM left temporary go.work behind: $signalled_workspace_path"

# A content-addressed source without VCS metadata is valid and must still be exact.
rmdir "$sdk_root/.git"
cp "$archive_root/pkg/plugin/go.mod" "$sdk_root/pkg/plugin/go.mod"
cp "$archive_root/wit/formance/fctl/plugin/v1/plugin.wit" "$sdk_root/wit/formance/fctl/plugin/v1/plugin.wit"
unlink "$sdk_root/pkg/plugin/tampered.go"
env PATH="$fake_bin:$PATH" FCTL_SDK_ROOT="$sdk_root" FAKE_NAR_HASH="$expected_nar_hash" \
  "$wrapper" true

actual_wit_hash="$(shasum -a 256 "$plugin_root/wit/plugin.wit" | awk '{print $1}')"
[[ "$actual_wit_hash" == "$expected_wit_hash" ]] || fail 'vendored plugin WIT differs from the SDK lock'

# The committed snapshot is the source every credential-free gate consumes, so
# its exact content is asserted here as well as inside the wrapper.
actual_bundle_nar_hash="$(nix --extra-experimental-features nix-command hash path --type sha256 --sri "$plugin_root/$expected_bundle_path/pkg/plugin")"
[[ "$actual_bundle_nar_hash" == "$expected_bundle_nar_hash" ]] || fail 'committed fctl SDK snapshot differs from the SDK lock'
actual_bundle_wit_hash="$(shasum -a 256 "$plugin_root/$expected_bundle_path/wit/formance/fctl/plugin/v1/plugin.wit" | awk '{print $1}')"
[[ "$actual_bundle_wit_hash" == "$expected_wit_hash" ]] || fail 'committed fctl SDK snapshot WIT differs from the SDK lock'

absolute_prefix="/$('printf' Users)/$('printf' davidragot)"
if grep -rn --fixed-strings "$absolute_prefix" \
  "$plugin_root/go.mod" "$plugin_root/fctl-sdk.lock.json" "$plugin_root/Justfile" \
  "$plugin_root/README.md" "$plugin_root/docs" "$plugin_root/scripts" "$plugin_root/sdk"; then
  fail 'tracked plugin contract contains a workstation-absolute path'
fi

printf 'fctl SDK wrapper contract: ok\n'
