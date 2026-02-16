package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validateTransferRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	if _, _, err := parseAssetUMN(pi.Asset); err != nil {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to parse asset %s: %w", pi.Asset, err),
			models.ErrInvalidRequest,
		)
	}

	req := &client.TransferRequest{
		IdempotencyKey:       pi.Reference,
		Amount:               pi.Amount.String(),
		Currency:             pi.Asset,
		SourceAccountId:      pi.SourceAccount.Reference,
		DestinationAccountId: pi.DestinationAccount.Reference,
		Description:          &pi.Description,
		Metadata:             pi.Metadata,
	}

	resp, err := p.client.CreateTransfer(ctx, req)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return transferResponseToPayment(resp)
}

func (p *Plugin) validateTransferRequest(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("source account is required"),
			models.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account is required"),
			models.ErrInvalidRequest,
		)
	}

	if pi.Amount == nil || pi.Amount.Cmp(big.NewInt(0)) <= 0 {
		return errorsutils.NewWrappedError(
			fmt.Errorf("amount must be positive"),
			models.ErrInvalidRequest,
		)
	}

	if pi.Reference == "" {
		return errorsutils.NewWrappedError(
			fmt.Errorf("reference is required"),
			models.ErrInvalidRequest,
		)
	}

	return nil
}

func transferResponseToPayment(resp *client.TransferResponse) (models.PSPPayment, error) {
	var amount big.Int
	_, ok := amount.SetString(resp.Amount, 10)
	if !ok {
		return models.PSPPayment{}, fmt.Errorf("failed to parse amount %s as integer", resp.Amount)
	}

	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to parse createdAt: %w", err)
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to marshal raw response: %w", err)
	}

	return models.PSPPayment{
		ParentReference:             resp.IdempotencyKey,
		Reference:                   resp.Id,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      &amount,
		Asset:                       resp.Currency,
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      mapStringToPaymentStatus(resp.Status),
		SourceAccountReference:      &resp.SourceAccountId,
		DestinationAccountReference: &resp.DestinationAccountId,
		Metadata:                    resp.Metadata,
		Raw:                         raw,
	}, nil
}
