package mappers

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

// PaymentToPSPPayment translates a wire Payment into the engine's
// PSPPayment. Unknown enum values intentionally map to the "_OTHER"
// constants — the engine treats those as "we ingested it but don't know
// what kind it is", which is strictly safer than failing the whole batch
// for a single new vendor-specific status.
func PaymentToPSPPayment(p client.Payment) (models.PSPPayment, error) {
	amount, err := ParseAmount(p.Amount)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("payment amount: %w", err)
	}
	r, err := Raw(p)
	if err != nil {
		return models.PSPPayment{}, err
	}
	return models.PSPPayment{
		ParentReference:             p.ParentReference,
		Reference:                   p.Reference,
		CreatedAt:                   DefaultTime(p.CreatedAt, p.UpdatedAt),
		Type:                        PaymentType(p.Type),
		Amount:                      amount,
		Asset:                       p.Asset,
		Scheme:                      PaymentScheme(p.Scheme),
		Status:                      PaymentStatus(p.Status),
		SourceAccountReference:      p.SourceAccountReference,
		DestinationAccountReference: p.DestinationAccountReference,
		Metadata:                    p.Metadata,
		Raw:                         r,
	}, nil
}

// PaymentStatus uses the canonical models.PaymentStatus.Scan codepath so
// every status string the engine knows about is mapped automatically.
// Anything else degrades to PAYMENT_STATUS_OTHER — see the Routable
// mapper for the same rationale.
func PaymentStatus(s string) models.PaymentStatus {
	var status models.PaymentStatus
	if err := status.Scan(s); err != nil {
		return models.PAYMENT_STATUS_OTHER
	}
	return status
}

func PaymentType(t string) models.PaymentType {
	switch t {
	case "PAYIN":
		return models.PAYMENT_TYPE_PAYIN
	case "PAYOUT":
		return models.PAYMENT_TYPE_PAYOUT
	case "TRANSFER":
		return models.PAYMENT_TYPE_TRANSFER
	default:
		return models.PAYMENT_TYPE_OTHER
	}
}

func PaymentScheme(s string) models.PaymentScheme {
	if s == "" {
		return models.PAYMENT_SCHEME_OTHER
	}
	var scheme models.PaymentScheme
	if err := scheme.Scan(s); err != nil {
		return models.PAYMENT_SCHEME_OTHER
	}
	return scheme
}
