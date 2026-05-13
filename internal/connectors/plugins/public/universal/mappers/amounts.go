// Package mappers translates the universal-openapi.yaml wire types
// (defined in client/types.go) into the Formance PSP types (defined in
// internal/models/). One file per primitive plus shared helpers — same
// layout as ee/plugins/routable/mappers/ so contributors moving between
// connectors find a familiar shape.
//
// All mappers are pure functions: they never call the network and never
// log. Errors are returned with enough context for the calling
// FetchNext* / TranslateWebhook to wrap them with the relevant primitive
// name (e.g. "balance amount: ...").
package mappers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"
)

// ParseAmount turns the contract's decimal-string minor-unit
// representation into the *big.Int the engine expects. Empty strings map
// to nil — used for optional fields like Order.Fee where zero and absent
// have different meanings.
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

// Raw is the canonical "snapshot the wire payload onto the PSP entity"
// helper. Every PSP type carries a Raw json.RawMessage so downstream
// audits / replay can inspect exactly what the counterparty served.
func Raw(v any) (json.RawMessage, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshalling raw: %w", err)
	}
	return b, nil
}

// DefaultTime returns primary if non-zero, otherwise fallback. Used when
// a wire field is technically optional but every concrete record we
// process has at least one of the timestamps populated.
func DefaultTime(primary, fallback time.Time) time.Time {
	if !primary.IsZero() {
		return primary
	}
	return fallback
}
