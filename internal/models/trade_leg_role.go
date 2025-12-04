package models

type TradeLegRole string

const (
	TRADE_LEG_ROLE_BASE  TradeLegRole = "BASE"
	TRADE_LEG_ROLE_QUOTE TradeLegRole = "QUOTE"
)

func (r TradeLegRole) String() string {
	return string(r)
}

func (r TradeLegRole) IsValid() bool {
	switch r {
	case TRADE_LEG_ROLE_BASE, TRADE_LEG_ROLE_QUOTE:
		return true
	default:
		return false
	}
}

