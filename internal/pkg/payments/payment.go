package payments

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/gibson042/canonicaljson-go"
)

type (
	Scheme string
	Status string
)

const (
	SchemeUnknown Scheme = "unknown"
	SchemeOther   Scheme = "other"

	SchemeCardVisa       Scheme = "visa"
	SchemeCardMasterCard Scheme = "mastercard"
	SchemeCardAmex       Scheme = "amex"
	SchemeCardDiners     Scheme = "diners"
	SchemeCardDiscover   Scheme = "discover"
	SchemeCardJCB        Scheme = "jcb"
	SchemeCardUnionPay   Scheme = "unionpay"

	SchemeSepaDebit  Scheme = "sepa debit"
	SchemeSepaCredit Scheme = "sepa credit"
	SchemeSepa       Scheme = "sepa"

	SchemeApplePay  Scheme = "apple pay"
	SchemeGooglePay Scheme = "google pay"

	SchemeA2A      Scheme = "a2a"
	SchemeACHDebit Scheme = "ach debit"
	SchemeACH      Scheme = "ach"
	SchemeRTP      Scheme = "rtp"

	TypePayIn    = "pay-in"
	TypePayout   = "payout"
	TypeTransfer = "transfer"
	TypeOther    = "other"

	StatusSucceeded Status = "succeeded"
	StatusCancelled Status = "cancelled"
	StatusFailed    Status = "failed"
	StatusPending   Status = "pending"
	StatusOther     Status = "other"
)

type Referenced struct {
	Reference string `json:"reference" bson:"reference"`
	Type      string `json:"type" bson:"type"`
}

type Identifier struct {
	Referenced `bson:",inline"`
	Provider   string `json:"provider" bson:"provider"`
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
	Status   Status      `json:"status" bson:"status"`
	Amount   int64       `json:"amount" bson:"amount"`
	Date     time.Time   `json:"date" bson:"date"`
	Raw      interface{} `json:"raw" bson:"raw"`
	Absolute bool        `json:"absolute" bson:"absolute"`
}

type Data struct {
	Status        Status      `json:"status" bson:"status"`
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

func (p Payment) HasInitialValue() bool {
	return p.InitialAmount != 0
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

func (p Payment) Computed() SavedPayment {
	aggregatedAdjustmentValue := int64(0)
	amount := int64(0)

	for i := 0; i < len(p.Adjustments)-1; i++ {
		adjustment := p.Adjustments[i]
		if adjustment.Absolute {
			amount = adjustment.Amount

			break
		}

		aggregatedAdjustmentValue += adjustment.Amount
	}

	if amount == 0 {
		amount = p.InitialAmount + aggregatedAdjustmentValue
	}

	return SavedPayment{
		Identifier:  p.Identifier,
		Data:        p.Data,
		Amount:      amount,
		Adjustments: p.Adjustments,
	}
}

type SavedPayment struct {
	Identifier
	Data
	Amount      int64        `json:"amount"`
	Adjustments []Adjustment `json:"adjustments"`
}

func (p SavedPayment) MarshalJSON() ([]byte, error) {
	type Aux SavedPayment

	return json.Marshal(struct {
		ID string `json:"id"`
		Aux
	}{
		ID:  p.Identifier.String(),
		Aux: Aux(p),
	})
}