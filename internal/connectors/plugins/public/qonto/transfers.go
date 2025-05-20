package qonto

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"math/big"
	"regexp"
	"time"
)

/*
*
Note that the reference returned by Qonto is NOT a transaction, and won't be fetched as one.
As such, we need to provide a custom UUID in the transfer reference, to be able to match this payment against
incoming transactions.
*/
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
	transferReference := fmt.Sprintf("transferReference:%s/%s", uuid.New().String(), pi.Reference)
	if len(transferReference) > client.QONTO_MAX_REFERENCE_LENGTH {
		transferReference = transferReference[:client.QONTO_MAX_REFERENCE_LENGTH]
	}

	request := client.TransferRequest{
		SourceIBAN:      pi.SourceAccount.Metadata["bank_account_iban"],
		DestinationIBAN: pi.DestinationAccount.Metadata["bank_account_iban"],
		Reference:       transferReference,
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
	if pi.Amount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("amount is required in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}
	if pi.Asset == "" {
		return errorsutils.NewWrappedError(
			fmt.Errorf("asset is required in transfer/payout request"),
			models.ErrInvalidRequest,
		)
	}
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

	amount, err := transfer.AmountCents.Int64()
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("invalid amount cent for transfer: %w", err)
	}

	if transfer.Status != "processing" && transfer.Status != "pending" && transfer.Status != "completed" {
		return models.PSPPayment{}, errors.Errorf("Unexpected status on newly created transfer: %s", transfer.Status)
	}
	currencyUsed := transfer.Currency
	if currencyUsed == "" {
		currencyUsed = "EUR"
	}

	paymentReference, externalReference, err := parseTransferReference(transfer.Reference)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		ParentReference:             "",
		Reference:                   paymentReference,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      big.NewInt(amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesForInternalAccounts, currencyUsed),
		Scheme:                      models.PAYMENT_SCHEME_SEPA,
		Status:                      models.PAYMENT_STATUS_PENDING,
		SourceAccountReference:      &sourceAccountReference,
		DestinationAccountReference: &destinationAccountReference,
		Metadata: map[string]string{
			"external_reference": externalReference,
			"transfer_id":        transfer.Id,
		},
		Raw: raw,
	}, nil
}

func parseTransferReference(transferReference string) (string, string, error) {
	regex, _ := regexp.Compile("transferReference:([^/]+)/(.+)")
	matches := regex.FindStringSubmatch(transferReference)
	if len(matches) < 3 {
		return "", "", errors.Errorf("Malformed transfer reference: %s", transferReference)
	}
	paymentReference := matches[1]
	err := uuid.Validate(paymentReference)
	if err != nil {
		return "", "", errors.Errorf("Invalid payment reference: %s", paymentReference)
	}

	externalReference := matches[2]
	return paymentReference, externalReference, nil
}
