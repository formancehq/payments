package models

type TradeSide string

const (
	TRADE_SIDE_BUY  TradeSide = "BUY"
	TRADE_SIDE_SELL TradeSide = "SELL"
)

func (s TradeSide) String() string {
	return string(s)
}

func (s TradeSide) IsValid() bool {
	switch s {
	case TRADE_SIDE_BUY, TRADE_SIDE_SELL:
		return true
	default:
		return false
	}
}

func MustTradeSideFromString(s string) TradeSide {
	side := TradeSide(s)
	if !side.IsValid() {
		panic("invalid trade side: " + s)
	}
	return side
}

