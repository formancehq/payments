package mappers

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

func ConversionToPSPConversion(c client.Conversion) (models.PSPConversion, error) {
	if err := requireRef("conversion", c.Reference); err != nil {
		return models.PSPConversion{}, err
	}
	src, err := ParseAmount(c.SourceAmount)
	if err != nil {
		return models.PSPConversion{}, fmt.Errorf("conversion %s sourceAmount: %w", c.Reference, err)
	}
	dst, err := ParseAmount(c.DestinationAmount)
	if err != nil {
		return models.PSPConversion{}, fmt.Errorf("conversion %s destinationAmount: %w", c.Reference, err)
	}
	fee, err := ParseAmount(c.Fee)
	if err != nil {
		return models.PSPConversion{}, fmt.Errorf("conversion %s fee: %w", c.Reference, err)
	}
	r, err := Raw(c)
	if err != nil {
		return models.PSPConversion{}, err
	}
	return models.PSPConversion{
		Reference:                   c.Reference,
		CreatedAt:                   c.CreatedAt,
		SourceAsset:                 c.SourceAsset,
		DestinationAsset:            c.DestinationAsset,
		SourceAmount:                src,
		DestinationAmount:           dst,
		Fee:                         fee,
		FeeAsset:                    c.FeeAsset,
		Status:                      ConversionStatus(c.Status),
		SourceAccountReference:      c.SourceAccountReference,
		DestinationAccountReference: c.DestinationAccountReference,
		Metadata:                    stampVersion(c.Metadata),
		Raw:                         r,
	}, nil
}

func ConversionStatus(s string) models.ConversionStatus {
	st, err := models.ConversionStatusFromString(s)
	if err != nil {
		return models.CONVERSION_STATUS_UNKNOWN
	}
	return st
}
