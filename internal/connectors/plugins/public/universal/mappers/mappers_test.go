package mappers_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

// Mappers are pure functions; table-driven tests give us cheap, focused
// coverage of every status / type / scheme branch.

func TestParseAmount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in     string
		want   *big.Int
		errors bool
	}{
		{"", nil, false},
		{"0", big.NewInt(0), false},
		{"1234", big.NewInt(1234), false},
		{"-50", big.NewInt(-50), false},
		{"abc", nil, true},
		{"1.5", nil, true},
	}
	for _, c := range cases {
		got, err := mappers.ParseAmount(c.in)
		if c.errors {
			if err == nil {
				t.Errorf("ParseAmount(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseAmount(%q): unexpected error: %v", c.in, err)
		}
		if (c.want == nil) != (got == nil) {
			t.Errorf("ParseAmount(%q): want=%v got=%v", c.in, c.want, got)
		}
		if c.want != nil && got != nil && c.want.Cmp(got) != 0 {
			t.Errorf("ParseAmount(%q): want=%s got=%s", c.in, c.want, got)
		}
	}
}

func TestPaymentStatusMappingExhaustive(t *testing.T) {
	t.Parallel()
	cases := map[string]models.PaymentStatus{
		"PENDING":          models.PAYMENT_STATUS_PENDING,
		"SUCCEEDED":        models.PAYMENT_STATUS_SUCCEEDED,
		"FAILED":           models.PAYMENT_STATUS_FAILED,
		"CANCELLED":        models.PAYMENT_STATUS_CANCELLED,
		"EXPIRED":          models.PAYMENT_STATUS_EXPIRED,
		"REFUNDED":         models.PAYMENT_STATUS_REFUNDED,
		"REFUND_REVERSED":  models.PAYMENT_STATUS_REFUND_REVERSED,
		"REFUNDED_FAILURE": models.PAYMENT_STATUS_REFUNDED_FAILURE,
		"DISPUTE":          models.PAYMENT_STATUS_DISPUTE,
		"DISPUTE_WON":      models.PAYMENT_STATUS_DISPUTE_WON,
		"DISPUTE_LOST":     models.PAYMENT_STATUS_DISPUTE_LOST,
		"AUTHORISATION":    models.PAYMENT_STATUS_AUTHORISATION,
		"CAPTURE":          models.PAYMENT_STATUS_CAPTURE,
		"CAPTURE_FAILED":   models.PAYMENT_STATUS_CAPTURE_FAILED,
		"OTHER":            models.PAYMENT_STATUS_OTHER,
		"made-up":          models.PAYMENT_STATUS_OTHER,
	}
	for in, want := range cases {
		if got := mappers.PaymentStatus(in); got != want {
			t.Errorf("PaymentStatus(%q) = %v; want %v", in, got, want)
		}
	}
}

func TestPaymentTypeMapping(t *testing.T) {
	t.Parallel()
	cases := map[string]models.PaymentType{
		"PAYIN":    models.PAYMENT_TYPE_PAYIN,
		"PAYOUT":   models.PAYMENT_TYPE_PAYOUT,
		"TRANSFER": models.PAYMENT_TYPE_TRANSFER,
		"WEIRD":    models.PAYMENT_TYPE_OTHER,
		"":         models.PAYMENT_TYPE_OTHER,
	}
	for in, want := range cases {
		if got := mappers.PaymentType(in); got != want {
			t.Errorf("PaymentType(%q) = %v; want %v", in, got, want)
		}
	}
}

func TestPaymentSchemeMapping(t *testing.T) {
	t.Parallel()
	if mappers.PaymentScheme("") != models.PAYMENT_SCHEME_OTHER {
		t.Errorf("empty scheme should map to OTHER")
	}
	if mappers.PaymentScheme("SEPA") != models.PAYMENT_SCHEME_SEPA {
		t.Errorf("SEPA scheme should map")
	}
	if mappers.PaymentScheme("xx") != models.PAYMENT_SCHEME_OTHER {
		t.Errorf("unknown scheme should fall back to OTHER")
	}
}

func TestOrderEnumMappings(t *testing.T) {
	t.Parallel()
	if mappers.OrderDirection("BUY") != models.ORDER_DIRECTION_BUY {
		t.Errorf("BUY direction must map")
	}
	if mappers.OrderDirection("?") != models.ORDER_DIRECTION_UNKNOWN {
		t.Errorf("unknown direction must fall back")
	}
	if mappers.OrderType("MARKET") != models.ORDER_TYPE_MARKET {
		t.Errorf("MARKET type must map")
	}
	if mappers.OrderStatus("FILLED") != models.ORDER_STATUS_FILLED {
		t.Errorf("FILLED status must map")
	}
	if mappers.TimeInForce("") != models.TIME_IN_FORCE_UNKNOWN {
		t.Errorf("empty TIF must map to UNKNOWN")
	}
	if mappers.TimeInForce("GTC") == models.TIME_IN_FORCE_UNKNOWN {
		t.Errorf("GTC must not map to UNKNOWN")
	}
}

func TestConversionStatusMapping(t *testing.T) {
	t.Parallel()
	if mappers.ConversionStatus("PENDING") != models.CONVERSION_STATUS_PENDING {
		t.Errorf("PENDING must map")
	}
	if mappers.ConversionStatus("xxx") != models.CONVERSION_STATUS_UNKNOWN {
		t.Errorf("unknown must fall back")
	}
}

func TestAccountToPSPAccountRoundtrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	name := "Op EUR"
	asset := "EUR/2"
	psp, err := mappers.AccountToPSPAccount(client.Account{
		Reference: "a1", CreatedAt: now, Name: &name, DefaultAsset: &asset,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if psp.Reference != "a1" || psp.CreatedAt != now || *psp.Name != "Op EUR" {
		t.Fatalf("unexpected mapping: %+v", psp)
	}
	if len(psp.Raw) == 0 {
		t.Fatal("Raw should not be empty")
	}
}

func TestOrderToPSPOrder_RejectsBadAmount(t *testing.T) {
	t.Parallel()
	_, err := mappers.OrderToPSPOrder(client.Order{
		Reference: "o1", BaseQuantityOrdered: "abc",
	})
	if err == nil {
		t.Fatal("expected ParseAmount to fail")
	}
}

func TestPaymentToPSPPaymentRespectsCreatedAtFallback(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	psp, err := mappers.PaymentToPSPPayment(client.Payment{
		Reference: "p1", UpdatedAt: now, // no CreatedAt
		Type: "PAYIN", Status: "SUCCEEDED", Amount: "100", Asset: "EUR/2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !psp.CreatedAt.Equal(now) {
		t.Fatalf("CreatedAt should fall back to UpdatedAt: got %v", psp.CreatedAt)
	}
}
