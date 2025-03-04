package gocardless

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"

	"github.com/formancehq/payments/internal/models"
)

func ParseGocardlessTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func externalAccountFromGocardlessData(data client.GocardlessGenericAccount) (models.PSPAccount, error) {

	raw, err := json.Marshal(data)

	if err != nil {
		return models.PSPAccount{}, err
	}

	parsedCreatedAt := time.Unix(data.CreatedAt, 0)

	defaultAsset := currency.FormatAsset(SupportedCurrenciesWithDecimal, data.Currency)

	metadata := extractExternalAccountMetadata(data.Metadata)
	metadata[client.GocardlessAccountTypeMetadataKey] = data.AccountType

	return models.PSPAccount{
		Reference:    data.ID,
		CreatedAt:    parsedCreatedAt,
		Name:         &data.AccountHolderName,
		Metadata:     metadata,
		DefaultAsset: &defaultAsset,
		Raw:          raw,
	}, nil
}

func extractExternalAccountMetadata(externalMetadata map[string]interface{}) map[string]string {

	metadata := make(map[string]string)
	for k, v := range externalMetadata {
		metadata[k] = fmt.Sprintf("%v", v)
	}

	return metadata

}
