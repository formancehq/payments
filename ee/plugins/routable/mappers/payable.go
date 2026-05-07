package mappers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// PayableToPSPPayment converts a Routable payable into a PSPPayment of
// PAYOUT type. Returns an error so the caller decides whether to skip
// the row (e.g. unsupported currency) or surface it.
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
		CreatedAt: pa.CreatedAt,
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

// PayablesToPSPPayments runs PayableToPSPPayment over each input row and
// tracks the latest status_changed_at observed (used by the cycle cursor).
// Skips invoke skip(id, err) so callers can log + count rather than abort
// the whole page on one bad row.
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

// LaterOf returns whichever of a or b is later (or zero when both are zero).
func LaterOf(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() || a.After(b) {
		return a
	}
	return b
}

// StatusChangedAtOrCreated picks status_changed_at when set, otherwise
// the created_at — Routable can return a nil status_changed_at on draft rows.
func StatusChangedAtOrCreated(statusChangedAt *time.Time, createdAt time.Time) time.Time {
	if statusChangedAt != nil && !statusChangedAt.IsZero() {
		return *statusChangedAt
	}
	return createdAt
}
