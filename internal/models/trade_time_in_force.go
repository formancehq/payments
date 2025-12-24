package models

type TradeTimeInForce string

const (
	TRADE_TIME_IN_FORCE_GTC TradeTimeInForce = "GTC" // Good Till Cancel
	TRADE_TIME_IN_FORCE_IOC TradeTimeInForce = "IOC" // Immediate Or Cancel
	TRADE_TIME_IN_FORCE_FOK TradeTimeInForce = "FOK" // Fill Or Kill
	TRADE_TIME_IN_FORCE_DAY TradeTimeInForce = "DAY" // Day
)

func (t TradeTimeInForce) String() string {
	return string(t)
}

func (t TradeTimeInForce) IsValid() bool {
	switch t {
	case TRADE_TIME_IN_FORCE_GTC,
		TRADE_TIME_IN_FORCE_IOC,
		TRADE_TIME_IN_FORCE_FOK,
		TRADE_TIME_IN_FORCE_DAY:
		return true
	default:
		return false
	}
}

func MustTradeTimeInForceFromString(s string) TradeTimeInForce {
	tif := TradeTimeInForce(s)
	if !tif.IsValid() {
		panic("invalid trade time in force: " + s)
	}
	return tif
}

