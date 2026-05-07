package mappers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// ReceivableToPSPPayment converts a Routable receivable into a PSPPayment
// of PAYIN type. Mirror of PayableToPSPPayment with source/destination
// flipped.
func ReceivableToPSPPayment(r client.Receivable) (models.PSPPayment, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("marshaling raw: %w", err)
	}
	precision, err := PrecisionFor(r.CurrencyCode)
	if err != nil {
		return models.PSPPayment{}, err
	}
	amount, err := ToMinorUnits(r.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("parsing amount: %w", err)
	}

	payment := models.PSPPayment{
		Reference: r.ID,
		CreatedAt: r.CreatedAt,
		Type:      models.PAYMENT_TYPE_PAYIN,
		Amount:    amount,
		Asset:     FormatAsset(r.CurrencyCode),
		Scheme:    DeliveryMethodToScheme(r.DeliveryMethod),
		Status:    PayableStatus(r.Status),
		Metadata:  ReceivableMetadata(r),
		Raw:       raw,
	}
	if r.PayFromCompany != nil && r.PayFromCompany.ID != "" {
		ref := r.PayFromCompany.ID
		payment.SourceAccountReference = &ref
	}
	if r.DepositToAccount != nil && r.DepositToAccount.ID != "" {
		ref := r.DepositToAccount.ID
		payment.DestinationAccountReference = &ref
	}
	return payment, nil
}

// ReceivablesToPSPPayments mirrors PayablesToPSPPayments for receivables.
func ReceivablesToPSPPayments(in []client.Receivable, watermark time.Time, skip func(id string, err error)) ([]models.PSPPayment, time.Time) {
	out := make([]models.PSPPayment, 0, len(in))
	for _, r := range in {
		payment, err := ReceivableToPSPPayment(r)
		if err != nil {
			if skip != nil {
				skip(r.ID, err)
			}
			continue
		}
		out = append(out, payment)
		watermark = LaterOf(watermark, StatusChangedAtOrCreated(r.StatusChangedAt, r.CreatedAt))
	}
	return out, watermark
}
