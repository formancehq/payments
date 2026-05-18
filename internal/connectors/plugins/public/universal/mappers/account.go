package mappers

import (
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

// MetadataPrefix is the namespace for connector-injected metadata. We
// only stamp the contract source so audit / replay can distinguish
// counterparty-supplied metadata from plugin-derived enrichment.
const MetadataPrefix = "com.universal.spec/"

// AccountToPSPAccount applies to both internal accounts (GET /v1/accounts)
// and external accounts (GET /v1/external-accounts) — identical wire
// shape, only the engine consumer differs.
//
// Engine invariants enforced here: non-empty Reference, non-zero
// CreatedAt (falls back to "now" so downstream sort keys stay sane).
// Per-connector enrichment lives in the MetadataPrefix namespace.
func AccountToPSPAccount(a client.Account) (models.PSPAccount, error) {
	if a.Reference == "" {
		return models.PSPAccount{}, fmt.Errorf("account: %w", errors.New("missing reference"))
	}
	r, err := Raw(a)
	if err != nil {
		return models.PSPAccount{}, err
	}
	meta := map[string]string{MetadataPrefix + "source": "universal"}
	for k, v := range a.Metadata {
		meta[k] = v
	}
	return models.PSPAccount{
		Reference:    a.Reference,
		CreatedAt:    DefaultTime(a.CreatedAt, time.Now().UTC()),
		Name:         a.Name,
		DefaultAsset: a.DefaultAsset,
		Metadata:     meta,
		Raw:          r,
	}, nil
}
