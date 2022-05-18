package stripe

import (
	payment "github.com/numary/payments/pkg"
	"github.com/stripe/stripe-go/v72"
	"strings"
	"time"
)

func TranslateBalanceTransaction(bt stripe.BalanceTransaction) payment.Payment {
	var (
		reference string
		scheme    string
		status    string
		amount    int64
		asset     string
		date      int64
		ptype     string
	)
	switch bt.Type {
	case "charge":
		reference = bt.Source.ID
		scheme = string(bt.Source.Charge.PaymentMethodDetails.Card.Brand)
		ptype = payment.TypePayIn
		status = string(bt.Source.Charge.Status)
		amount = bt.Source.Charge.Amount
		asset = string(bt.Source.Charge.Currency)
		date = bt.Created
	case "payout":
		reference = bt.Source.ID
		ptype = payment.TypePayout
		scheme = string(bt.Source.Payout.Type)
		status = string(bt.Source.Payout.Status)
		amount = bt.Source.Payout.Amount
		asset = string(bt.Source.Payout.Currency)
		date = bt.Created
	}
	return payment.Payment{
		Identifier: payment.Identifier{
			Provider:  ConnectorName,
			Reference: reference,
			Scheme:    scheme,
			Type:      ptype,
		},
		Data: payment.Data{
			Value: payment.Value{
				Amount: amount,
				Asset:  strings.ToUpper(asset),
			},
			Date:   time.Unix(date, 0),
			Raw:    bt,
			Status: status,
		},
	}
}
