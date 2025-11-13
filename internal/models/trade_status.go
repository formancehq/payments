package models

type TradeStatus string

const (
	TRADE_STATUS_OPEN              TradeStatus = "OPEN"
	TRADE_STATUS_PARTIALLY_FILLED  TradeStatus = "PARTIALLY_FILLED"
	TRADE_STATUS_FILLED            TradeStatus = "FILLED"
	TRADE_STATUS_CANCELED          TradeStatus = "CANCELED"
	TRADE_STATUS_REJECTED          TradeStatus = "REJECTED"
	TRADE_STATUS_EXPIRED           TradeStatus = "EXPIRED"
)

func (s TradeStatus) String() string {
	return string(s)
}

func (s TradeStatus) IsValid() bool {
	switch s {
	case TRADE_STATUS_OPEN,
		TRADE_STATUS_PARTIALLY_FILLED,
		TRADE_STATUS_FILLED,
		TRADE_STATUS_CANCELED,
		TRADE_STATUS_REJECTED,
		TRADE_STATUS_EXPIRED:
		return true
	default:
		return false
	}
}

func MustTradeStatusFromString(s string) TradeStatus {
	status := TradeStatus(s)
	if !status.IsValid() {
		panic("invalid trade status: " + s)
	}
	return status
}

