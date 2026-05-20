package mappers

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// CompanyToPSPAccount maps a Routable company onto an external PSPAccount.
// We deliberately do NOT fan out to GET /v1/companies/{id}/payment-methods
// per row; payment-method resolution is deferred to payable creation,
// which is the only place it actually matters.
func CompanyToPSPAccount(co client.Company) (models.PSPAccount, error) {
	raw, err := json.Marshal(co)
	if err != nil {
		return models.PSPAccount{}, fmt.Errorf("marshaling raw: %w", err)
	}
	displayName := co.DisplayName
	if displayName == "" {
		displayName = co.BusinessName
	}
	return models.PSPAccount{
		Reference: co.ID,
		CreatedAt: co.CreatedAt,
		Name:      pointerOrNil(displayName),
		Metadata:  CompanyMetadata(co),
		Raw:       raw,
	}, nil
}
