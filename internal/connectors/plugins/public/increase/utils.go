package increase

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validatePayoutRequests(pi models.PSPPaymentInitiation) error {
	payoutMethod := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey)
	if payoutMethod == "" {
		return fmt.Errorf("payoutMethod is a required metadata: %w", models.ErrInvalidRequest)
	}

	validMethods := map[string]bool{
		increaseACHPayoutMethod:    true,
		increaseWirePaymentMethod:  true,
		increaseCheckPaymentMethod: true,
		increaseRTPPaymentMethod:   true,
	}

	if !validMethods[payoutMethod] {
		return fmt.Errorf("payoutMethod must be one of: ach, wire, check, rtp: %w", models.ErrInvalidRequest)
	}

	if pi.Description == "" {
		return fmt.Errorf("description is required: %w", models.ErrInvalidRequest)
	}

	if pi.Amount == nil {
		return fmt.Errorf("amount is required: %w", models.ErrInvalidRequest)
	}

	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	if payoutMethod == increaseCheckPaymentMethod && models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseFufillmentMethodMetadataKey) == "" {
		return fmt.Errorf("fulfillmentMethod is a required metadata: %w", models.ErrInvalidRequest)
	}

	sourceAccountNumberID := models.ExtractNamespacedMetadata(pi.SourceAccount.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
	if sourceAccountNumberID == "" && (payoutMethod == increaseCheckPaymentMethod || payoutMethod == increaseRTPPaymentMethod) {
		return fmt.Errorf("sourceAccountNumberID is a required source account metadata: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) validateTransferRequests(pi models.PSPPaymentInitiation) error {
	if pi.Amount == nil {
		return fmt.Errorf("amount is required: %w", models.ErrInvalidRequest)
	}

	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	if pi.Description == "" {
		return fmt.Errorf("description is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) validateBankAccountRequests(ba models.BankAccount) error {
	routingNumber := models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseRoutingNumberMetadataKey)
	if routingNumber == "" {
		return fmt.Errorf("missing routingNumber in bank account metadata: %w", models.ErrInvalidRequest)
	}

	accountHolder := models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseAccountHolderMetadataKey)
	if accountHolder == "" {
		return fmt.Errorf("missing accountHolder in bank account metadata: %w", models.ErrInvalidRequest)
	}

	description := models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseDescriptionMetadataKey)
	if description == "" {
		return fmt.Errorf("missing description in bank account metadata: %w", models.ErrInvalidRequest)
	}

	if ba.AccountNumber == nil {
		return fmt.Errorf("missing accountNumber in bank account request: %w", models.ErrInvalidRequest)
	}

	return nil
}
