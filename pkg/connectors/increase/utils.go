package increase

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/formancehq/payments/pkg/connectors/increase/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) validatePayoutRequests(pi connector.PSPPaymentInitiation) error {
	payoutMethod := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey)
	if payoutMethod == "" {
		return connector.NewConnectorValidationError(client.IncreasePayoutMethodMetadataKey, connector.ErrMissingConnectorMetadata)
	}

	validMethods := map[string]bool{
		increaseACHPayoutMethod:    true,
		increaseWirePaymentMethod:  true,
		increaseCheckPaymentMethod: true,
		increaseRTPPaymentMethod:   true,
	}

	if !validMethods[payoutMethod] {
		return connector.NewConnectorValidationError(client.IncreasePayoutMethodMetadataKey, connector.ErrInvalidRequest)
	}

	if pi.Description == "" {
		return connector.NewConnectorValidationError("description", connector.ErrMissingConnectorField)
	}

	if pi.Amount == nil {
		return connector.NewConnectorValidationError("amount", connector.ErrMissingConnectorField)
	}

	if pi.SourceAccount == nil {
		return connector.NewConnectorValidationError("sourceAccount", connector.ErrMissingConnectorField)
	}

	if pi.DestinationAccount == nil {
		return connector.NewConnectorValidationError("destinationAccount", connector.ErrMissingConnectorField)
	}

	if payoutMethod == increaseCheckPaymentMethod && connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseFulfillmentMethodMetadataKey) == "" {
		return connector.NewConnectorValidationError(client.IncreaseFulfillmentMethodMetadataKey, connector.ErrMissingConnectorMetadata)
	}

	sourceAccountNumberID := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
	if sourceAccountNumberID == "" && (payoutMethod == increaseCheckPaymentMethod || payoutMethod == increaseRTPPaymentMethod) {
		return connector.NewConnectorValidationError(client.IncreaseSourceAccountNumberIdMetadataKey, connector.ErrMissingConnectorMetadata)
	}

	return nil
}

func (p *Plugin) validateTransferRequests(pi connector.PSPPaymentInitiation) error {
	if pi.Amount == nil {
		return connector.NewConnectorValidationError("amount", connector.ErrMissingConnectorField)
	}

	if pi.SourceAccount == nil {
		return connector.NewConnectorValidationError("sourceAccount", connector.ErrMissingConnectorField)
	}

	if pi.DestinationAccount == nil {
		return connector.NewConnectorValidationError("destinationAccount", connector.ErrMissingConnectorField)
	}

	if pi.Description == "" {
		return connector.NewConnectorValidationError("description", connector.ErrMissingConnectorField)
	}

	return nil
}

func (p *Plugin) validateBankAccountRequests(ba connector.BankAccount) error {
	routingNumber := connector.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseRoutingNumberMetadataKey)
	if routingNumber == "" {
		return connector.NewConnectorValidationError(client.IncreaseRoutingNumberMetadataKey, connector.ErrMissingConnectorMetadata)
	}

	accountHolder := connector.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseAccountHolderMetadataKey)
	if accountHolder == "" {
		return connector.NewConnectorValidationError(client.IncreaseAccountHolderMetadataKey, connector.ErrMissingConnectorMetadata)
	}

	description := connector.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseDescriptionMetadataKey)
	if description == "" {
		return connector.NewConnectorValidationError(client.IncreaseDescriptionMetadataKey, connector.ErrMissingConnectorMetadata)
	}

	if ba.AccountNumber == nil {
		return connector.NewConnectorValidationError("AccountNumber", connector.ErrMissingConnectorField)
	}

	return nil
}

func (p *Plugin) generateIdempotencyKey(values ...string) string {
	joined := strings.Join(values, "-")
	hash := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(hash[:])
}
