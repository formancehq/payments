# fctl Payments plugin — preparation tranche

This directory is the Payments-owned preparation for the fctl Payments plugin
(fctl-v2 programme Task 8). It currently contains **no plugin**: no runtime, no
component entry point, no HTTP client, no generated bindings, no ABI. It
contains the one thing that can be delivered and defended before the fctl-v2
plugin runtime is frozen — a reproducible, versioned inventory of the Payments
operation surface and its mapping onto the legacy fctl command baseline.

## Why the inventory comes first

The fctl-v2 programme gates every product plugin implementation behind its MVP4
contract freeze (portable component lifecycle, host-owned access and
capabilities, exact per-operation authorisation scopes). Writing a catalogue or
an adapter against an unfrozen ABI would produce work that has to be thrown
away. Establishing *which operations exist, which ones the legacy CLI covered,
what each one requires, and what is genuinely blocked* does not depend on that
freeze, and it is the input the later implementation needs.

## Contents

| Path | Role |
|---|---|
| `docs/command-inventory.md` | Source of truth: method, reasoning, evidence, risks, blockers, gates. Hand-written. |
| `docs/v3-operations.generated.md` | Generated tables: totals, per-family operations, baseline mapping, deprecations. |
| `audit/spec.go` | Reads the merged `openapi.yaml` into typed operations. No inference. |
| `audit/baseline.go` | The pinned legacy fctl `payments` command baseline and its /v3 mapping. |
| `audit/classify.go` | Frozen family table and per-operation risk derivation. |
| `audit/blockers.go` | Recorded admission blockers and spec-versus-server divergences. |
| `audit/report.go` | Assembles the report and derives every quoted count. |
| `audit/testdata/report.json` | Golden report; the determinism gate. |
| `cmd/specaudit` | Regenerates the two committed artefacts. |

## Commands

Run from the repository root inside `nix develop`:

```sh
just fctl-audit         # regenerate the committed inventory artefacts
just fctl-audit-check   # fail if they no longer match openapi.yaml
```

Or from this directory:

```sh
go test ./...
```

`just tests`, `just lint`, `just tidy` and `just pre-commit` at the repository
root include this module.

## What must not be inferred from this directory

- No operation is accepted into a plugin catalogue. The inventory records
  proven facts and separately records blockers.
- No authorisation scope is invented. Three operations carry no declared scopes
  and are blocked for exactly that reason.
- No transport, component build, OCI installation or dual-host behaviour is
  claimed, prepared or gated here.

The remaining gates are listed at the end of `docs/command-inventory.md`.
