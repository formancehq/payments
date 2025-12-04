package models

type TradeLegDirection string

const (
	TRADE_LEG_DIRECTION_CREDIT TradeLegDirection = "CREDIT"
	TRADE_LEG_DIRECTION_DEBIT  TradeLegDirection = "DEBIT"
)

func (d TradeLegDirection) String() string {
	return string(d)
}

func (d TradeLegDirection) IsValid() bool {
	switch d {
	case TRADE_LEG_DIRECTION_CREDIT, TRADE_LEG_DIRECTION_DEBIT:
		return true
	default:
		return false
	}
}

