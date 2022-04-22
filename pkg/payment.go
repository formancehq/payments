package payment

import (
	"encoding/json"
	"time"
)

const (
	SchemeVisa       = "visa"
	SchemeMasterCard = "mastercard"
	SchemeApplePay   = "apple pay"
	SchemeGooglePay  = "google pay"
	SchemeSepaDebit  = "sepa debit"
	SchemeSepaCredit = "sepa credit"
	SchemeSepa       = "sepa"
	SchemeA2A        = "a2a"
	SchemeAchDebit   = "ach debit"
	SchemeAch        = "ach"
	SchemeRtp        = "rtp"
	SchemeOther      = "other"

	TypePayIn  = "pay-in"
	TypePayout = "payout"
	TypeOther  = "other"
)

type Value struct {
	Amount int64  `json:"amount"`
	Asset  string `json:"asset"`
}

type Payment struct {
	ID        string          `json:"id" bson:"id"`
	Provider  string          `json:"provider" bson:"provider"`
	Reference string          `json:"reference" bson:"reference"`
	Scheme    string          `json:"scheme" bson:"scheme"`
	Type      string          `json:"type" bson:"type"`
	Status    string          `json:"status" bson:"status"`
	Value     Value           `json:"value" bson:"value"`
	Date      time.Time       `json:"date" bson:"date"`
	Raw       json.RawMessage `json:"raw" bson:"raw"`
}
