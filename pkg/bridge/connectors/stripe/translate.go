package stripe

import (
	payment "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge"
	"github.com/stripe/stripe-go/v72"
	"time"
)

func CreateBatchElement(bt stripe.BalanceTransaction, connectorName string, forward bool) (bridge.BatchElement, bool) {
	var (
		identifier  payment.Identifier
		paymentData *payment.Data
		adjustment  *payment.Adjustment
	)
	switch bt.Type {
	case "charge":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Charge.ID,
			Type:      payment.TypePayIn,
		}
		paymentData = &payment.Data{
			Status:        string(bt.Status),
			InitialAmount: bt.Source.Charge.Amount,
			Asset:         string(bt.Source.Charge.Currency),
			Raw:           bt,
			Scheme:        payment.Scheme(bt.Source.Charge.PaymentMethodDetails.Card.Brand),
			CreatedAt:     time.Unix(bt.Source.Charge.Created, 0),
		}
	case "payout":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Payout.ID,
			Type:      payment.TypePayout,
		}
		paymentData = &payment.Data{
			Status:        string(bt.Status),
			InitialAmount: bt.Source.Payout.Amount,
			Raw:           bt,
			Asset:         string(bt.Source.Payout.Currency),
			Scheme:        "", // TODO
			CreatedAt:     time.Unix(bt.Source.Payout.Created, 0),
		}

	case "transfer":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Transfer.ID,
			Type:      payment.TypePayout,
		}
		paymentData = &payment.Data{
			Status:        string(bt.Status),
			InitialAmount: bt.Source.Transfer.Amount,
			Raw:           bt,
			Asset:         string(bt.Source.Transfer.Currency),
			Scheme:        payment.SchemeSepa, // TODO: Check with clem
			CreatedAt:     time.Unix(bt.Source.Transfer.Created, 0),
		}
	case "refund":
		identifier = payment.Identifier{
			Provider:  connectorName,
			Reference: bt.Source.Refund.Charge.ID,
			Type:      payment.TypePayIn,
		}
		adjustment = &payment.Adjustment{
			Status: string(bt.Status),
			Amount: bt.Source.Refund.Amount,
			Date:   time.Unix(bt.Source.Refund.Created, 0),
			Raw:    bt,
		}
	default:
		return bridge.BatchElement{}, false
	}

	return bridge.BatchElement{
		Identifier: identifier,
		Payment:    paymentData,
		Adjustment: adjustment,
		Forward:    forward,
	}, true
}
