package mappers

import "github.com/formancehq/payments/internal/models"

// LedgerTypeKind classifies a Kraken ledger row at the orchestrator
// level. Trade-related rows belong to FETCH_ORDERS, conversion rows
// to FETCH_CONVERSIONS, the rest to FETCH_PAYMENTS (or are skipped).
type LedgerTypeKind int

const (
	LedgerKindUnknown    LedgerTypeKind = iota
	LedgerKindPayment                   // emit as PSPPayment
	LedgerKindOrder                     // belongs to TradesHistory pipeline, skip here
	LedgerKindConversion                // emit via FETCH_CONVERSIONS pairing
)

// ledgerTypeEntry is one row in the declarative classification
// table. signDriven=true means the row's amount sign decides
// PAYIN vs PAYOUT (the `transfer` family); otherwise paymentType
// is the canonical mapping.
type ledgerTypeEntry struct {
	kind        LedgerTypeKind
	paymentType models.PaymentType
	signDriven  bool
}

// ledgerTypes is the single source of truth for the Kraken ledger
// `type` enum. Adding a new ledger type means adding one row here —
// IsKnownLedgerType + ClassifyLedgerType both derive from this map
// so classification and "do we recognise this value?" can never
// drift apart.
var ledgerTypes = map[string]ledgerTypeEntry{
	"deposit":    {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
	"withdrawal": {LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT, false},

	// "transfer" is a real Kraken ledger type (spot<->futures and
	// subaccount moves). Sign-driven: direction depends on the amount
	// sign — positive credits the account (PAYIN), negative debits (PAYOUT).
	"transfer": {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, true},

	"staking":  {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
	"reward":   {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
	"dividend": {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
	"credit":   {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
	// Kraken NFT rebate — both spellings observed across API versions.
	"nft_rebate": {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
	"nftrebate":  {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},

	// Instant buy/sell (Kraken "Spend"/"Receive"): a spend debits the
	// funding asset (PAYOUT), a receive credits the bought asset (PAYIN).
	"spend":   {LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT, false},
	"receive": {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},

	"nftcreatorfee": {LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT, false},

	"adjustment":      {LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
	"rollover":        {LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
	"settled":         {LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
	"reserve":         {LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
	"reserved_fee":    {LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
	"ic_settlement":   {LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
	"nfttrade":        {LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
	"custodytransfer": {LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},

	// Trade-side ledgers are dispatched to FETCH_ORDERS (handled via
	// OpenOrders/ClosedOrders), so they're skipped by the payments
	// orchestrator.
	"trade":   {LedgerKindOrder, models.PAYMENT_TYPE_OTHER, false},
	"eqtrade": {LedgerKindOrder, models.PAYMENT_TYPE_OTHER, false},

	// Conversions — spot + margin + derivatives variants. Spot-only
	// accounts only ever see the first four; the derivatives-* rows
	// are classified for exhaustiveness so a margin-enabled account
	// doesn't silently fall through to PAYMENT_TYPE_OTHER.
	"conversion":                  {LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
	"sale":                        {LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
	"marginconversion":            {LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
	"margin_conversion":           {LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
	"derivativesflexconversion":   {LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
	"derivativestaxconversion":    {LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
	"derivativesconversioncredit": {LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
	"collateralconversion":        {LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
}

// ClassifyLedgerType returns the orchestrator-routing kind, the
// canonical payment type, and whether the sign of the amount should
// override the payment type (transfer family). Unknown types fall
// through to PAYMENT_TYPE_OTHER on the payments pipeline so the
// orchestrator can log + emit them (catalogue L8: surface enum gaps
// loudly).
func ClassifyLedgerType(t string) (LedgerTypeKind, models.PaymentType, bool) {
	if e, ok := ledgerTypes[t]; ok {
		return e.kind, e.paymentType, e.signDriven
	}
	return LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false
}

// IsKnownLedgerType reports whether t is in the classification table.
// Used by orchestrators to differentiate "expected enum value" from
// "previously-unseen value that should surface in logs".
func IsKnownLedgerType(t string) bool {
	_, ok := ledgerTypes[t]
	return ok
}

// MapOrderType maps Kraken's ordertype string to models.OrderType.
// The second return value reports whether the input was recognised
// (so the orchestrator can Infof when a new value surfaces — the
// logging interface has no Warnf level).
func MapOrderType(s string) (models.OrderType, bool) {
	switch s {
	case "market":
		return models.ORDER_TYPE_MARKET, true
	case "limit":
		return models.ORDER_TYPE_LIMIT, true
	case "stop-loss":
		return models.ORDER_TYPE_STOP, true
	case "stop-loss-limit":
		return models.ORDER_TYPE_STOP_LIMIT, true
	case "trailing-stop":
		return models.ORDER_TYPE_TRAILING_STOP, true
	case "trailing-stop-limit":
		return models.ORDER_TYPE_TRAILING_STOP_LIMIT, true
	case "take-profit":
		return models.ORDER_TYPE_TAKE_PROFIT, true
	case "take-profit-limit":
		return models.ORDER_TYPE_TAKE_PROFIT_LIMIT, true
	case "limit-maker":
		return models.ORDER_TYPE_LIMIT_MAKER, true
	case "iceberg", "settle-position":
		// No direct equivalent in the platform enum; fall back to MARKET
		// rather than UNKNOWN so the order still validates.
		return models.ORDER_TYPE_MARKET, true
	default:
		return models.ORDER_TYPE_UNKNOWN, false
	}
}
