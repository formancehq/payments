package universal

import (
	"slices"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
)

// capabilitySet is a small immutable lookup of the capabilities a specific
// counterparty has declared at install time. It is built once during Install
// and read concurrently by every fetch/create/webhook call thereafter.
type capabilitySet map[models.Capability]struct{}

func newCapabilitySet(declared []models.Capability) capabilitySet {
	s := make(capabilitySet, len(declared))
	for _, c := range declared {
		s[c] = struct{}{}
	}
	return s
}

func (s capabilitySet) has(c models.Capability) bool {
	_, ok := s[c]
	return ok
}

// require returns plugins.ErrNotImplemented when the counterparty did not
// declare the requested capability at install time. Engine activities map
// ErrNotImplemented to a non-retryable terminal failure, which is the
// behaviour we want here.
func (s capabilitySet) require(c models.Capability) error {
	if !s.has(c) {
		return plugins.ErrNotImplemented
	}
	return nil
}

// applyOverrides narrows the declared set with an installer-supplied
// allow-list. Entries that the counterparty did not declare are silently
// dropped (caller already validated the strings via validator's `oneof`).
func applyOverrides(declared []models.Capability, overrides []string) []models.Capability {
	if len(overrides) == 0 {
		return declared
	}
	allow := make(map[string]struct{}, len(overrides))
	for _, o := range overrides {
		allow[o] = struct{}{}
	}
	out := declared[:0:0]
	for _, c := range declared {
		if _, ok := allow[c.String()]; ok {
			out = append(out, c)
		}
	}
	return out
}

// dedupCapabilities returns a deterministic, dedup'd slice. The order matters
// for the install-time workflow tree (we follow the constant order from
// internal/models/capabilities.go).
func dedupCapabilities(in []models.Capability) []models.Capability {
	seen := make(map[models.Capability]struct{}, len(in))
	out := make([]models.Capability, 0, len(in))
	for _, c := range in {
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b models.Capability) int {
		return int(a) - int(b)
	})
	return out
}
