package stripe

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	payment "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge"
	"github.com/stripe/stripe-go/v72"
	"runtime/debug"
	"strings"
	"time"
)

type currency struct {
	decimals int
}

var currencies = map[string]currency{
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

func CreateBatchElement(bt *stripe.BalanceTransaction, forward bool) (bridge.BatchElement, bool) {
	var (
		identifier  payment.Identifier
		paymentData *payment.Data
		adjustment  *payment.Adjustment
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

	formatAsset := func(cur stripe.Currency) string {
		asset := strings.ToUpper(string(cur))
		def, ok := currencies[asset]
		if !ok {
			return asset
		}
		if def.decimals == 0 {
			return asset
		}
		return fmt.Sprintf("%s/%d", asset, def.decimals)
	}

	switch bt.Type {
	case "charge":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Charge.ID,
			Type:      payment.TypePayIn,
		}
		paymentData = &payment.Data{
			Status:        string(bt.Status),
			InitialAmount: bt.Source.Charge.Amount,
			Asset:         formatAsset(bt.Source.Charge.Currency),
			Raw:           bt,
			Scheme:        payment.Scheme(bt.Source.Charge.PaymentMethodDetails.Card.Brand),
			CreatedAt:     time.Unix(bt.Created, 0),
		}
	case "payout":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Payout.ID,
			Type:      payment.TypePayout,
		}
		paymentData = &payment.Data{
			Status:        string(bt.Status),
			InitialAmount: bt.Source.Payout.Amount,
			Raw:           bt,
			Asset:         formatAsset(bt.Source.Payout.Currency),
			Scheme:        "", // TODO
			CreatedAt:     time.Unix(bt.Created, 0),
		}

	case "transfer":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Transfer.ID,
			Type:      payment.TypePayout,
		}
		paymentData = &payment.Data{
			Status:        string(bt.Status),
			InitialAmount: bt.Source.Transfer.Amount,
			Raw:           bt,
			Asset:         formatAsset(bt.Source.Transfer.Currency),
			Scheme:        payment.SchemeSepa, // TODO: Check with clem
			CreatedAt:     time.Unix(bt.Created, 0),
		}
	case "refund":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Refund.Charge.ID,
			Type:      payment.TypePayIn,
		}
		adjustment = &payment.Adjustment{
			Status: string(bt.Status),
			Amount: bt.Amount,
			Date:   time.Unix(bt.Created, 0),
			Raw:    bt,
		}
	case "payment":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Charge.ID,
			Type:      payment.TypePayIn,
		}
		paymentData = &payment.Data{
			Status:        string(bt.Status),
			InitialAmount: bt.Source.Charge.Amount,
			Raw:           bt,
			Asset:         formatAsset(bt.Source.Charge.Currency),
			Scheme:        payment.SchemeSepa,
			CreatedAt:     time.Unix(bt.Created, 0),
		}
	case "stripe_fee":
	case "network_cost":
	case "payout_cancel":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Payout.ID,
			Type:      payment.TypePayout,
		}
		adjustment = &payment.Adjustment{
			Status:   string(bt.Status),
			Amount:   0,
			Date:     time.Unix(bt.Created, 0),
			Raw:      bt,
			Absolute: true,
		}
	case "payout_failure":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Payout.ID,
			Type:      payment.TypePayIn,
		}
		adjustment = &payment.Adjustment{
			Status:   string(bt.Status),
			Amount:   0,
			Date:     time.Unix(bt.Created, 0),
			Raw:      bt,
			Absolute: true,
		}
	case "payment_refund":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Refund.Charge.ID,
			Type:      payment.TypePayIn,
		}
		adjustment = &payment.Adjustment{
			Status: string(bt.Status),
			Amount: bt.Amount,
			Date:   time.Unix(bt.Created, 0),
			Raw:    bt,
		}
	case "adjustment":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Dispute.Charge.ID,
			Type:      payment.TypePayIn,
		}
		adjustment = &payment.Adjustment{
			Status: string(bt.Status),
			Amount: bt.Amount,
			Date:   time.Unix(bt.Created, 0),
			Raw:    bt,
		}
	default:
		return bridge.BatchElement{}, false
	}

	return bridge.BatchElement{
		Identifier: identifier,
		Payment:    paymentData,
		Adjustment: adjustment,
		Forward:    forward,
	}, true
}
