package models

type TradeOrderType string

const (
	TRADE_ORDER_TYPE_MARKET      TradeOrderType = "MARKET"
	TRADE_ORDER_TYPE_LIMIT       TradeOrderType = "LIMIT"
	TRADE_ORDER_TYPE_STOP_LIMIT  TradeOrderType = "STOP_LIMIT"
	TRADE_ORDER_TYPE_STOP_MARKET TradeOrderType = "STOP_MARKET"
	TRADE_ORDER_TYPE_RFQ         TradeOrderType = "RFQ"
)

func (t TradeOrderType) String() string {
	return string(t)
}

func (t TradeOrderType) IsValid() bool {
	switch t {
	case TRADE_ORDER_TYPE_MARKET,
		TRADE_ORDER_TYPE_LIMIT,
		TRADE_ORDER_TYPE_STOP_LIMIT,
		TRADE_ORDER_TYPE_STOP_MARKET,
		TRADE_ORDER_TYPE_RFQ:
		return true
	default:
		return false
	}
}

func MustTradeOrderTypeFromString(s string) TradeOrderType {
	orderType := TradeOrderType(s)
	if !orderType.IsValid() {
		panic("invalid trade order type: " + s)
	}
	return orderType
}

