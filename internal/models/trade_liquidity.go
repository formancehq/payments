package models

type TradeLiquidity string

const (
	TRADE_LIQUIDITY_MAKER   TradeLiquidity = "MAKER"
	TRADE_LIQUIDITY_TAKER   TradeLiquidity = "TAKER"
	TRADE_LIQUIDITY_UNKNOWN TradeLiquidity = "UNKNOWN"
)

func (l TradeLiquidity) String() string {
	return string(l)
}

func (l TradeLiquidity) IsValid() bool {
	switch l {
	case TRADE_LIQUIDITY_MAKER, TRADE_LIQUIDITY_TAKER, TRADE_LIQUIDITY_UNKNOWN:
		return true
	default:
		return false
	}
}

func MustTradeLiquidityFromString(s string) TradeLiquidity {
	liquidity := TradeLiquidity(s)
	if !liquidity.IsValid() {
		panic("invalid trade liquidity: " + s)
	}
	return liquidity
}

