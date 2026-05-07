package mappers

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// SettingsAccountToPSPAccount maps a Routable settings account onto a
// Formance internal PSPAccount. Routable carries the currency on
// type_details; we surface it as DefaultAsset only when we recognize the
// code.
func SettingsAccountToPSPAccount(a client.Account) (models.PSPAccount, error) {
	raw, err := json.Marshal(a)
	if err != nil {
		return models.PSPAccount{}, fmt.Errorf("marshaling raw: %w", err)
	}
	out := models.PSPAccount{
		Reference: a.ID,
		CreatedAt: a.CreatedAt,
		Name:      pointerOrNil(a.Name),
		Metadata:  SettingsAccountMetadata(a),
		Raw:       raw,
	}
	if asset := FormatAsset(a.CurrencyCode); asset != "" {
		out.DefaultAsset = &asset
	}
	return out, nil
}

func pointerOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
