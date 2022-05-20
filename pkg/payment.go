package payment

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gibson042/canonicaljson-go"
	"time"
)

type Scheme string

const (
	SchemeVisa       Scheme = "visa"
	SchemeMasterCard Scheme = "mastercard"
	SchemeApplePay   Scheme = "apple pay"
	SchemeGooglePay  Scheme = "google pay"
	SchemeSepaDebit  Scheme = "sepa debit"
	SchemeSepaCredit Scheme = "sepa credit"
	SchemeSepa       Scheme = "sepa"
	SchemeA2A        Scheme = "a2a"
	SchemeAchDebit   Scheme = "ach debit"
	SchemeAch        Scheme = "ach"
	SchemeRtp        Scheme = "rtp"
	SchemeOther      Scheme = "other"

	TypePayIn  = "pay-in"
	TypePayout = "payout"
	TypeOther  = "other"

	StatusSucceeded = "succeeded"
)

type Value struct {
	Amount int64  `json:"amount" bson:"amount"`
	Asset  string `json:"asset" bson:"asset"`
}

type Identifier struct {
	Provider  string `json:"provider" bson:"provider"`
	Reference string `json:"reference" bson:"reference"`
	Type      string `json:"type" bson:"type"`
}

func (i Identifier) String() string {
	data, err := canonicaljson.Marshal(i)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(data)
}

func IdentifierFromString(v string) (*Identifier, error) {
	data, err := base64.URLEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	ret := Identifier{}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

type Data struct {
	Scheme Scheme      `json:"scheme" bson:"scheme"`
	Status string      `json:"status" bson:"status"`
	Value  Value       `json:"value" bson:"value"`
	Date   time.Time   `json:"date" bson:"date"`
	Raw    interface{} `json:"raw" bson:"raw"`
}

type Payment struct {
	Identifier `bson:",inline"`
	Data       `bson:",inline"`
}

func (p Payment) MarshalJSON() ([]byte, error) {
	type Aux Payment
	return json.Marshal(struct {
		ID string `json:"id"`
		Aux
	}{
		ID:  p.Identifier.String(),
		Aux: Aux(p),
	})
}
