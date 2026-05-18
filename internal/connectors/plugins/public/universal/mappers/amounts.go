// Package mappers translates the universal-openapi.yaml wire types
// (client/types.go) into Formance PSP types (internal/models/). Pure
// functions, never call the network, never log. Errors carry enough
// context for the calling FetchNext* / TranslateWebhook to wrap.
package mappers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// ContractVersion is the universal-openapi.yaml major version this
// plugin implements. Stamped onto every PSP record's Metadata under
// MetadataPrefix so audit / replay can correlate stored records with
// the contract that produced them.
const (
	ContractVersion = "v1"
	MetadataPrefix  = "com.universal.spec/"
	metaVersionKey  = MetadataPrefix + "contract"
)

// ParseAmount turns the contract's decimal-string minor-unit into
// *big.Int. Empty input → nil (distinguishes "absent" from "zero" for
// optional fields like Order.Fee).
func ParseAmount(s string) (*big.Int, error) {
	if s == "" {
		return nil, nil
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal-integer amount %q", s)
	}
	return v, nil
}

// Raw snapshots the wire payload onto the PSP entity's Raw field so
// audit / replay can inspect what the counterparty actually served.
func Raw(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshalling raw: %w", err)
	}
	return b, nil
}

// DefaultTime returns primary if non-zero else fallback.
func DefaultTime(primary, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary
	}
	return fallback
}

// requireRef enforces the engine invariant that every PSP record has a
// non-empty Reference. `kind` is the primitive name used in the
// surfaced error so failures point straight at the bad row.
func requireRef(kind, ref string) error {
	if strings.TrimSpace(ref) == "" {
		return fmt.Errorf("%s: missing reference", kind)
	}
	return nil
}

// stampVersion returns the input metadata with the contract version
// stamped under MetadataPrefix. Always returns a non-nil map so callers
// don't need to handle the nil-vs-empty distinction.
func stampVersion(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	out[metaVersionKey] = ContractVersion
	return out
}
