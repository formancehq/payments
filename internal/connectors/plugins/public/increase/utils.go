package increase

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validatePayoutRequests(pi models.PSPPaymentInitiation) error {
	payoutMethod := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey)
	if payoutMethod == "" {
		return models.NewConnectorValidationError(client.IncreasePayoutMethodMetadataKey, models.ErrMissingConnectorMetadata)
	}

	validMethods := map[string]bool{
		increaseACHPayoutMethod:    true,
		increaseWirePaymentMethod:  true,
		increaseCheckPaymentMethod: true,
		increaseRTPPaymentMethod:   true,
	}

	if !validMethods[payoutMethod] {
		return models.NewConnectorValidationError(client.IncreasePayoutMethodMetadataKey, models.ErrInvalidRequest)
	}

	if pi.Description == "" {
		return models.NewConnectorValidationError("description", models.ErrMissingConnectorField)
	}

	if pi.Amount == nil {
		return models.NewConnectorValidationError("amount", models.ErrMissingConnectorField)
	}

	if pi.SourceAccount == nil {
		return models.NewConnectorValidationError("sourceAccount", models.ErrMissingConnectorField)
	}

	if pi.DestinationAccount == nil {
		return models.NewConnectorValidationError("destinationAccount", models.ErrMissingConnectorField)
	}

	if payoutMethod == increaseCheckPaymentMethod && models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseFulfillmentMethodMetadataKey) == "" {
		return models.NewConnectorValidationError(client.IncreaseFulfillmentMethodMetadataKey, models.ErrMissingConnectorMetadata)
	}

	sourceAccountNumberID := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
	if sourceAccountNumberID == "" && (payoutMethod == increaseCheckPaymentMethod || payoutMethod == increaseRTPPaymentMethod) {
		return models.NewConnectorValidationError(client.IncreaseSourceAccountNumberIdMetadataKey, models.ErrMissingConnectorMetadata)
	}

	return nil
}

func (p *Plugin) validateTransferRequests(pi models.PSPPaymentInitiation) error {
	if pi.Amount == nil {
		return models.NewConnectorValidationError("amount", models.ErrMissingConnectorField)
	}

	if pi.SourceAccount == nil {
		return models.NewConnectorValidationError("sourceAccount", models.ErrMissingConnectorField)
	}

	if pi.DestinationAccount == nil {
		return models.NewConnectorValidationError("destinationAccount", models.ErrMissingConnectorField)
	}

	if pi.Description == "" {
		return models.NewConnectorValidationError("description", models.ErrMissingConnectorField)
	}

	return nil
}

func (p *Plugin) validateBankAccountRequests(ba models.BankAccount) error {
	routingNumber := models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseRoutingNumberMetadataKey)
	if routingNumber == "" {
		return models.NewConnectorValidationError(client.IncreaseRoutingNumberMetadataKey, models.ErrMissingConnectorMetadata)
	}

	accountHolder := models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseAccountHolderMetadataKey)
	if accountHolder == "" {
		return models.NewConnectorValidationError(client.IncreaseAccountHolderMetadataKey, models.ErrMissingConnectorMetadata)
	}

	description := models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseDescriptionMetadataKey)
	if description == "" {
		return models.NewConnectorValidationError(client.IncreaseDescriptionMetadataKey, models.ErrMissingConnectorMetadata)
	}

	if ba.AccountNumber == nil {
		return models.NewConnectorValidationError("AccountNumber", models.ErrMissingConnectorField)
	}

	return nil
}

func (p *Plugin) generateIdempotencyKey(values ...string) string {
	joined := strings.Join(values, "-")
	hash := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(hash[:])
}
