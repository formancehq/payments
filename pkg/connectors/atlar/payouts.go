package atlar

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
)

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (string, error) {
	if err := validateTransferPayoutRequest(pi); err != nil {
		return "", err
	}

	currency, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return "", connector.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			connector.ErrInvalidRequest,
		)
	}

	paymentSchemeType := "SCT" // SEPA Credit Transfer
	remittanceInformationType := "UNSTRUCTURED"
	remittanceInformationValue := pi.Description
	amount := atlar_models.AmountInput{
		Currency:    &currency,
		Value:       pi.Amount.Int64(),
		StringValue: amountToString(*pi.Amount, precision),
	}
	date := pi.CreatedAt
	if date.IsZero() {
		date = time.Now()
	}
	dateString := date.Format(time.DateOnly)

	createPaymentRequest := atlar_models.CreatePaymentRequest{
		SourceAccountID:              &pi.SourceAccount.Reference,
		DestinationExternalAccountID: &pi.DestinationAccount.Reference,
		Amount:                       &amount,
		Date:                         &dateString,
		ExternalID:                   pi.Reference,
		PaymentSchemeType:            &paymentSchemeType,
		RemittanceInformation: &atlar_models.RemittanceInformation{
			Type:  &remittanceInformationType,
			Value: &remittanceInformationValue,
		},
	}

	_, err = p.client.PostV1CreditTransfers(ctx, &createPaymentRequest)
	if err != nil {
		return "", err
	}

	return pi.Reference, nil
}

func (p *Plugin) pollPayoutStatus(ctx context.Context, payoutID string) (connector.PollPayoutStatusResponse, error) {
	resp, err := p.client.GetV1CreditTransfersGetByExternalIDExternalID(
		ctx,
		payoutID,
	)
	if err != nil {
		return connector.PollPayoutStatusResponse{}, err
	}

	status := resp.Payload.Status
	// Status docs: https://docs.atlar.com/docs/payment-details#payment-states--events
	switch status {
	case "CREATED", "APPROVED", "PENDING_SUBMISSION", "SENT", "PENDING_AT_BANK", "ACCEPTED", "EXECUTED":
		// By setting both payment and error to nil, the workflow will continue
		// polling until the payment status is either RECONCILED or one of the
		// terminal states.
		return connector.PollPayoutStatusResponse{
			Payment: nil,
			Error:   nil,
		}, nil

	case "RECONCILED":
		// The payment has been reconciled and the funds have been transferred.
		transactionID := resp.Payload.Reconciliation.BookedTransactionID
		payment, err := p.getAtlarTransaction(ctx, transactionID)
		if err != nil {
			return connector.PollPayoutStatusResponse{}, fmt.Errorf("failed to get atlar transaction: %w", err)
		}

		return connector.PollPayoutStatusResponse{
			Payment: payment,
			Error:   nil,
		}, nil

	case "REJECTED", "FAILED", "RETURNED":
		return connector.PollPayoutStatusResponse{
			Error: pointer.For(fmt.Sprintf("payment failed: %s", status)),
		}, nil

	default:
		return connector.PollPayoutStatusResponse{}, fmt.Errorf(
			"unknown status \"%s\" encountered while fetching payment initiation status of payment \"%s\"",
			status, resp.Payload.ID,
		)
	}
}

func (p *Plugin) getAtlarTransaction(ctx context.Context, transactionID string) (*connector.PSPPayment, error) {
	resp, err := p.client.GetV1TransactionsID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get atlar transaction: %w", err)
	}

	payment, err := p.transactionToPayment(ctx, resp.Payload)
	if err != nil {
		return nil, err
	}

	if payment == nil {
		return nil, fmt.Errorf("failed to convert transaction to payment, invalid currency")
	}

	return payment, nil
}
