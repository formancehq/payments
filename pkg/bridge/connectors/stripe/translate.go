package stripe

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/stripe/stripe-go/v72"
)

type Currency struct {
	Decimals int
}

var Currencies = map[string]Currency{
	"ARS": {2}, //  Argentine Peso
	"AMD": {2}, //  Armenian Dram
	"AWG": {2}, //  Aruban Guilder
	"AUD": {2}, //  Australian Dollar
	"BSD": {2}, //  Bahamian Dollar
	"BHD": {3}, //  Bahraini Dinar
	"BDT": {2}, //  Bangladesh, Taka
	"BZD": {2}, //  Belize Dollar
	"BMD": {2}, //  Bermudian Dollar
	"BOB": {2}, //  Bolivia, Boliviano
	"BAM": {2}, //  Bosnia and Herzegovina, Convertible Marks
	"BWP": {2}, //  Botswana, Pula
	"BRL": {2}, //  Brazilian Real
	"BND": {2}, //  Brunei Dollar
	"CAD": {2}, //  Canadian Dollar
	"KYD": {2}, //  Cayman Islands Dollar
	"CLP": {0}, //  Chilean Peso
	"CNY": {2}, //  China Yuan Renminbi
	"COP": {2}, //  Colombian Peso
	"CRC": {2}, //  Costa Rican Colon
	"HRK": {2}, //  Croatian Kuna
	"CUC": {2}, //  Cuban Convertible Peso
	"CUP": {2}, //  Cuban Peso
	"CYP": {2}, //  Cyprus Pound
	"CZK": {2}, //  Czech Koruna
	"DKK": {2}, //  Danish Krone
	"DOP": {2}, //  Dominican Peso
	"XCD": {2}, //  East Caribbean Dollar
	"EGP": {2}, //  Egyptian Pound
	"SVC": {2}, //  El Salvador Colon
	"ATS": {2}, //  Euro
	"BEF": {2}, //  Euro
	"DEM": {2}, //  Euro
	"EEK": {2}, //  Euro
	"ESP": {2}, //  Euro
	"EUR": {2}, //  Euro
	"FIM": {2}, //  Euro
	"FRF": {2}, //  Euro
	"GRD": {2}, //  Euro
	"IEP": {2}, //  Euro
	"ITL": {2}, //  Euro
	"LUF": {2}, //  Euro
	"NLG": {2}, //  Euro
	"PTE": {2}, //  Euro
	"GHC": {2}, //  Ghana, Cedi
	"GIP": {2}, //  Gibraltar Pound
	"GTQ": {2}, //  Guatemala, Quetzal
	"HNL": {2}, //  Honduras, Lempira
	"HKD": {2}, //  Hong Kong Dollar
	"HUF": {0}, //  Hungary, Forint
	"ISK": {0}, //  Iceland Krona
	"INR": {2}, //  Indian Rupee
	"IDR": {2}, //  Indonesia, Rupiah
	"IRR": {2}, //  Iranian Rial
	"JMD": {2}, //  Jamaican Dollar
	"JPY": {0}, //  Japan, Yen
	"JOD": {3}, //  Jordanian Dinar
	"KES": {2}, //  Kenyan Shilling
	"KWD": {3}, //  Kuwaiti Dinar
	"LVL": {2}, //  Latvian Lats
	"LBP": {0}, //  Lebanese Pound
	"LTL": {2}, //  Lithuanian Litas
	"MKD": {2}, //  Macedonia, Denar
	"MYR": {2}, //  Malaysian Ringgit
	"MTL": {2}, //  Maltese Lira
	"MUR": {0}, //  Mauritius Rupee
	"MXN": {2}, //  Mexican Peso
	"MZM": {2}, //  Mozambique Metical
	"NPR": {2}, //  Nepalese Rupee
	"ANG": {2}, //  Netherlands Antillian Guilder
	"ILS": {2}, //  New Israeli Shekel
	"TRY": {2}, //  New Turkish Lira
	"NZD": {2}, //  New Zealand Dollar
	"NOK": {2}, //  Norwegian Krone
	"PKR": {2}, //  Pakistan Rupee
	"PEN": {2}, //  Peru, Nuevo Sol
	"UYU": {2}, //  Peso Uruguayo
	"PHP": {2}, //  Philippine Peso
	"PLN": {2}, //  Poland, Zloty
	"GBP": {2}, //  Pound Sterling
	"OMR": {3}, //  Rial Omani
	"RON": {2}, //  Romania, New Leu
	"ROL": {2}, //  Romania, Old Leu
	"RUB": {2}, //  Russian Ruble
	"SAR": {2}, //  Saudi Riyal
	"SGD": {2}, //  Singapore Dollar
	"SKK": {2}, //  Slovak Koruna
	"SIT": {2}, //  Slovenia, Tolar
	"ZAR": {2}, //  South Africa, Rand
	"KRW": {0}, //  South Korea, Won
	"SZL": {2}, //  Swaziland, Lilangeni
	"SEK": {2}, //  Swedish Krona
	"CHF": {2}, //  Swiss Franc
	"TZS": {2}, //  Tanzanian Shilling
	"THB": {2}, //  Thailand, Baht
	"TOP": {2}, //  Tonga, Paanga
	"AED": {2}, //  UAE Dirham
	"UAH": {2}, //  Ukraine, Hryvnia
	"USD": {2}, //  US Dollar
	"VUV": {0}, //  Vanuatu, Vatu
	"VEF": {2}, //  Venezuela Bolivares Fuertes
	"VEB": {2}, //  Venezuela, Bolivar
	"VND": {0}, //  Viet Nam, Dong
	"ZWD": {2}, //  Zimbabwe Dollar
}

