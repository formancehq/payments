package universal

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.CreatePayoutResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_CREATE_PAYOUT); err != nil {
		return models.CreatePayoutResponse{}, err
	}

	pi := req.PaymentInitiation
	if err := validatePaymentInitiation(pi); err != nil {
		return models.CreatePayoutResponse{}, err
	}

	wire := &client.PayoutRequest{
		Reference:                   pi.Reference,
		Description:                 pi.Description,
		Amount:                      pi.Amount.String(),
		Asset:                       pi.Asset,
		SourceAccountReference:      pi.SourceAccount.Reference,
		DestinationAccountReference: pi.DestinationAccount.Reference,
		Metadata:                    pi.Metadata,
	}
	resp, err := p.client.CreatePayout(ctx, pi.Reference, wire)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("creating payout %s: %w", pi.Reference, err)
	}
	return interpretInitiationResponse[models.CreatePayoutResponse](resp.Mode, resp.PollingID, resp.Payment, resp.Error,
		func(payment *models.PSPPayment, polling *string) models.CreatePayoutResponse {
			return models.CreatePayoutResponse{Payment: payment, PollingPayoutID: polling}
		})
}

func (p *Plugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.ReversePayoutResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_CREATE_PAYOUT); err != nil {
		return models.ReversePayoutResponse{}, err
	}

	rev := req.PaymentInitiationReversal
	if err := validateReversal(rev); err != nil {
		return models.ReversePayoutResponse{}, err
	}

	resp, err := p.client.ReversePayout(ctx, rev.Reference, rev.RelatedPaymentInitiation.Reference, &client.ReverseRequest{
		Reference:   rev.Reference,
		Description: rev.Description,
		Amount:      rev.Amount.String(),
		Asset:       rev.Asset,
		Metadata:    rev.Metadata,
	})
	if err != nil {
		return models.ReversePayoutResponse{}, fmt.Errorf("reversing payout %s: %w", rev.Reference, err)
	}
	if resp.Payment == nil {
		return models.ReversePayoutResponse{}, fmt.Errorf("counterparty did not return a payment for payout reverse %s", rev.Reference)
	}
	pp, err := mappers.PaymentToPSPPayment(*resp.Payment)
	if err != nil {
		return models.ReversePayoutResponse{}, err
	}
	return models.ReversePayoutResponse{Payment: pp}, nil
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.PollPayoutStatusResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_CREATE_PAYOUT); err != nil {
		return models.PollPayoutStatusResponse{}, err
	}
	resp, err := p.client.GetPayout(ctx, req.PayoutID)
	if err != nil {
		return models.PollPayoutStatusResponse{}, err
	}
	return pollResp[models.PollPayoutStatusResponse](resp.Payment, resp.Error,
		func(payment *models.PSPPayment, errStr *string) models.PollPayoutStatusResponse {
			return models.PollPayoutStatusResponse{Payment: payment, Error: errStr}
		})
}

// interpretInitiationResponse turns the contract's terminal-or-polling envelope
// into the engine's ResponseT shape. If Mode is "polling" the engine schedules
// PollPayoutStatus / PollTransferStatus via Temporal until terminal. The
// "terminal" branch must include a Payment; an explicit Error string is also
// honoured for synchronous failures.
func interpretInitiationResponse[ResponseT any](
	mode string,
	pollingID string,
	payment *client.Payment,
	errStr string,
	build func(*models.PSPPayment, *string) ResponseT,
) (ResponseT, error) {
	var zero ResponseT
	if errStr != "" {
		return zero, errorsutils.NewWrappedError(errors.New(errStr), models.ErrInvalidRequest)
	}
	switch mode {
	case "polling":
		if pollingID == "" {
			return zero, errors.New("polling response missing pollingID")
		}
		id := pollingID
		return build(nil, &id), nil
	case "terminal", "":
		if payment == nil {
			return zero, errors.New("terminal response missing payment")
		}
		pp, err := mappers.PaymentToPSPPayment(*payment)
		if err != nil {
			return zero, err
		}
		return build(&pp, nil), nil
	default:
		return zero, fmt.Errorf("unknown response mode %q", mode)
	}
}

// pollResp normalises a poll result into the engine's PollResponse contract.
// Both nil → keep polling; payment set → terminal success; error set →
// terminal failure (engine writes the error onto the payment-initiation
// adjustment trail).
func pollResp[ResponseT any](
	payment *client.Payment,
	errStr string,
	build func(*models.PSPPayment, *string) ResponseT,
) (ResponseT, error) {
	var zero ResponseT
	if errStr != "" {
		s := errStr
		return build(nil, &s), nil
	}
	if payment == nil {
		return build(nil, nil), nil
	}
	pp, err := mappers.PaymentToPSPPayment(*payment)
	if err != nil {
		return zero, err
	}
	return build(&pp, nil), nil
}

func validatePaymentInitiation(pi models.PSPPaymentInitiation) error {
	if pi.Reference == "" {
		return errorsutils.NewWrappedError(errors.New("missing reference"), models.ErrInvalidRequest)
	}
	if pi.Amount == nil || pi.Amount.Cmp(big.NewInt(0)) <= 0 {
		return errorsutils.NewWrappedError(errors.New("amount must be positive"), models.ErrInvalidRequest)
	}
	if pi.SourceAccount == nil {
		return errorsutils.NewWrappedError(errors.New("missing source account"), models.ErrInvalidRequest)
	}
	if pi.DestinationAccount == nil {
		return errorsutils.NewWrappedError(errors.New("missing destination account"), models.ErrInvalidRequest)
	}
	if pi.Asset == "" {
		return errorsutils.NewWrappedError(errors.New("missing asset"), models.ErrInvalidRequest)
	}
	return nil
}

func validateReversal(rev models.PSPPaymentInitiationReversal) error {
	if rev.Reference == "" {
		return errorsutils.NewWrappedError(errors.New("missing reversal reference"), models.ErrInvalidRequest)
	}
	if rev.RelatedPaymentInitiation.Reference == "" {
		return errorsutils.NewWrappedError(errors.New("missing related payment-initiation reference"), models.ErrInvalidRequest)
	}
	if rev.Amount == nil || rev.Amount.Cmp(big.NewInt(0)) <= 0 {
		return errorsutils.NewWrappedError(errors.New("reversal amount must be positive"), models.ErrInvalidRequest)
	}
	return nil
}
