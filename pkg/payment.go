package payments

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gibson042/canonicaljson-go"
	"go.mongodb.org/mongo-driver/bson"
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

type Adjustment struct {
	Status   string      `json:"status" bson:"status"`
	Amount   int64       `json:"amount" bson:"amount"`
	Date     time.Time   `json:"date" bson:"date"`
	Raw      interface{} `json:"raw" bson:"raw"`
	Absolute bool        `json:"absolute" bson:"absolute"`
}

type Data struct {
	Status        string      `json:"status" bson:"status"`
	InitialAmount int64       `json:"initialAmount" bson:"initialAmount"`
	Scheme        Scheme      `json:"scheme" bson:"scheme"`
	Asset         string      `json:"asset" bson:"asset"`
	CreatedAt     time.Time   `json:"createdAt" bson:"createdAt"`
	Raw           interface{} `json:"raw" bson:"raw"`
}

type Payment struct {
	Identifier  `bson:",inline"`
	Data        `bson:",inline"`
	Adjustments []Adjustment `json:"adjustments" bson:"adjustments"`
}

func (p *Payment) UnmarshalBSON(bytes []byte) error {
	type Aux Payment
	type WithRaw struct {
		Aux
		Raw map[string]interface{} `bson:"raw"`
	}
	wr := WithRaw{}
	err := bson.Unmarshal(bytes, &wr)
	if err != nil {
		return err
	}

	*p = Payment(wr.Aux)
	p.Raw = wr.Raw
	return nil
}

var _ bson.Unmarshaler = &Payment{}

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

func (p Payment) Computed() ComputedPayment {

	aggregatedAdjustmentValue := p.InitialAmount
	amount := int64(0)
	for _, a := range p.Adjustments {
		if a.Absolute {
			amount = a.Amount
			break
		}

		aggregatedAdjustmentValue += a.Amount
	}
	if amount == 0 {
		amount = p.InitialAmount + aggregatedAdjustmentValue
	}

	return ComputedPayment{
		Identifier:  p.Identifier,
		Data:        p.Data,
		Amount:      amount,
		Adjustments: p.Adjustments,
	}
}

type ComputedPayment struct {
	Identifier
	Data
	Amount      int64        `json:"amount"`
	Adjustments []Adjustment `json:"adjustments"`
}

func (p ComputedPayment) MarshalJSON() ([]byte, error) {
	type Aux ComputedPayment
	return json.Marshal(struct {
		ID string `json:"id"`
		Aux
	}{
		ID:  p.Identifier.String(),
		Aux: Aux(p),
	})
}
