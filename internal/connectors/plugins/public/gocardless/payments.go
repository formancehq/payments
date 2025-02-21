package gocardless

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type paymentsState struct {
	After            string    `url:"after,omitempty" json:"after,omitempty"`
	Before           string    `url:"before,omitempty" json:"before,omitempty"`
	LastCreationDate time.Time `json:"LastCreationDate"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState

	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := paymentsState{
		After:            oldState.After,
		Before:           oldState.Before,
		LastCreationDate: oldState.LastCreationDate,
	}

	var payments []models.PSPPayment
	needMore := false
	hasMore := false

	for {

		pagedPayments, nextCursor, err := p.client.GetPayments(
			ctx, client.PaymentPayload{Mandate: ""}, req.PageSize, newState.After, newState.Before,
		)

		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		newState.After = nextCursor.After
		newState.Before = nextCursor.Before

		payments, err = fillPayments(oldState.LastCreationDate, pagedPayments, payments)

		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(payments, pagedPayments, req.PageSize)

		if !needMore || !hasMore {
			break
		}

	}

	if !needMore {
		payments = payments[:req.PageSize]
	}

	if len(payments) > 0 {
		newState.LastCreationDate = payments[len(payments)-1].CreatedAt
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillPayments(
	lastCreatedAt time.Time,
	pagedPayments []client.GocardlessPayment,
	payments []models.PSPPayment,
) ([]models.PSPPayment, error) {

	for _, payment := range pagedPayments {

		createdAt := time.Unix(payment.CreatedAt, 0)

		switch createdAt.Compare(lastCreatedAt) {
		case -1, 0:
			continue
		default:
		}

		payments = append(payments, models.PSPPayment{
			Reference:                   payment.ID,
			Amount:                      big.NewInt(int64(payment.Amount)),
			CreatedAt:                   createdAt,
			Status:                      mapPaymentStatus(payment.Status),
			Asset:                       payment.Asset,
			Metadata:                    payment.Metadata,
			SourceAccountReference:      &payment.SourceAccountReference,
			DestinationAccountReference: &payment.DestinationAccountReference,
		})

	}

	return payments, nil
}

func mapPaymentStatus(gcStatus string) models.PaymentStatus {
	switch gcStatus {
	case "pending":
		return models.PAYMENT_STATUS_PENDING
	case "paid":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "failed":
		return models.PAYMENT_STATUS_FAILED
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}
