package models

type TradeFeeAppliedOn string

const (
	TRADE_FEE_APPLIED_ON_QUOTE TradeFeeAppliedOn = "QUOTE"
	TRADE_FEE_APPLIED_ON_BASE  TradeFeeAppliedOn = "BASE"
	TRADE_FEE_APPLIED_ON_OTHER TradeFeeAppliedOn = "OTHER"
)

func (a TradeFeeAppliedOn) String() string {
	return string(a)
}

func (a TradeFeeAppliedOn) IsValid() bool {
	switch a {
	case TRADE_FEE_APPLIED_ON_QUOTE, TRADE_FEE_APPLIED_ON_BASE, TRADE_FEE_APPLIED_ON_OTHER:
		return true
	default:
		return false
	}
}

func MustTradeFeeAppliedOnFromString(s string) TradeFeeAppliedOn {
	appliedOn := TradeFeeAppliedOn(s)
	if !appliedOn.IsValid() {
		panic("invalid trade fee applied on: " + s)
	}
	return appliedOn
}

