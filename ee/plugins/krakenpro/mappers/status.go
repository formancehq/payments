package mappers

import "github.com/formancehq/payments/pkg/domain/models"

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

// ledgerTypeEntry is one row in the declarative classification table.
type ledgerTypeEntry struct {
	kind        LedgerTypeKind
	paymentType models.PaymentType
}

// ledgerTypes is the single source of truth for the Kraken ledger
// `type` enum. Adding a new ledger type means adding one row here —
// IsKnownLedgerType + ClassifyLedgerType both derive from this map
// so classification and "do we recognise this value?" can never
// drift apart. Direction (source vs destination) is not encoded here;
// it is derived from the amount sign by the payment mapper.
var ledgerTypes = map[string]ledgerTypeEntry{
	"deposit":    {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
	"withdrawal": {LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT},

	// transfer / custodytransfer are internal movements between the
	// owner's own wallets (spot<->futures, subaccount, spot<->staking
	// allocation, often carrying a `subtype` such as spottostaking) —
	// not external pay-ins/payouts — so they map to TRANSFER. Staking
	// REWARDS are income and stay PAYIN below (Kraken's `staking` type);
	// confirm against UAT whether rewards arrive as `staking` or as a
	// `transfer` subtype.
	"transfer":        {LedgerKindPayment, models.PAYMENT_TYPE_TRANSFER},
	"custodytransfer": {LedgerKindPayment, models.PAYMENT_TYPE_TRANSFER},

	"staking":  {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
	"reward":   {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
	"dividend": {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
	"credit":   {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
	// Kraken NFT rebate — both spellings observed across API versions.
	"nft_rebate": {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
	"nftrebate":  {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},

	// Instant buy/sell (Kraken "Spend"/"Receive"): a spend debits the
	// funding asset (PAYOUT), a receive credits the bought asset (PAYIN).
	"spend":   {LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT},
	"receive": {LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},

	"nftcreatorfee": {LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT},

	"adjustment":    {LedgerKindPayment, models.PAYMENT_TYPE_OTHER},
	"rollover":      {LedgerKindPayment, models.PAYMENT_TYPE_OTHER},
	"settled":       {LedgerKindPayment, models.PAYMENT_TYPE_OTHER},
	"reserve":       {LedgerKindPayment, models.PAYMENT_TYPE_OTHER},
	"reserved_fee":  {LedgerKindPayment, models.PAYMENT_TYPE_OTHER},
	"ic_settlement": {LedgerKindPayment, models.PAYMENT_TYPE_OTHER},
	"nfttrade":      {LedgerKindPayment, models.PAYMENT_TYPE_OTHER},

	// Trade-side ledgers are dispatched to FETCH_ORDERS (handled via
	// OpenOrders/ClosedOrders), so they're skipped by the payments
	// orchestrator.
	"trade":   {LedgerKindOrder, models.PAYMENT_TYPE_OTHER},
	"eqtrade": {LedgerKindOrder, models.PAYMENT_TYPE_OTHER},

	// Conversions — spot + margin + derivatives variants. Spot-only
	// accounts only ever see the first four; the derivatives-* rows
	// are classified for exhaustiveness so a margin-enabled account
	// doesn't silently fall through to PAYMENT_TYPE_OTHER.
	"conversion":                  {LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
	"sale":                        {LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
	"marginconversion":            {LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
	"margin_conversion":           {LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
	"derivativesflexconversion":   {LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
	"derivativestaxconversion":    {LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
	"derivativesconversioncredit": {LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
	"collateralconversion":        {LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
}

// ClassifyLedgerType returns the orchestrator-routing kind and the
// canonical payment type. Unknown types fall through to
// PAYMENT_TYPE_OTHER on the payments pipeline so the orchestrator can
// log + emit them (catalogue L8: surface enum gaps loudly).
func ClassifyLedgerType(t string) (LedgerTypeKind, models.PaymentType) {
	if e, ok := ledgerTypes[t]; ok {
		return e.kind, e.paymentType
	}
	return LedgerKindPayment, models.PAYMENT_TYPE_OTHER
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
