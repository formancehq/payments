//go:build contract

// Package contracttest holds helpers shared by the connector contract tests in
// internal/connectors/plugins/public/<connector>/client/contract_test.go
// (see docs/contract-tests-connector-prompt.md).
//
// Everything here is gated behind the `contract` build tag, so it is compiled
// only for contract runs (`go test -tags contract ...`) and never in production
// builds or the `-tags it` suite. The package lives OUTSIDE plugins/public/
// on purpose: the plugin compilers (tools/compile-plugins, tools/compile-configs)
// treat every directory under public/ as a connector, so a helper package there
// would break them.
package contracttest

import (
	"fmt"
	"os"
	"time"
)

// BootstrapEnabled reports whether the schema specs should print paste-ready
// pinned-ID literals to stderr. It is true when either the connector-specific
// <CONNECTOR_UPPER>_CONTRACT_BOOTSTRAP or the shared CONTRACT_BOOTSTRAP env var
// is set. Off by default so a green run stays quiet once the pins are filled; on
// demand it (re)prints the pins after the sandbox is reseeded.
func BootstrapEnabled(connectorUpper string) bool {
	return os.Getenv(connectorUpper+"_CONTRACT_BOOTSTRAP") != "" || os.Getenv("CONTRACT_BOOTSTRAP") != ""
}

// LogBootstrap prints a paste-ready Go slice literal of the given IDs to stderr,
// so a maintainer can copy it into the expected*IDs pins in a connector's
// contract_test.go. `go test` discards a passing package's stdout/stderr unless
// `-v` is set, so the `contract-tests` Justfile target runs with `-v` for this
// output to surface.
func LogBootstrap(name string, ids []string) {
	fmt.Fprintf(os.Stderr, "\n// BOOTSTRAP — paste into contract_test.go to pin ordering:\nvar %s = []string{\n", name)
	for _, id := range ids {
		fmt.Fprintf(os.Stderr, "\t%q,\n", id)
	}
	fmt.Fprintf(os.Stderr, "}\n")
}

// FilterToPinned keeps only the ids present in pinned, preserving their order in
// the source slice. Used by the relative-order (subsequence) ordering contract to
// assert that the known records retain their relative order while ignoring any
// newly created ones.
func FilterToPinned(ids, pinned []string) []string {
	set := make(map[string]struct{}, len(pinned))
	for _, id := range pinned {
		set[id] = struct{}{}
	}
	out := make([]string, 0, len(pinned))
	for _, id := range ids {
		if _, ok := set[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

// Ref returns a per-run-unique idempotency reference of the form
// "<connector>-contract-<prefix>-<nanos>", so money-movement specs always create
// a fresh record and never collide with a prior run's idempotency key.
func Ref(connector, prefix string) string {
	return fmt.Sprintf("%s-contract-%s-%d", connector, prefix, time.Now().UnixNano())
}
