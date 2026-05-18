package universal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.CreateTransferResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_CREATE_TRANSFER); err != nil {
		return models.CreateTransferResponse{}, err
	}

	pi := req.PaymentInitiation
	if err := validatePaymentInitiation(pi); err != nil {
		return models.CreateTransferResponse{}, err
	}

	resp, err := p.client.CreateTransfer(ctx, pi.Reference, &client.TransferRequest{
		Reference:                   pi.Reference,
		Description:                 pi.Description,
		Amount:                      pi.Amount.String(),
		Asset:                       pi.Asset,
		SourceAccountReference:      pi.SourceAccount.Reference,
		DestinationAccountReference: pi.DestinationAccount.Reference,
		Metadata:                    pi.Metadata,
	})
	if err != nil {
		return models.CreateTransferResponse{}, fmt.Errorf("create transfer %s: %w", pi.Reference, err)
	}
	return interpretInitiationResponse[models.CreateTransferResponse](resp.Mode, resp.PollingID, resp.Payment, resp.Error,
		func(payment *models.PSPPayment, polling *string) models.CreateTransferResponse {
			return models.CreateTransferResponse{Payment: payment, PollingTransferID: polling}
		})
}

func (p *Plugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.ReverseTransferResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_CREATE_TRANSFER); err != nil {
		return models.ReverseTransferResponse{}, err
	}

	rev := req.PaymentInitiationReversal
	if err := validateReversal(rev); err != nil {
		return models.ReverseTransferResponse{}, err
	}

	resp, err := p.client.ReverseTransfer(ctx, rev.Reference, rev.RelatedPaymentInitiation.Reference, &client.ReverseRequest{
		Reference:   rev.Reference,
		Description: rev.Description,
		Amount:      rev.Amount.String(),
		Asset:       rev.Asset,
		Metadata:    rev.Metadata,
	})
	if err != nil {
		return models.ReverseTransferResponse{}, fmt.Errorf("reverse transfer %s: %w", rev.Reference, err)
	}
	if resp.Payment == nil {
		return models.ReverseTransferResponse{}, fmt.Errorf("counterparty did not return a payment for transfer reverse %s", rev.Reference)
	}
	pp, err := mappers.PaymentToPSPPayment(*resp.Payment)
	if err != nil {
		return models.ReverseTransferResponse{}, err
	}
	return models.ReverseTransferResponse{Payment: pp}, nil
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.PollTransferStatusResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_CREATE_TRANSFER); err != nil {
		return models.PollTransferStatusResponse{}, err
	}
	id := strings.TrimSpace(req.TransferID)
	if id == "" {
		return models.PollTransferStatusResponse{}, errorsutils.NewWrappedError(errors.New("PollTransferStatus requires a non-empty TransferID"), models.ErrInvalidRequest)
	}
	resp, err := p.client.GetTransfer(ctx, id)
	if err != nil {
		return models.PollTransferStatusResponse{}, fmt.Errorf("get transfer %s: %w", id, err)
	}
	return pollResp[models.PollTransferStatusResponse](resp.Payment, resp.Error,
		func(payment *models.PSPPayment, errStr *string) models.PollTransferStatusResponse {
			return models.PollTransferStatusResponse{Payment: payment, Error: errStr}
		})
}
