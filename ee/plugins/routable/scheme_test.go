package routable

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
)

func TestDeliveryMethodToScheme(t *testing.T) {
	cases := map[string]models.PaymentScheme{
		"ach":             models.PAYMENT_SCHEME_ACH,
		"ach_standard":    models.PAYMENT_SCHEME_ACH,
		"ach_same_day":    models.PAYMENT_SCHEME_ACH,
		"ach_expedited":   models.PAYMENT_SCHEME_ACH,
		" ach_standard ": models.PAYMENT_SCHEME_ACH, // surrounding whitespace tolerated
		"wire":            models.PAYMENT_SCHEME_OTHER,
		"check":           models.PAYMENT_SCHEME_OTHER,
		"":                models.PAYMENT_SCHEME_OTHER,
		"weird":           models.PAYMENT_SCHEME_OTHER,
		"INTERNATIONAL":   models.PAYMENT_SCHEME_OTHER,
	}
	for in, want := range cases {
		if got := deliveryMethodToScheme(in); got != want {
			t.Errorf("deliveryMethodToScheme(%q) = %v, want %v", in, got, want)
		}
	}
}
