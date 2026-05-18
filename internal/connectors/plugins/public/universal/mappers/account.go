package mappers

import (
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

// AccountToPSPAccount applies to both /v1/accounts and
// /v1/external-accounts — identical wire shape; only the consumer
// differs. Reference is required by the engine; CreatedAt falls back
// to "now" so downstream sort keys stay sane.
func AccountToPSPAccount(a client.Account) (models.PSPAccount, error) {
	if err := requireRef("account", a.Reference); err != nil {
		return models.PSPAccount{}, err
	}
	r, err := Raw(a)
	if err != nil {
		return models.PSPAccount{}, err
	}
	return models.PSPAccount{
		Reference:    a.Reference,
		CreatedAt:    DefaultTime(a.CreatedAt, time.Now().UTC()),
		Name:         a.Name,
		DefaultAsset: a.DefaultAsset,
		Metadata:     stampVersion(a.Metadata),
		Raw:          r,
	}, nil
}
