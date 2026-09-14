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
artifact capability. The plugin never opens local paths. Product requests are
constructed and decoded by the Payments-generated v3 client over
`producthttp`, so the plugin never receives an endpoint or credential.
Generated-client ambient authentication and retries are disabled; the host
exclusively owns both concerns. The host's `--all` control asks the plugin to
follow opaque Payments cursors within the canonical limits of 100 pages,
10,000 items and 4 MiB.

All 44 admitted v3 operations have a generated-client mapping. Connector
install and update canonicalize the requested provider against the live
connector-config catalogue before decoding its generated union. A newly added
provider that is absent from the pinned generated client fails closed and
requires client regeneration; it is never forwarded as untyped JSON.

Three bank-account operations currently have an explicitly empty scope set:
the Payments OpenAPI document protects the routes but omits their scopes. This
preserves the source contract without inventing authorization metadata; the
inventory records the upstream correction as a release gap rather than an
admission blocker.

## Verification

`fctl-sdk.lock.json` pins the SDK module, source repository, commit, NAR content
hash and canonical WIT hash without embedding a checkout path. Point
`FCTL_SDK_ROOT` at either the matching Git checkout or an exact
content-addressed source tree. Every Go test and component build runs through a
wrapper that validates the lock, creates an ephemeral `go.work`, and removes it
on both success and failure:

```sh
export FCTL_SDK_ROOT=/path/to/fctl-v2-poc
just test
```

The module gate exercises the catalogue, adapter and component entry point
with the race detector, and fails when aggregate plugin statement coverage is
below 80%. The root `just tests` recipe invokes this same gate.

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
