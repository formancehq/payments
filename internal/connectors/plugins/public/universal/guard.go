package universal

import (
	"fmt"
	"slices"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
)

// capabilitySet is the per-install lookup built once during Install and
// read concurrently by every primitive thereafter.
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
// declare the capability. The engine maps this to a non-retryable terminal
// failure on the activity.
func (s capabilitySet) require(c models.Capability) error {
	if !s.has(c) {
		return plugins.ErrNotImplemented
	}
	return nil
}

// applyOverrides narrows the declared set with an installer-supplied
// allow-list. Each override MUST be one of the capabilities the
// counterparty actually declared; otherwise we surface the operator
// mistake instead of silently masking it.
func applyOverrides(declared []models.Capability, overrides []string) ([]models.Capability, error) {
	if len(overrides) == 0 {
		return declared, nil
	}
	declaredByName := make(map[string]models.Capability, len(declared))
	for _, c := range declared {
		declaredByName[c.String()] = c
	}
	out := make([]models.Capability, 0, len(overrides))
	for _, o := range overrides {
		c, ok := declaredByName[o]
		if !ok {
			return nil, fmt.Errorf("capability override %q not declared by counterparty", o)
		}
		out = append(out, c)
	}
	return out, nil
}

// dedupCapabilities returns a deterministic, dedup'd slice ordered by the
// canonical models.Capability constant — the workflow tree builder relies
// on that order.
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
