# fctl Payments plugin

This directory owns the portable Payments v3 command plugin. It exposes the
44 executable Payments command leaves, maps them to the 44 corresponding v3
operations, and keeps endpoint resolution, credentials, pagination control,
input files and rendering under host control.

The plugin is implemented in source and covered by a local unit contract. That
evidence does not claim an installed component, live API, OCI, native-host or
browser-host acceptance result.

The deprecated `transfer_initiation update_status` command is intentionally
absent: v3 exposes the same two transitions as the explicit `approve` and
`reject` commands.

## Layout

| Path | Role |
|---|---|
| `core/` | Immutable catalogue and generated-client host adapter. |
| `component/` | Portable descriptor assembly. |
| `entrypoints/payments/` | WIT lifecycle export. |
| `wit/plugin.wit` | Public portable lifecycle interface. |
| `scripts/build-component.sh` | Two-lane reproducible component build and 16 MiB admission gate. |
| `docs/command-inventory.md` | Auditable OpenAPI and compatibility inventory. |
| `docs/generated-client-provenance.json` | Machine-readable Payments, fctl SDK and RFC 0011 adoption evidence. |
| `audit/`, `cmd/specaudit/` | Deterministic inventory extraction and drift checks. |

JSON request files and list query bodies are read through the host input
artifact capability. Query artifacts are optional, matching the `--query`
flag, while operation input arguments remain required. The plugin never opens
local paths. Product requests are constructed and decoded by the
Payments-generated v3 client over `producthttp`, so the plugin never receives
an endpoint or credential.
Generated-client ambient authentication and retries are disabled; the host
exclusively owns both concerns. The host's `--all` control asks the plugin to
follow opaque Payments cursors within the canonical limits of 100 pages,
10,000 items and 4 MiB.

Every paginated command also declares an optional `--cursor` flag. The host
continuation contract carries a mode only, so a cursor the plugin emits in
`PageInfo.NextCursor` has no host-owned route back; without the flag a listing
larger than the `--all` ceiling is unreachable past its first page, because
`--all` fails wholesale on breach and offers no resume point. `--cursor`
resumes a single page or an `--all` sweep from the supplied position. A
Payments cursor already encodes the filters and page size of the listing that
produced it, so combining `--cursor` with any other flag on the same command is
rejected rather than silently dropped.

Commands whose emitted payload can back truthful fixed columns declare compact
table render hints; the host keeps rendering the exhaustive payload for JSON
and YAML. A column field is a dot-separated JSON path into the emitted payload:
one collection item for a listing, the product envelope for an object result.
Thirteen commands declare no hints and the reason is recorded per command in
`core/render_test.go`: nine emit `204 No Content`, two emit an array under
`data`, one a provider-keyed map, and `connectors get-config` a redacted,
provider-specific configuration union.

Connector-configuration redaction is exhaustive by construction rather than by
a hand-maintained list: every JSON property reachable from the generated
`V3ConnectorConfig` union must be either redacted or explicitly acknowledged as
carrying no credential, and a regenerated client that introduces an unreviewed
property fails the suite instead of forwarding it.

Nine admitted listing commands preserve the Payments API's JSON body on `GET`.
That contract works through the native host transport, but browser `fetch`
rejects `GET` requests with a body. Browser acceptance for those commands is
therefore explicitly blocked pending a body-free product API alternative; the
plugin never drops the body and silently broadens a filtered query.

All 44 admitted v3 operations have a generated-client mapping. Connector
install and update canonicalize the requested provider against the live
connector-config catalogue before decoding its generated union. A newly added
provider that is absent from the pinned generated client fails closed and
requires client regeneration; it is never forwarded as untyped JSON.
Credential-bearing install and update inputs are declared sensitive host
artifacts, and connector decode failures return constant diagnostics without
embedding request or response bytes.

Three bank-account operations currently have an explicitly empty scope set:
the Payments OpenAPI document protects the routes but omits their scopes. This
preserves the source contract without inventing authorization metadata; the
inventory records the upstream correction as a release gap rather than an
admission blocker.

## Verification

`fctl-sdk.lock.json` pins the SDK module, source repository, commit, NAR content
hash and canonical WIT hash without embedding a checkout path. With no
`FCTL_SDK_ROOT` set, the wrapper materialises the locked commit itself: it
fetches that exact revision — never a branch, tag or default reference — from
the locked repository into a content-addressed cache
(`$FCTL_SDK_CACHE_DIR`, default `${XDG_CACHE_HOME:-~/.cache}/formancehq/fctl-sdk`)
and reuses it on every later run. No developer checkout is required, so ordinary
repository CI runs the same gate as a workstation. The SDK repository is
private; CI obtains the credential from the `GIT_PRIVATE_TOKEN` secret, which
`formancehq/ci`'s `setup-nix` turns into a `github.com/formancehq` Git
credential. The wrapper itself reads, logs and writes no token.

For local development, `FCTL_SDK_ROOT` still overrides materialisation and
points at either the matching Git checkout or an exact content-addressed source
tree. Every Go test and component build runs through a wrapper that validates
the lock, creates an ephemeral `go.work`, and removes it on both success and
failure:

The additive optional-query descriptor requires the SDK's optional
`InputArtifactSpec` source field. The lock is sealed at fctl revision
`e9b1395f46f3100b381dbe00f5213de28e6df0e1`, which integrates it. The canonical
WIT hash is unchanged from the previous pin, so the portable lifecycle
interface is untouched by this repin.

```sh
export FCTL_SDK_ROOT=/path/to/fctl-v2-poc
just test
```

The module gate exercises the catalogue, adapter and component entry point
with the race detector, and fails when aggregate plugin statement coverage is
below 80%. The root `just tests` recipe invokes this same gate.

`core.Version` is the plugin's single build-time variable. A release build
injects the published SemVer by exporting `FCTL_PLUGIN_VERSION` before
`just build-component`; the value is validated as SemVer 2.0.0 and linked with
`-ldflags -X`. Unset means a development build, which keeps the compiled
default and produces byte-identical artifacts.

When Git metadata is present, the wrapper requires the locked commit and origin
URL, exports only the SDK and WIT from that commit, and validates the exported
projection. Dirty or ignored checkout files never enter the Go workspace. A
source tree without `.git` remains valid only when its SDK NAR hash and WIT
digest match the lock exactly.

From the repository root, inside `nix develop`:

```sh
just fctl-audit-check
just tests
just pre-commit
```

To exercise the portable build, enter the fctl authoring shell as well so
the pinned `componentize-go 0.4.1`, patched `wasi-virt 0.2.0`, `wasm-tools
1.239.0`, and `wasm-opt 124` are available. The version guard runs before the
build:

```sh
cd plugins/fctl
just build-component
```

The build performs two independent lanes, validates the final component,
requires exactly the five declared WASI clock/random/environment/poll
interfaces, compares the component, extracted WIT, import list and hashes
byte-for-byte, and rejects an artifact above 16 MiB. Build outputs remain
ignored under `dist/`.
