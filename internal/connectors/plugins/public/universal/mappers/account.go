package mappers

import (
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

// AccountToPSPAccount applies to both internal accounts (GET /v1/accounts)
// and external accounts (GET /v1/external-accounts) — the wire shape is
// identical; only the engine consumer differs.
func AccountToPSPAccount(a client.Account) (models.PSPAccount, error) {
	r, err := Raw(a)
	if err != nil {
		return models.PSPAccount{}, err
	}
	return models.PSPAccount{
		Reference:    a.Reference,
		CreatedAt:    a.CreatedAt,
		Name:         a.Name,
		DefaultAsset: a.DefaultAsset,
		Metadata:     a.Metadata,
		Raw:          r,
	}, nil
}
