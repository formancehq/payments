package v2

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestGetQueryBuilder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	ctx, span := tracer.Start(ctx, "test")
	defer span.End()

	t.Run("with body", func(t *testing.T) {
		t.Parallel()

		body := `{"$match": {"foo": "bar"}}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))

		qb, err := getQueryBuilder(span, req)
		require.NoError(t, err)
		require.NotNil(t, qb)
	})

	t.Run("with query parameter", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test?query={\"$match\":{\"foo\":\"bar\"}}", nil)

		qb, err := getQueryBuilder(span, req)
		require.NoError(t, err)
		require.NotNil(t, qb)
	})

	t.Run("with empty body and no query parameter", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		qb, err := getQueryBuilder(span, req)
		require.NoError(t, err)
		require.NotNil(t, qb)
	})

	t.Run("with body read error", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodPost, "/test", &errorReader{})

		_, err := getQueryBuilder(span, req)
		require.Error(t, err)
	})
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestGetPagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	ctx, span := tracer.Start(ctx, "test")
	defer span.End()

	t.Run("with valid query", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test?pageSize=10", nil)

		options, err := getPagination(span, req, struct{}{})
		require.NoError(t, err)
		require.NotNil(t, options)
		require.Equal(t, 10, options.PageSize)
	})

	t.Run("with invalid page size", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/test?pageSize=invalid", nil)

		_, err := getPagination(span, req, struct{}{})
		require.Error(t, err)
	})
}

func TestToV2Provider(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		provider string
		expected string
	}{
		{"adyen", "adyen", connectorAdyen},
		{"atlar", "atlar", connectorAtlar},
		{"bankingcircle", "bankingcircle", connectorBankingCircle},
		{"currencycloud", "currencycloud", connectorCurrencyCloud},
		{"dummypay", "dummypay", connectorDummyPay},
		{"generic", "generic", connectorGeneric},
		{"mangopay", "mangopay", connectorMangopay},
		{"modulr", "modulr", connectorModulr},
		{"moneycorp", "moneycorp", connectorMoneycorp},
		{"stripe", "stripe", connectorStripe},
		{"wise", "wise", connectorWise},
		{"unknown", "unknown", "unknown"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := toV2Provider(tc.provider)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestToV3Provider(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		provider string
		expected string
	}{
		{"ADYEN", connectorAdyen, "adyen"},
		{"ATLAR", connectorAtlar, "atlar"},
		{"BANKING-CIRCLE", connectorBankingCircle, "bankingcircle"},
		{"CURRENCY-CLOUD", connectorCurrencyCloud, "currencycloud"},
		{"DUMMY-PAY", connectorDummyPay, "dummypay"},
		{"GENERIC", connectorGeneric, "generic"},
		{"MANGOPAY", connectorMangopay, "mangopay"},
		{"MODULR", connectorModulr, "modulr"},
		{"MONEYCORP", connectorMoneycorp, "moneycorp"},
		{"STRIPE", connectorStripe, "stripe"},
		{"WISE", connectorWise, "wise"},
		{"unknown", "unknown", "unknown"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := toV3Provider(tc.provider)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestToV2PaymentScheme(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		scheme   models.PaymentScheme
		expected string
	}{
		{"unknown", models.PAYMENT_SCHEME_UNKNOWN, paymentSchemeUnknown},
		{"other", models.PAYMENT_SCHEME_OTHER, paymentSchemeOther},
		{"visa", models.PAYMENT_SCHEME_CARD_VISA, paymentSchemeVisa},
		{"mastercard", models.PAYMENT_SCHEME_CARD_MASTERCARD, paymentSchemeMastercard},
		{"amex", models.PAYMENT_SCHEME_CARD_AMEX, paymentSchemeAmex},
		{"diners", models.PAYMENT_SCHEME_CARD_DINERS, paymentSchemeDiners},
		{"discover", models.PAYMENT_SCHEME_CARD_DISCOVER, paymentSchemeDiscover},
		{"jcb", models.PAYMENT_SCHEME_CARD_JCB, paymentSchemeJcb},
		{"unionpay", models.PAYMENT_SCHEME_CARD_UNION_PAY, paymentSchemeUnionpay},
		{"alipay", models.PAYMENT_SCHEME_CARD_ALIPAY, paymentSchemeAlipay},
		{"cup", models.PAYMENT_SCHEME_CARD_CUP, paymentSchemeCup},
		{"sepa debit", models.PAYMENT_SCHEME_SEPA_DEBIT, paymentSchemeSepaDebit},
		{"sepa credit", models.PAYMENT_SCHEME_SEPA_CREDIT, paymentSchemeSepaCredit},
		{"sepa", models.PAYMENT_SCHEME_SEPA, paymentSchemeSepa},
		{"apple pay", models.PAYMENT_SCHEME_APPLE_PAY, paymentSchemeApplePay},
		{"google pay", models.PAYMENT_SCHEME_GOOGLE_PAY, paymentSchemeGooglePay},
		{"doku", models.PAYMENT_SCHEME_DOKU, paymentSchemeDoku},
		{"dragonpay", models.PAYMENT_SCHEME_DRAGON_PAY, paymentSchemeDragonpay},
		{"maestro", models.PAYMENT_SCHEME_MAESTRO, paymentSchemeMaestro},
		{"molpay", models.PAYMENT_SCHEME_MOL_PAY, paymentSchemeMolpay},
		{"a2a", models.PAYMENT_SCHEME_A2A, paymentSchemeA2a},
		{"ach debit", models.PAYMENT_SCHEME_ACH_DEBIT, paymentSchemeAchDebit},
		{"ach", models.PAYMENT_SCHEME_ACH, paymentSchemeAch},
		{"rtp", models.PAYMENT_SCHEME_RTP, paymentSchemeRtp},
		{"invalid", models.PaymentScheme("invalid"), paymentSchemeUnknown},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := toV2PaymentScheme(tc.scheme)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestToV3PaymentScheme(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		scheme   string
		expected string
	}{
		{"unknown", "unknown", models.PAYMENT_SCHEME_UNKNOWN.String()},
		{"other", "other", models.PAYMENT_SCHEME_OTHER.String()},
		{"visa", "visa", models.PAYMENT_SCHEME_CARD_VISA.String()},
		{"mastercard", "mastercard", models.PAYMENT_SCHEME_CARD_MASTERCARD.String()},
		{"amex", "amex", models.PAYMENT_SCHEME_CARD_AMEX.String()},
		{"diners", "diners", models.PAYMENT_SCHEME_CARD_DINERS.String()},
		{"discover", "discover", models.PAYMENT_SCHEME_CARD_DISCOVER.String()},
		{"jcb", "jcb", models.PAYMENT_SCHEME_CARD_JCB.String()},
		{"unionpay", "unionpay", models.PAYMENT_SCHEME_CARD_UNION_PAY.String()},
		{"alipay", "alipay", models.PAYMENT_SCHEME_CARD_ALIPAY.String()},
		{"cup", "cup", models.PAYMENT_SCHEME_CARD_CUP.String()},
		{"sepa debit", "sepa debit", models.PAYMENT_SCHEME_SEPA_DEBIT.String()},
		{"sepa credit", "sepa credit", models.PAYMENT_SCHEME_SEPA_CREDIT.String()},
		{"sepa", "sepa", models.PAYMENT_SCHEME_SEPA.String()},
		{"apple pay", "apple pay", models.PAYMENT_SCHEME_APPLE_PAY.String()},
		{"google pay", "google pay", models.PAYMENT_SCHEME_GOOGLE_PAY.String()},
		{"doku", "doku", models.PAYMENT_SCHEME_DOKU.String()},
		{"dragonpay", "dragonpay", models.PAYMENT_SCHEME_DRAGON_PAY.String()},
		{"maestro", "maestro", models.PAYMENT_SCHEME_MAESTRO.String()},
		{"molpay", "molpay", models.PAYMENT_SCHEME_MOL_PAY.String()},
		{"a2a", "a2a", models.PAYMENT_SCHEME_A2A.String()},
		{"ach debit", "ach debit", models.PAYMENT_SCHEME_ACH_DEBIT.String()},
		{"ach", "ach", models.PAYMENT_SCHEME_ACH.String()},
		{"rtp", "rtp", models.PAYMENT_SCHEME_RTP.String()},
		{"invalid", "invalid", "invalid"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := toV3PaymentScheme(tc.scheme)
			require.Equal(t, tc.expected, result)
		})
	}
}
