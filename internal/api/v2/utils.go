package v2

import (
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func getQueryBuilder(span trace.Span, r *http.Request) (query.Builder, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		span.SetAttributes(attribute.String("query", string(data)))
		return query.ParseJSON(string(data))
	} else {
		// In order to be backward compatible
		span.SetAttributes(attribute.String("query", r.URL.Query().Get("query")))
		return query.ParseJSON(r.URL.Query().Get("query"))
	}
}

func getPagination[T any](span trace.Span, r *http.Request, options T) (*bunpaginate.PaginatedQueryOptions[T], error) {
	qb, err := getQueryBuilder(span, r)
	if err != nil {
		return nil, err
	}

	pageSize, err := bunpaginate.GetPageSize(r)
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.Int64("pageSize", int64(pageSize)))

	return pointer.For(bunpaginate.NewPaginatedQueryOptions(options).WithQueryBuilder(qb).WithPageSize(pageSize)), nil
}

const (
	connectorStripe        string = "STRIPE"
	connectorDummyPay      string = "DUMMY-PAY"
	connectorWise          string = "WISE"
	connectorModulr        string = "MODULR"
	connectorCurrencyCloud string = "CURRENCY-CLOUD"
	connectorBankingCircle string = "BANKING-CIRCLE"
	connectorMangopay      string = "MANGOPAY"
	connectorMoneycorp     string = "MONEYCORP"
	connectorAtlar         string = "ATLAR"
	connectorAdyen         string = "ADYEN"
	connectorGeneric       string = "GENERIC"
	connectorBinance       string = "BINANCE"
	connectorBitstamp      string = "BITSTAMP"
	connectorCoinbaseprime string = "COINBASEPRIME"
	connectorColumn        string = "COLUMN"
	connectorFireblocks    string = "FIREBLOCKS"
	connectorIncrease      string = "INCREASE"
	connectorKraken        string = "KRAKEN"
	connectorPlaid         string = "PLAID"
	connectorPowens        string = "POWENS"
	connectorQonto         string = "QONTO"
	connectorTink          string = "TINK"
)

func toV2Provider(provider string) string {
	switch provider {
	case "adyen":
		return connectorAdyen
	case "atlar":
		return connectorAtlar
	case "bankingcircle":
		return connectorBankingCircle
	case "currencycloud":
		return connectorCurrencyCloud
	case "dummypay":
		return connectorDummyPay
	case "generic":
		return connectorGeneric
	case "mangopay":
		return connectorMangopay
	case "modulr":
		return connectorModulr
	case "moneycorp":
		return connectorMoneycorp
	case "stripe":
		return connectorStripe
	case "wise":
		return connectorWise
	case "binance":
		return connectorBinance
	case "bitstamp":
		return connectorBitstamp
	case "coinbaseprime":
		return connectorCoinbaseprime
	case "column":
		return connectorColumn
	case "fireblocks":
		return connectorFireblocks
	case "increase":
		return connectorIncrease
	case "kraken":
		return connectorKraken
	case "plaid":
		return connectorPlaid
	case "powens":
		return connectorPowens
	case "qonto":
		return connectorQonto
	case "tink":
		return connectorTink
	default:
		return provider
	}
}

func toV3Provider(provider string) string {
	switch provider {
	case connectorAdyen:
		return "adyen"
	case connectorAtlar:
		return "atlar"
	case connectorBankingCircle:
		return "bankingcircle"
	case connectorCurrencyCloud:
		return "currencycloud"
	case connectorDummyPay:
		return "dummypay"
	case connectorGeneric:
		return "generic"
	case connectorMangopay:
		return "mangopay"
	case connectorModulr:
		return "modulr"
	case connectorMoneycorp:
		return "moneycorp"
	case connectorStripe:
		return "stripe"
	case connectorWise:
		return "wise"
	case connectorBinance:
		return "binance"
	case connectorBitstamp:
		return "bitstamp"
	case connectorCoinbaseprime:
		return "coinbaseprime"
	case connectorColumn:
		return "column"
	case connectorFireblocks:
		return "fireblocks"
	case connectorIncrease:
		return "increase"
	case connectorKraken:
		return "kraken"
	case connectorPlaid:
		return "plaid"
	case connectorPowens:
		return "powens"
	case connectorQonto:
		return "qonto"
	case connectorTink:
		return "tink"
	default:
		return provider
	}
}

const (
	paymentSchemeUnknown    string = "unknown"
	paymentSchemeOther      string = "other"
	paymentSchemeVisa       string = "visa"
	paymentSchemeMastercard string = "mastercard"
	paymentSchemeAmex       string = "amex"
	paymentSchemeDiners     string = "diners"
	paymentSchemeDiscover   string = "discover"
	paymentSchemeJcb        string = "jcb"
	paymentSchemeUnionpay   string = "unionpay"
	paymentSchemeAlipay     string = "alipay"
	paymentSchemeCup        string = "cup"
	paymentSchemeSepaDebit  string = "sepa debit"
	paymentSchemeSepaCredit string = "sepa credit"
	paymentSchemeSepa       string = "sepa"
	paymentSchemeApplePay   string = "apple pay"
	paymentSchemeGooglePay  string = "google pay"
	paymentSchemeDoku       string = "doku"
	paymentSchemeDragonpay  string = "dragonpay"
	paymentSchemeMaestro    string = "maestro"
	paymentSchemeMolpay     string = "molpay"
	paymentSchemeA2a        string = "a2a"
	paymentSchemeAchDebit   string = "ach debit"
	paymentSchemeAch        string = "ach"
	paymentSchemeRtp        string = "rtp"
)