func CreateBatchElement(bt *stripe.BalanceTransaction, forward bool) (ingestion.BatchElement, bool) {
	var (
		reference   payments.Referenced
		paymentData *payments.Data
		adjustment  *payments.Adjustment
	)
	defer func() {
		// DEBUG
		if e := recover(); e != nil {
			fmt.Println("Error translating transaction")
			debug.PrintStack()
			spew.Dump(bt)
			panic(e)
		}
	}()

	if bt.Source == nil {
		return ingestion.BatchElement{}, false
	}

	formatAsset := func(cur stripe.Currency) string {
		asset := strings.ToUpper(string(cur))
		def, ok := Currencies[asset]
		if !ok {
			return asset
		}
		if def.Decimals == 0 {
			return asset
		}
		return fmt.Sprintf("%s/%d", asset, def.Decimals)
	}

	convertPayoutStatus := func() (status payments.Status) {
		switch bt.Source.Payout.Status {
		case stripe.PayoutStatusCanceled:
			status = payments.StatusCancelled
		case stripe.PayoutStatusFailed:
			status = payments.StatusFailed
		case stripe.PayoutStatusInTransit, stripe.PayoutStatusPending:
			status = payments.StatusPending
		case stripe.PayoutStatusPaid:
			status = payments.StatusSucceeded
		}
		return
	}

	switch bt.Type {
	case "charge":
		reference = payments.Referenced{
			Reference: bt.Source.Charge.ID,
			Type:      payments.TypePayIn,
		}
		paymentData = &payments.Data{
			Status:        payments.StatusSucceeded,
			InitialAmount: bt.Source.Charge.Amount,
			Asset:         formatAsset(bt.Source.Charge.Currency),
			Raw:           bt,
			Scheme:        payments.Scheme(bt.Source.Charge.PaymentMethodDetails.Card.Brand),
			CreatedAt:     time.Unix(bt.Created, 0),
		}
	case "payout":
		reference = payments.Referenced{
			Reference: bt.Source.Payout.ID,
			Type:      payments.TypePayout,
		}
		paymentData = &payments.Data{
			Status:        convertPayoutStatus(),
			InitialAmount: bt.Source.Payout.Amount,
			Raw:           bt,
			Asset:         formatAsset(bt.Source.Payout.Currency),
			Scheme: func() payments.Scheme {
				switch bt.Source.Payout.Type {
				case "bank_account":
					return payments.SchemeSepaCredit
				case "card":
					return payments.Scheme(bt.Source.Payout.Card.Brand)
				}
				return payments.SchemeUnknown
			}(),
			CreatedAt: time.Unix(bt.Created, 0),
		}

	case "transfer":
		reference = payments.Referenced{
			Reference: bt.Source.Transfer.ID,
			Type:      payments.TypePayout,
		}
		paymentData = &payments.Data{
			Status:        payments.StatusSucceeded,
			InitialAmount: bt.Source.Transfer.Amount,
			Raw:           bt,
			Asset:         formatAsset(bt.Source.Transfer.Currency),
			Scheme:        payments.SchemeOther,
			CreatedAt:     time.Unix(bt.Created, 0),
		}
	case "refund":
		reference = payments.Referenced{
			Reference: bt.Source.Refund.Charge.ID,
			Type:      payments.TypePayIn,
		}
		adjustment = &payments.Adjustment{
			Status: payments.StatusSucceeded,
			Amount: bt.Amount,
			Date:   time.Unix(bt.Created, 0),
			Raw:    bt,
		}
	case "payment":
		reference = payments.Referenced{
			Reference: bt.Source.Charge.ID,
			Type:      payments.TypePayIn,
		}
		paymentData = &payments.Data{
			Status:        payments.StatusSucceeded,
			InitialAmount: bt.Source.Charge.Amount,
			Raw:           bt,
			Asset:         formatAsset(bt.Source.Charge.Currency),
			Scheme:        payments.SchemeOther,
			CreatedAt:     time.Unix(bt.Created, 0),
		}
	case "payout_cancel":
		reference = payments.Referenced{
			Reference: bt.Source.Payout.ID,
			Type:      payments.TypePayout,
		}
		adjustment = &payments.Adjustment{
			Status:   convertPayoutStatus(),
			Amount:   0,
			Date:     time.Unix(bt.Created, 0),
			Raw:      bt,
			Absolute: true,
		}
	case "payout_failure":
		reference = payments.Referenced{
			Reference: bt.Source.Payout.ID,
			Type:      payments.TypePayIn,
		}
		adjustment = &payments.Adjustment{
			Status:   convertPayoutStatus(),
			Amount:   0,
			Date:     time.Unix(bt.Created, 0),
			Raw:      bt,
			Absolute: true,
		}
	case "payment_refund":
		reference = payments.Referenced{
			Reference: bt.Source.Refund.Charge.ID,
			Type:      payments.TypePayIn,
		}
		adjustment = &payments.Adjustment{
			Status: payments.StatusSucceeded,
			Amount: bt.Amount,
			Date:   time.Unix(bt.Created, 0),
			Raw:    bt,
		}
	case "adjustment":
		reference = payments.Referenced{
			Reference: bt.Source.Dispute.Charge.ID,
			Type:      payments.TypePayIn,
		}
		adjustment = &payments.Adjustment{
			Status: payments.StatusCancelled,
			Amount: bt.Amount,
			Date:   time.Unix(bt.Created, 0),
			Raw:    bt,
		}
	case "stripe_fee", "network_cost":
		return ingestion.BatchElement{}, true
	default:
		return ingestion.BatchElement{}, false
	}

	return ingestion.BatchElement{
		Referenced: reference,
		Payment:    paymentData,
		Adjustment: adjustment,
		Forward:    forward,
	}, true
}
