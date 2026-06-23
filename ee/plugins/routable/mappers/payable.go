package mappers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

func PayableToPSPPayment(pa client.Payable) (models.PSPPayment, error) {
	raw, err := json.Marshal(pa)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("marshaling raw: %w", err)
	}
	precision, err := PrecisionFor(pa.CurrencyCode)
	if err != nil {
		return models.PSPPayment{}, err
	}
	amount, err := ToMinorUnits(pa.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("parsing amount: %w", err)
	}

	payment := models.PSPPayment{
		Reference: pa.ID,
		// status_changed_at drives engine adjustment timestamps.
		CreatedAt: StatusChangedAtOrCreated(pa.StatusChangedAt, pa.CreatedAt),
		Type:      models.PAYMENT_TYPE_PAYOUT,
		Amount:    amount,
		Asset:     FormatAsset(pa.CurrencyCode),
		Scheme:    DeliveryMethodToScheme(pa.DeliveryMethod),
		Status:    PayableStatus(pa.Status),
		Metadata:  PayableMetadata(pa),
		Raw:       raw,
	}
	if pa.WithdrawFromAccount != nil && pa.WithdrawFromAccount.ID != "" {
		ref := pa.WithdrawFromAccount.ID
		payment.SourceAccountReference = &ref
	}
	if pa.PayToCompany != nil && pa.PayToCompany.ID != "" {
		ref := pa.PayToCompany.ID
		payment.DestinationAccountReference = &ref
	}
	return payment, nil
}

// PayablesToPSPPayments tracks the latest status_changed_at observed for
// the cycle cursor and routes per-row mapping errors to skip(id, err) so
// callers can log + count without aborting the whole page.
func PayablesToPSPPayments(in []client.Payable, watermark time.Time, skip func(id string, err error)) ([]models.PSPPayment, time.Time) {
	out := make([]models.PSPPayment, 0, len(in))
	for _, pa := range in {
		payment, err := PayableToPSPPayment(pa)
		if err != nil {
			if skip != nil {
				skip(pa.ID, err)
			}
			continue
		}
		out = append(out, payment)
		watermark = LaterOf(watermark, StatusChangedAtOrCreated(pa.StatusChangedAt, pa.CreatedAt))
	}
	return out, watermark
}

func LaterOf(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() || a.After(b) {
		return a
	}
	return b
}

// StatusChangedAtOrCreated falls back to created_at because Routable
// returns a nil status_changed_at on draft rows.
func StatusChangedAtOrCreated(statusChangedAt *time.Time, createdAt time.Time) time.Time {
	if statusChangedAt != nil && !statusChangedAt.IsZero() {
		return *statusChangedAt
	}
	return createdAt
}