func toV2PaymentScheme(scheme models.PaymentScheme) string {
	switch scheme {
	case models.PAYMENT_SCHEME_UNKNOWN:
		return paymentSchemeUnknown
	case models.PAYMENT_SCHEME_CARD_VISA:
		return paymentSchemeVisa
	case models.PAYMENT_SCHEME_CARD_MASTERCARD:
		return paymentSchemeMastercard
	case models.PAYMENT_SCHEME_CARD_AMEX:
		return paymentSchemeAmex
	case models.PAYMENT_SCHEME_CARD_DINERS:
		return paymentSchemeDiners
	case models.PAYMENT_SCHEME_CARD_DISCOVER:
		return paymentSchemeDiscover
	case models.PAYMENT_SCHEME_CARD_JCB:
		return paymentSchemeJcb
	case models.PAYMENT_SCHEME_CARD_UNION_PAY:
		return paymentSchemeUnionpay
	case models.PAYMENT_SCHEME_CARD_ALIPAY:
		return paymentSchemeAlipay
	case models.PAYMENT_SCHEME_CARD_CUP:
		return paymentSchemeCup
	case models.PAYMENT_SCHEME_SEPA_DEBIT:
		return paymentSchemeSepaDebit
	case models.PAYMENT_SCHEME_SEPA_CREDIT:
		return paymentSchemeSepaCredit
	case models.PAYMENT_SCHEME_SEPA:
		return paymentSchemeSepa
	case models.PAYMENT_SCHEME_GOOGLE_PAY:
		return paymentSchemeGooglePay
	case models.PAYMENT_SCHEME_APPLE_PAY:
		return paymentSchemeApplePay
	case models.PAYMENT_SCHEME_DOKU:
		return paymentSchemeDoku
	case models.PAYMENT_SCHEME_DRAGON_PAY:
		return paymentSchemeDragonpay
	case models.PAYMENT_SCHEME_MAESTRO:
		return paymentSchemeMaestro
	case models.PAYMENT_SCHEME_MOL_PAY:
		return paymentSchemeMolpay
	case models.PAYMENT_SCHEME_A2A:
		return paymentSchemeA2a
	case models.PAYMENT_SCHEME_ACH_DEBIT:
		return paymentSchemeAchDebit
	case models.PAYMENT_SCHEME_ACH:
		return paymentSchemeAch
	case models.PAYMENT_SCHEME_RTP:
		return paymentSchemeRtp
	case models.PAYMENT_SCHEME_OTHER:
		return paymentSchemeOther
	default:
		return paymentSchemeUnknown
	}
}

func toV3PaymentScheme(scheme string) string {
	switch scheme {
	case "unknown":
		return models.PAYMENT_SCHEME_UNKNOWN.String()
	case "other":
		return models.PAYMENT_SCHEME_OTHER.String()
	case "visa":
		return models.PAYMENT_SCHEME_CARD_VISA.String()
	case "mastercard":
		return models.PAYMENT_SCHEME_CARD_MASTERCARD.String()
	case "amex":
		return models.PAYMENT_SCHEME_CARD_AMEX.String()
	case "diners":
		return models.PAYMENT_SCHEME_CARD_DINERS.String()
	case "discover":
		return models.PAYMENT_SCHEME_CARD_DISCOVER.String()
	case "jcb":
		return models.PAYMENT_SCHEME_CARD_JCB.String()
	case "unionpay":
		return models.PAYMENT_SCHEME_CARD_UNION_PAY.String()
	case "alipay":
		return models.PAYMENT_SCHEME_CARD_ALIPAY.String()
	case "cup":
		return models.PAYMENT_SCHEME_CARD_CUP.String()
	case "sepa debit":
		return models.PAYMENT_SCHEME_SEPA_DEBIT.String()
	case "sepa credit":
		return models.PAYMENT_SCHEME_SEPA_CREDIT.String()
	case "sepa":
		return models.PAYMENT_SCHEME_SEPA.String()
	case "apple pay":
		return models.PAYMENT_SCHEME_APPLE_PAY.String()
	case "google pay":
		return models.PAYMENT_SCHEME_GOOGLE_PAY.String()
	case "doku":
		return models.PAYMENT_SCHEME_DOKU.String()
	case "dragonpay":
		return models.PAYMENT_SCHEME_DRAGON_PAY.String()
	case "maestro":
		return models.PAYMENT_SCHEME_MAESTRO.String()
	case "molpay":
		return models.PAYMENT_SCHEME_MOL_PAY.String()
	case "a2a":
		return models.PAYMENT_SCHEME_A2A.String()
	case "ach debit":
		return models.PAYMENT_SCHEME_ACH_DEBIT.String()
	case "ach":
		return models.PAYMENT_SCHEME_ACH.String()
	case "rtp":
		return models.PAYMENT_SCHEME_RTP.String()
	default:
		return scheme
	}
}
