package models

type TradeInstrumentType string

const (
	TRADE_INSTRUMENT_TYPE_SPOT TradeInstrumentType = "SPOT"
	TRADE_INSTRUMENT_TYPE_FX   TradeInstrumentType = "FX"
)

func (t TradeInstrumentType) String() string {
	return string(t)
}

func (t TradeInstrumentType) IsValid() bool {
	switch t {
	case TRADE_INSTRUMENT_TYPE_SPOT, TRADE_INSTRUMENT_TYPE_FX:
		return true
	default:
		return false
	}
}

func MustTradeInstrumentTypeFromString(s string) TradeInstrumentType {
	instrumentType := TradeInstrumentType(s)
	if !instrumentType.IsValid() {
		panic("invalid trade instrument type: " + s)
	}
	return instrumentType
}

