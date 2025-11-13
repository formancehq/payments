package models

type TradeFeeKind string

const (
	TRADE_FEE_KIND_MAKER TradeFeeKind = "MAKER"
	TRADE_FEE_KIND_TAKER TradeFeeKind = "TAKER"
	TRADE_FEE_KIND_OTHER TradeFeeKind = "OTHER"
)

func (k TradeFeeKind) String() string {
	return string(k)
}

func (k TradeFeeKind) IsValid() bool {
	switch k {
	case TRADE_FEE_KIND_MAKER, TRADE_FEE_KIND_TAKER, TRADE_FEE_KIND_OTHER:
		return true
	default:
		return false
	}
}

func MustTradeFeeKindFromString(s string) TradeFeeKind {
	kind := TradeFeeKind(s)
	if !kind.IsValid() {
		panic("invalid trade fee kind: " + s)
	}
	return kind
}

