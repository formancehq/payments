package mappers

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
)

func TestDeliveryMethodToScheme(t *testing.T) {
	cases := map[string]models.PaymentScheme{
		// ACH family — every documented Routable variant must map.
		"ach":           models.PAYMENT_SCHEME_ACH,
		"ach_standard":  models.PAYMENT_SCHEME_ACH,
		"ach_same_day":  models.PAYMENT_SCHEME_ACH,
		"same_day_ach":  models.PAYMENT_SCHEME_ACH, // alternate spelling some Routable docs use
		"ach_expedited": models.PAYMENT_SCHEME_ACH,
		// Surrounding whitespace + uppercase tolerated.
		" ach_standard ": models.PAYMENT_SCHEME_ACH,
		"ACH":            models.PAYMENT_SCHEME_ACH,
		// Non-ACH rails — Formance enum has no dedicated WIRE/CHECK
		// constants, so all explicitly map to OTHER (not via default).
		"wire":               models.PAYMENT_SCHEME_OTHER,
		"international_wire": models.PAYMENT_SCHEME_OTHER,
		"swift":              models.PAYMENT_SCHEME_OTHER,
		"international":      models.PAYMENT_SCHEME_OTHER,
		"check":              models.PAYMENT_SCHEME_OTHER,
		"payout":             models.PAYMENT_SCHEME_OTHER,
		"wallet":             models.PAYMENT_SCHEME_OTHER,
		// Catch-all for unknown values.
		"":              models.PAYMENT_SCHEME_OTHER,
		"weird":         models.PAYMENT_SCHEME_OTHER,
		"INTERNATIONAL": models.PAYMENT_SCHEME_OTHER,
	}
	for in, want := range cases {
		if got := DeliveryMethodToScheme(in); got != want {
			t.Errorf("DeliveryMethodToScheme(%q) = %v, want %v", in, got, want)
		}
	}
}
