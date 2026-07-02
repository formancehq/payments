//go:build contract

// Package contracttest holds helpers shared by the connector contract tests in
// {ce,ee}/plugins/<connector>/client/contract_test.go
// (see docs/contract-tests-connector-prompt.md).
//
// Everything here is gated behind the `contract` build tag, so it is compiled
// only for contract runs (`go test -tags contract ...`) and never in production
// builds or the `-tags it` suite. The package lives in pkg/domain on purpose:
// CE plugins are separate Go modules that cannot import the root module's
// internal/ tree, but every plugin already depends on pkg/domain. Until a
// pkg/domain release containing this package is tagged, each CE plugin's
// go.mod carries a temporary local replace of pkg/domain.
package contracttest

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/onsi/gomega"
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
//
// NOTE: this form is ~40+ chars. If the PSP caps the idempotency-key length (e.g.
// Mangopay requires 16–36 chars) or requires a UUID, use UUIDRef instead.
func Ref(connector, prefix string) string {
	return fmt.Sprintf("%s-contract-%s-%d", connector, prefix, time.Now().UnixNano())
}

// UUIDRef returns a random UUIDv4 string — a universally-safe per-run idempotency
// key: 36 chars, alphanumeric + dashes, high entropy, never colliding. Prefer it
// over Ref for a PSP that caps the idempotency-key length (e.g. Mangopay's 16–36,
// which rejects Ref's longer form) or that requires the reference to be a UUID.
func UUIDRef() string {
	return uuid.NewString()
}

// AssertIntegerAmount asserts n (a PSP money amount carried as a json.Number)
// parses as an integer in minor units, mirroring the connectors that parse
// balances/amounts via big.Int.SetString(s, 10) and error on a non-integer. label
// names the field in the failure message.
func AssertIntegerAmount(n json.Number, label string) {
	var amt big.Int
	_, ok := amt.SetString(n.String(), 10)
	gomega.Expect(ok).To(gomega.BeTrue(), "%s amount %q is not an integer", label, n.String())
}

// AssertDecimalAmount asserts n (a PSP money amount carried as a json.Number)
// parses as a decimal, mirroring the connectors whose amounts are major-unit
// decimal strings (e.g. CurrencyCloud). label names the field in the failure
// message.
func AssertDecimalAmount(n json.Number, label string) {
	_, err := n.Float64()
	gomega.Expect(err).To(gomega.BeNil(), "%s amount %q is not numeric", label, n.String())
}

// AssertNonDecreasing asserts values are in non-decreasing order — the ordering
// contract for a list the connector consumes in a sort order it relies on (e.g. a
// CreationDate:ASC watermark walk). It is a maintenance-free alternative to
// pinned-ID subsequences when the real dependency is "the API honors its sort
// key". label names the list in the failure message.
func AssertNonDecreasing[T cmp.Ordered](values []T, label string) {
	for i := 1; i < len(values); i++ {
		gomega.Expect(values[i] >= values[i-1]).To(gomega.BeTrue(),
			"%s is not sorted ascending at index %d: %v < %v", label, i, values[i], values[i-1])
	}
}
