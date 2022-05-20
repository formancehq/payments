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
		scheme    payment.Scheme
		status    string
		amount    int64
		asset     string
		date      int64
		ptype     string
	)
	switch bt.Type {
	case "charge":
		reference = bt.Source.ID
		scheme = payment.Scheme(bt.Source.Charge.PaymentMethodDetails.Card.Brand) // TODO: Check with clem
		ptype = payment.TypePayIn
		status = string(bt.Source.Charge.Status)
		amount = bt.Source.Charge.Amount
		asset = string(bt.Source.Charge.Currency)
		date = bt.Created
	case "payout":
		reference = bt.Source.ID
		ptype = payment.TypePayIn
		scheme = payment.Scheme(bt.Source.Charge.PaymentMethodDetails.Card.Brand) // TODO: Check with clem
		status = string(bt.Source.Payout.Status)
		amount = bt.Source.Payout.Amount
		asset = string(bt.Source.Payout.Currency)
		date = bt.Created
	}
	return payment.Payment{
		Identifier: payment.Identifier{
			Provider:  ConnectorName,
			Reference: reference,
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
			Scheme: scheme,
		},
	}
}
