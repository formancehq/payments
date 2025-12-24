package models

type TradeExecutionModel string

const (
	TRADE_EXECUTION_MODEL_ORDER_BOOK TradeExecutionModel = "ORDER_BOOK"
	TRADE_EXECUTION_MODEL_RFQ        TradeExecutionModel = "RFQ"
)

func (t TradeExecutionModel) String() string {
	return string(t)
}

func (t TradeExecutionModel) IsValid() bool {
	switch t {
	case TRADE_EXECUTION_MODEL_ORDER_BOOK, TRADE_EXECUTION_MODEL_RFQ:
		return true
	default:
		return false
	}
}

func MustTradeExecutionModelFromString(s string) TradeExecutionModel {
	model := TradeExecutionModel(s)
	if !model.IsValid() {
		panic("invalid trade execution model: " + s)
	}
	return model
}

