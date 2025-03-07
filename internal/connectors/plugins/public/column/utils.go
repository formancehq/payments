package column

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validateTransferRequest(pi models.PSPPaymentInitiation) error {
	if pi.Amount == nil {
		return fmt.Errorf("required field amount must be provided")
	}

	if pi.Asset == "" {
		return fmt.Errorf("required field asset must be provided")
	}

	if pi.Metadata == nil {
		return fmt.Errorf("required field metadata must be provided")
	}

	if pi.SourceAccount == nil {
		return fmt.Errorf("required field sourceAccount is missing")
	}

	if pi.SourceAccount.Name == nil {
		return fmt.Errorf("required sourceAccount field name is missing")
	}

	if pi.SourceAccount.Reference == "" {
		return fmt.Errorf("required sourceAccount field reference is missing")
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("required field destinationAccount must be provided")
	}

	if pi.DestinationAccount.Reference == "" {
		return fmt.Errorf("required destinationAccount field reference is missing")
	}

	allowOverdraft := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnAllowOverdraftMetadataKey)
	if allowOverdraft != "" {
		if allowOverdraft != "true" && allowOverdraft != "false" {
			return fmt.Errorf("required field allow overdraft must be provided")
		}
	}

	hold := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnHoldMetadataKey)
	if hold != "" {
		if hold != "true" && hold != "false" {
			return fmt.Errorf("required field hold must be provided")
		}
	}

	err := validateAddress(pi.Metadata)

	if err != nil {
		return err
	}

	addressLine1 := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnAddressLine1MetadataKey)

	if addressLine1 != "" {
		countryCode := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnCountryCodeMetadataKey)

		if countryCode == "" {
			return fmt.Errorf("required metadata field %s is missing", client.ColumnCountryCodeMetadataKey)
		}

	}

	if addressLine1 == "" && models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnCountryCodeMetadataKey) != "" {
		return fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnCountryCodeMetadataKey)
	}

	return nil
}

func validateAddress(addressMetadata map[string]string) error {
	addressLine1 := models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressLine1MetadataKey)

	if addressLine1 != "" {
		city := models.ExtractNamespacedMetadata(addressMetadata, client.ColumnCityMetadataKey)

		if city == "" {
			return fmt.Errorf("required metadata field %s is missing", client.ColumnCityMetadataKey)
		}
	}

	if addressLine1 == "" && models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressLine1MetadataKey) != "" {
		return fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressLine1MetadataKey)
	}

	if addressLine1 == "" && models.ExtractNamespacedMetadata(addressMetadata, client.ColumnCityMetadataKey) != "" {
		return fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnCityMetadataKey)
	}

	if addressLine1 == "" && models.ExtractNamespacedMetadata(addressMetadata, client.ColumnStateMetadataKey) != "" {
		return fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnStateMetadataKey)
	}

	if addressLine1 == "" && models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressPostalCodeMetadataKey) != "" {
		return fmt.Errorf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressPostalCodeMetadataKey)
	}

	return nil
}

func (p *Plugin) validatePayoutRequests(pi models.PSPPaymentInitiation) error {

	if pi.Amount == nil {
		return fmt.Errorf("required field amount must be provided")
	}

	if pi.SourceAccount == nil {
		return fmt.Errorf("required field sourceAccount must be provided")
	}

	if pi.SourceAccount.Reference == "" {
		return fmt.Errorf("required field sourceAccount.reference must be provided")
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("required field destinationAccount must be provided")
	}

	if pi.DestinationAccount.Reference == "" {
		return fmt.Errorf("required field destinationAccount.reference must be provided")
	}

	if pi.Metadata == nil {
		return fmt.Errorf("required field metadata must be provided")
	}

	payoutType := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnPayoutTypeMetadataKey)

	if payoutType == "" {
		return fmt.Errorf("required field metadata field %s must be provided", client.ColumnPayoutTypeMetadataKey)
	}

	if payoutType != "ach" && payoutType != "wire" && payoutType != "realtime" && payoutType != "international-wire" {
		return fmt.Errorf("required field metadata field %s must be one of: ach, wire, realtime, international-wire", client.ColumnPayoutTypeMetadataKey)
	}

	if pi.Asset == "" {
		return fmt.Errorf("required field asset must be provided")
	}

	return nil
}

func (p *Plugin) validateReversePayout(pr models.PSPPaymentInitiationReversal) error {
	if pr.Metadata == nil {
		return fmt.Errorf("required field metadata must be provided")
	}

	reason := models.ExtractNamespacedMetadata(pr.Metadata, client.ColumnReasonMetadataKey)
	if reason == "" {
		return fmt.Errorf("required field metadata field %s must be provided", client.ColumnReasonMetadataKey)
	}

	if !IsValidReversePayoutReason(reason) {
		return fmt.Errorf("required field metadata field %s must be a valid reason", client.ColumnReasonMetadataKey)
	}

	if pr.RelatedPaymentInitiation.Reference == "" {
		return fmt.Errorf("required field relatedPaymentInitiation.reference must be provided")
	}

	return nil
}

func validateExternalBankAccount(newExternalBankAccount models.BankAccount) error {
	if newExternalBankAccount.AccountNumber == nil {
		return fmt.Errorf("account number is required")
	}

	routingNumber := models.ExtractNamespacedMetadata(newExternalBankAccount.Metadata, client.ColumnRoutingNumberMetadataKey)

	if routingNumber == "" {
		return fmt.Errorf("required metadata field %s is missing", client.ColumnRoutingNumberMetadataKey)
	}

	err := validateAddress(newExternalBankAccount.Metadata)

	if err != nil {
		return err
	}

	addressLine1 := models.ExtractNamespacedMetadata(newExternalBankAccount.Metadata, client.ColumnAddressLine1MetadataKey)

	if addressLine1 != "" {

		if newExternalBankAccount.Country == nil {
			return fmt.Errorf("country is required")
		}

	}

	if addressLine1 == "" && newExternalBankAccount.Country != nil {
		return fmt.Errorf("metadata field country is not required when addressLine1 is not provided")
	}

	return nil
}

func ParseColumnTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func matchStatus(status string) models.PaymentStatus {
	switch status {
	case "REJECTED":
		return models.PAYMENT_STATUS_FAILED
	case "COMPLETED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "HOLD":
		return models.PAYMENT_STATUS_PENDING
	case "CANCELED":
		return models.PAYMENT_STATUS_CANCELLED
	}

	return models.PAYMENT_STATUS_UNKNOWN
}

func IsValidReversePayoutReason(reason string) bool {
	switch client.ReversePayoutReason(reason) {
	case client.ReasonDuplicatedEntry,
		client.ReasonIncorrectAmount,
		client.ReasonIncorrectReceiverAccount,
		client.ReasonDebitEarlierThanIntended,
		client.ReasonCreditLaterThanIntended:
		return true
	}
	return false
}
