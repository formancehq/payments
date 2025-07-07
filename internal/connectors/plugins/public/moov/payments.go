package moov

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

type paymentsState struct {
	CompletedSkip     int             `url:"completedSkip,omitempty" json:"completedSkip,omitempty"`
	CompletedTimeline client.Timeline `url:"completedTimeline,omitempty" json:"completedTimeline,omitempty"`

	PendingSkip     int             `url:"pendingSkip,omitempty" json:"pendingSkip,omitempty"`
	PendingTimeline client.Timeline `url:"pendingTimeline,omitempty" json:"pendingTimeline,omitempty"`

	ReversedSkip     int             `url:"reversedSkip,omitempty" json:"reversedSkip,omitempty"`
	ReversedTimeline client.Timeline `url:"reversedTimeline,omitempty" json:"reversedTimeline,omitempty"`

	CreatedSkip     int             `url:"createdSkip,omitempty" json:"createdSkip,omitempty"`
	CreatedTimeline client.Timeline `url:"createdTimeline,omitempty" json:"createdTimeline,omitempty"`

	QueuedSkip     int             `url:"queuedSkip,omitempty" json:"queuedSkip,omitempty"`
	QueuedTimeline client.Timeline `url:"queuedTimeline,omitempty" json:"queuedTimeline,omitempty"`

	CancelledSkip     int             `url:"cancelledSkip,omitempty" json:"cancelledSkip,omitempty"`
	CancelledTimeline client.Timeline `url:"cancelledTimeline,omitempty" json:"cancelledTimeline,omitempty"`

	FailedSkip     int             `url:"failedSkip,omitempty" json:"failedSkip,omitempty"`
	FailedTimeline client.Timeline `url:"failedTimeline,omitempty" json:"failedTimeline,omitempty"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	var from moov.Account
	if req.FromPayload != nil {
		if err := json.Unmarshal(req.FromPayload, &from); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	createdTransfers, createdTimeline, createdHasMore, createdSkip, err := p.client.GetPayments(ctx, from.AccountID, moov.TransferStatus_Created, int(oldState.CreatedSkip), req.PageSize, oldState.CreatedTimeline)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	pendingTransfers, pendingTimeline, pendingHasMore, pendingSkip, err := p.client.GetPayments(ctx, from.AccountID, moov.TransferStatus_Pending, int(oldState.PendingSkip), req.PageSize, oldState.PendingTimeline)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	reversedTransfers, reversedTimeline, reversedHasMore, reversedSkip, err := p.client.GetPayments(ctx, from.AccountID, moov.TransferStatus_Reversed, int(oldState.ReversedSkip), req.PageSize, oldState.ReversedTimeline)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	queuedTransfers, queuedTimeline, queuedHasMore, queuedSkip, err := p.client.GetPayments(ctx, from.AccountID, moov.TransferStatus_Queued, int(oldState.QueuedSkip), req.PageSize, oldState.QueuedTimeline)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	cancelledTransfers, cancelledTimeline, cancelledHasMore, cancelledSkip, err := p.client.GetPayments(ctx, from.AccountID, moov.TransferStatus_Canceled, int(oldState.CancelledSkip), req.PageSize, oldState.CancelledTimeline)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	failedTransfers, failedTimeline, failedHasMore, failedSkip, err := p.client.GetPayments(ctx, from.AccountID, moov.TransferStatus_Failed, int(oldState.FailedSkip), req.PageSize, oldState.FailedTimeline)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	completedTransfers, completedTimeline, completedHasMore, completedSkip, err := p.client.GetPayments(ctx, from.AccountID, moov.TransferStatus_Completed, int(oldState.CompletedSkip), req.PageSize, oldState.CompletedTimeline)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	totalTransfersCount := len(completedTransfers) + len(pendingTransfers) + len(reversedTransfers) + len(createdTransfers) + len(queuedTransfers) + len(cancelledTransfers) + len(failedTransfers)
	transfers := make([]moov.Transfer, 0, totalTransfersCount)

	transfers = append(transfers, completedTransfers...)
	transfers = append(transfers, pendingTransfers...)
	transfers = append(transfers, reversedTransfers...)
	transfers = append(transfers, createdTransfers...)
	transfers = append(transfers, queuedTransfers...)
	transfers = append(transfers, cancelledTransfers...)
	transfers = append(transfers, failedTransfers...)
	payments, err := p.fillPayments(transfers)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	var newState paymentsState

	hasMore := completedHasMore || pendingHasMore || reversedHasMore || createdHasMore || queuedHasMore || cancelledHasMore || failedHasMore
	newState.CompletedSkip = completedSkip
	newState.PendingSkip = pendingSkip
	newState.ReversedSkip = reversedSkip
	newState.CreatedSkip = createdSkip
	newState.QueuedSkip = queuedSkip
	newState.CancelledSkip = cancelledSkip
	newState.FailedSkip = failedSkip

	newState.CompletedTimeline = completedTimeline
	newState.PendingTimeline = pendingTimeline
	newState.ReversedTimeline = reversedTimeline
	newState.CreatedTimeline = createdTimeline
	newState.QueuedTimeline = queuedTimeline
	newState.CancelledTimeline = cancelledTimeline
	newState.FailedTimeline = failedTimeline
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		HasMore:  hasMore,
		NewState: payload,
		Payments: payments,
	}, nil
}

func (p *Plugin) fillPayments(transfers []moov.Transfer) ([]models.PSPPayment, error) {

	payments := make([]models.PSPPayment, 0, len(transfers))
	for _, transfer := range transfers {

		raw, err := json.Marshal(transfer)
		if err != nil {
			return []models.PSPPayment{}, err
		}

		paymentType := mapPaymentType(transfer)

		status := mapStatus(transfer.Status)

		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.Amount.Currency)

		metadata := mapPaymentMetadata(transfer)

		payments = append(payments, models.PSPPayment{
			Reference:                   transfer.TransferID,
			Amount:                      big.NewInt(transfer.Amount.Value),
			Asset:                       asset,
			Status:                      status,
			CreatedAt:                   transfer.CreatedOn,
			Type:                        paymentType,
			SourceAccountReference:      extractSourceAccountReference(transfer.Source),
			DestinationAccountReference: extractDestinationAccountReference(transfer.Destination),
			Metadata:                    metadata,
			Raw:                         raw,
		})
	}

	return payments, nil
}
