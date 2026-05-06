package routable

import (
	"context"
	"errors"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// createPayout maps a PSPPaymentInitiation onto a Routable payable. The
// engine keeps polling via PollPayoutStatus when we return PollingPayoutID,
// which is how we honour Routable's async 202 contract without blocking.
func (p *Plugin) createPayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	pi := req.PaymentInitiation
	if err := validatePaymentInitiation(pi); err != nil {
		return models.CreatePayoutResponse{}, err
	}

	payable, err := p.initiatePayable(ctx, pi)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	payment, err := p.payableToPSPPayment(*payable)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("mapping payable response: %w", err)
	}

	if isTerminalStatus(payment.Status) {
		return models.CreatePayoutResponse{Payment: &payment}, nil
	}
	id := payable.ID
	return models.CreatePayoutResponse{PollingPayoutID: &id}, nil
}

// pollPayableStatus is shared by PollPayoutStatus and PollTransferStatus.
// Both surfaces map onto the same Routable payable, so we keep a single
// implementation and dispatch from the plugin shim.
func (p *Plugin) pollPayableStatus(ctx context.Context, payableID string) (models.PollPayoutStatusResponse, error) {
	if payableID == "" {
		return models.PollPayoutStatusResponse{}, errors.New("missing payable id")
	}

	pa, err := p.client.GetPayable(ctx, payableID)
	if err != nil {
		// Eventual consistency: Routable can return 404 right after a 202.
		// Telling the engine "not yet" causes it to retry on schedule.
		if errors.Is(err, client.ErrPayableNotFound) {
			return models.PollPayoutStatusResponse{}, nil
		}
		return models.PollPayoutStatusResponse{}, fmt.Errorf("polling payable %s: %w", payableID, err)
	}

	payment, err := p.payableToPSPPayment(*pa)
	if err != nil {
		return models.PollPayoutStatusResponse{}, fmt.Errorf("mapping payable: %w", err)
	}

	if !isTerminalStatus(payment.Status) {
		return models.PollPayoutStatusResponse{}, nil
	}

	if payment.Status == models.PAYMENT_STATUS_FAILED ||
		payment.Status == models.PAYMENT_STATUS_CANCELLED ||
		payment.Status == models.PAYMENT_STATUS_EXPIRED {
		errMsg := fmt.Sprintf("routable payable %s ended in %s", payableID, payment.Status)
		return models.PollPayoutStatusResponse{Error: &errMsg}, nil
	}

	return models.PollPayoutStatusResponse{Payment: &payment}, nil
}

// initiatePayable is the shared CreatePayable plumbing used by both
// CreateTransfer and CreatePayout. Both flows produce a Routable payable;
// the only thing that differs at the call site is which engine response
// envelope wraps the result.
func (p *Plugin) initiatePayable(ctx context.Context, pi models.PSPPaymentInitiation) (*client.Payable, error) {
	if pi.SourceAccount == nil || pi.SourceAccount.Reference == "" {
		return nil, errors.New("missing source account reference")
	}
	if pi.DestinationAccount == nil || pi.DestinationAccount.Reference == "" {
		return nil, errors.New("missing destination account reference")
	}

	currencyCode, _, err := splitAsset(pi.Asset)
	if err != nil {
		return nil, fmt.Errorf("invalid asset %q: %w", pi.Asset, err)
	}
	precision, err := precisionFor(currencyCode)
	if err != nil {
		return nil, err
	}
	amount := fromMinorUnits(pi.Amount, precision)

	req := client.CreatePayableRequest{
		Type:                fieldOr(pi.Metadata, MetadataKeyType, defaultPayableType),
		DeliveryMethod:      fieldOr(pi.Metadata, MetadataKeyDeliveryMethod, defaultDeliveryMethod),
		PayToCompany:        pi.DestinationAccount.Reference,
		WithdrawFromAccount: pi.SourceAccount.Reference,
		Amount:              amount,
		CurrencyCode:        currencyCode,
		LineItems: []client.PayableLineItem{{
			UnitPrice:   amount,
			Amount:      amount,
			Quantity:    1,
			Description: fieldOr(pi.Metadata, MetadataKeyLineDescription, pi.Description),
		}},
		ActingTeamMember: fieldOr(pi.Metadata, MetadataKeyActingTeamMember, p.config.ActingTeamMember),
		Reference:        pi.Reference,
		ExternalID:       pi.Metadata[MetadataKeyExternalID],
		Memo:             fieldOr(pi.Metadata, MetadataKeyMemo, pi.Description),
		IdempotencyKey:   pi.Reference,
	}

	p.logger.Infof("initiating routable payable: type=%s delivery=%s amount=%s %s reference=%s", req.Type, req.DeliveryMethod, req.Amount, req.CurrencyCode, req.Reference)
	return p.client.CreatePayable(ctx, req)
}

func validatePaymentInitiation(pi models.PSPPaymentInitiation) error {
	if pi.Reference == "" {
		return errors.New("missing payment initiation reference")
	}
	if pi.Amount == nil {
		return errors.New("missing payment initiation amount")
	}
	if pi.Asset == "" {
		return errors.New("missing payment initiation asset")
	}
	return nil
}

// splitAsset splits a Formance asset string ("USD/2") into its currency
// code and precision parts. We accept both the prefixed form and the bare
// currency code so PSPPaymentInitiation values from older callers still
// work.
func splitAsset(asset string) (string, int, error) {
	for i := 0; i < len(asset); i++ {
		if asset[i] == '/' {
			code := asset[:i]
			precision, err := precisionFor(code)
			if err != nil {
				return "", 0, err
			}
			return code, precision, nil
		}
	}
	precision, err := precisionFor(asset)
	if err != nil {
		return "", 0, err
	}
	return asset, precision, nil
}
