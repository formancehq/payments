package modulr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			models.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get string amount from big int amount %v: %w", pi.Amount, err),
			models.ErrInvalidRequest,
		)
	}

	description := pi.Description

	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			IdempotencyKey:  pi.Reference,
			SourceAccountID: pi.SourceAccount.Reference,
			Destination: client.Destination{
				Type: string(client.DestinationTypeAccount),
				ID:   pi.DestinationAccount.Reference,
			},
			Currency:          curr,
			Amount:            json.Number(amount),
			Reference:         description,
			ExternalReference: description,
		},
	)
	if err != nil {
		return nil, err
	}

	return translateTransferToPayment(resp)
}

func translateTransferToPayment(
	from *client.TransferResponse,
) (*models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	status := matchPaymentStatus(from.Status)

	createdAt, err := time.Parse("2006-01-02T15:04:05.999-0700", from.CreatedDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", from.CreatedDate, err)
	}

	precision, ok := supportedCurrenciesWithDecimal[from.Details.Currency]
	if !ok {
		return nil, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.Details.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount %s: %w", from.Details.Amount, err)
	}

	return &models.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.Details.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &from.Details.SourceAccountID,
		DestinationAccountReference: &from.Details.Destination.ID,
		Raw:                         raw,
	}, nil
}

func matchPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "SUBMITTED", "VALIDATED", "PENDING_FOR_DATE", "PENDING_FOR_FUNDS", "ER_EXTCONN":
		return models.PAYMENT_STATUS_PENDING
	case "PROCESSED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "CANCELLED":
		return models.PAYMENT_STATUS_CANCELLED
	case "ER_EXPIRED":
		return models.PAYMENT_STATUS_EXPIRED
	case "ER_INVALID", "ER_EXTSYS", "ER_GENERAL":
		return models.PAYMENT_STATUS_FAILED
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}
