package qonto

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
	"time"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {

	if err := validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesForInternalAccounts, pi.Asset)
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

	request := client.TransferRequest{
		SourceIBAN:      pi.SourceAccount.Metadata["bank_account_iban"],
		DestinationIBAN: pi.DestinationAccount.Metadata["bank_account_iban"],
		Reference:       pi.Reference,
		Currency:        curr,
		Amount:          amount,
	}

	resp, err := p.client.CreateInternalTransfer(ctx, pi.Reference, request)
	if (err != nil) || (resp == nil) {
		return nil, err
	}

	payment, err := transferToPayment(resp, pi.SourceAccount.Reference, pi.DestinationAccount.Reference)

	if err != nil {
		return nil, err
	}
	return &payment, nil
}

func validateTransferPayoutRequests(pi models.PSPPaymentInitiation) error {
	if err := validateAccount(pi.SourceAccount, "source"); err != nil {
		return err
	}
	if err := validateAccount(pi.DestinationAccount, "destination"); err != nil {
		return err
	}
	return nil
}

func validateAccount(account *models.PSPAccount, accountType string) error {
	if account == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("%v account is required in transfer/payout request", accountType),
			models.ErrInvalidRequest,
		)
	}
	if account.Metadata["bank_account_iban"] == "" {
		return errorsutils.NewWrappedError(
			fmt.Errorf("iban is required in %v account", accountType),
			models.ErrInvalidRequest,
		)
	}
	return nil
}

func transferToPayment(transfer *client.TransferResponse, sourceAccountReference, destinationAccountReference string) (models.PSPPayment, error) {

	raw, err := json.Marshal(transfer)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to marshal transfer: %w", err)
	}

	createdAt, err := time.ParseInLocation(client.QONTO_TIMEFORMAT, transfer.CreatedDate, time.UTC)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("invalid time format for transfer: %w", err)
	}

	amount, err := strconv.ParseInt(transfer.AmountCents, 10, 64)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("invalid amount cent for transfer: %w", err)
	}

	if transfer.Status != client.TransactionStatusPending {
		return models.PSPPayment{}, errors.Errorf("Unexpected status on newly created transfer: %s", transfer.Status)
	}

	return models.PSPPayment{
		ParentReference:             "",
		Reference:                   transfer.Id,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      big.NewInt(amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesForInternalAccounts, transfer.Currency),
		Scheme:                      models.PAYMENT_SCHEME_SEPA,
		Status:                      models.PAYMENT_STATUS_PENDING,
		SourceAccountReference:      &sourceAccountReference,
		DestinationAccountReference: &destinationAccountReference,
		Metadata:                    map[string]string{"external_reference": transfer.Reference},
		Raw:                         raw,
	}, nil
}
