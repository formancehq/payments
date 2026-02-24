package wise

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/wise/client"
	"github.com/formancehq/payments/pkg/connector"
)

type paymentsState struct {
	Offset         int    `json:"offset"`
	LastTransferID uint64 `json:"lastTransferID"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}
	}

	var from client.Profile
	if req.FromPayload == nil {
		return connector.FetchNextPaymentsResponse{}, connector.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	newState := paymentsState{
		Offset: oldState.Offset,
	}

	var payments []connector.PSPPayment
	var paymentIDs []uint64
	needMore := false
	hasMore := false
	for {
		pagedTransfers, err := p.client.GetTransfers(ctx, from.ID, newState.Offset, req.PageSize)
		if err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}

		payments, paymentIDs, err = fillPayments(p.logger, pagedTransfers, payments, paymentIDs, oldState)
		if err != nil {
			return connector.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = connector.ShouldFetchMore(payments, pagedTransfers, req.PageSize)
		if !needMore || !hasMore {
			break
		}

		newState.Offset += req.PageSize
	}

	if !needMore {
		payments = payments[:req.PageSize]
		paymentIDs = paymentIDs[:req.PageSize]

		// Wise is very annoying with that point, the offset must be a multiple
		// of the pageSize, otherwise, we will have an error inconsistent
		// pagination.
		newState.Offset += req.PageSize
	}

	if len(paymentIDs) > 0 {
		newState.LastTransferID = paymentIDs[len(paymentIDs)-1]
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}

	return connector.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillPayments(
	logger logging.Logger,
	pagedTransfers []client.Transfer,
	payments []connector.PSPPayment,
	paymentIDs []uint64,
	oldState paymentsState,
) ([]connector.PSPPayment, []uint64, error) {
	for _, transfer := range pagedTransfers {
		if oldState.LastTransferID != 0 && transfer.ID <= oldState.LastTransferID {
			continue
		}

		payment, err := fromTransferToPayment(transfer)
		if err != nil {
			if errors.Is(err, connector.ErrCurrencyNotSupported) {
				// Do not insert unknown currencies
				logger.WithField("transfer_id", transfer.ID).Info("skipping unsupported wise payment")
				continue
			}
			return nil, nil, err
		}

		payments = append(payments, payment)
		paymentIDs = append(paymentIDs, transfer.ID)
	}

	return payments, paymentIDs, nil
}

func fromTransferToPayment(from client.Transfer) (connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[from.TargetCurrency]
	if !ok {
		return connector.PSPPayment{}, fmt.Errorf("unsupported currency: %s: %w", from.TargetCurrency, connector.ErrCurrencyNotSupported)
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.TargetValue.String(), precision)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	p := connector.PSPPayment{
		Reference: fmt.Sprintf("%d", from.ID),
		CreatedAt: from.CreatedAt,
		Type:      connector.PAYMENT_TYPE_TRANSFER,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, from.TargetCurrency),
		Scheme:    connector.PAYMENT_SCHEME_OTHER,
		Status:    matchTransferStatus(from.Status),
		Raw:       raw,
	}

	if from.SourceBalanceID != 0 {
		p.SourceAccountReference = pointer.For(fmt.Sprintf("%d", from.SourceBalanceID))
	}

	if from.DestinationBalanceID != 0 {
		p.DestinationAccountReference = pointer.For(fmt.Sprintf("%d", from.DestinationBalanceID))
	}

	return p, nil
}

func matchTransferStatus(status string) connector.PaymentStatus {
	switch status {
	case "incoming_payment_waiting", "incoming_payment_initiated", "processing", "funds_converted", "bounced_back":
		return connector.PAYMENT_STATUS_PENDING
	case "outgoing_payment_sent":
		return connector.PAYMENT_STATUS_SUCCEEDED
	case "funds_refunded", "charged_back":
		return connector.PAYMENT_STATUS_FAILED
	case "cancelled":
		return connector.PAYMENT_STATUS_CANCELLED
	}

	return connector.PAYMENT_STATUS_OTHER
}
