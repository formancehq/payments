package payment

import (
	"encoding/json"
	"github.com/pborman/uuid"
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

type Data struct {
	Provider  string          `json:"provider" bson:"provider"`
	Reference string          `json:"reference" bson:"reference"`
	Scheme    string          `json:"scheme" bson:"scheme"`
	Type      string          `json:"type" bson:"type"`
	Status    string          `json:"status" bson:"status"`
	Value     Value           `json:"value" bson:"value"`
	Date      time.Time       `json:"data" bson:"date"`
	Raw       json.RawMessage `json:"raw" bson:"raw"`
}

type Payment struct {
	Data `bson:",inline"`
	ID   string `json:"id" bson:"_id"`
}

func NewPayment(data Data) Payment {
	return Payment{
		Data: data,
		ID:   uuid.New(),
	}
}
