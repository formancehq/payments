package column

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	err := p.validateExternalBankAccount(ba)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	routingNumber := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnRoutingNumberMetadataKey)
	routingNumberType := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnRoutingNumberTypeMetadataKey)
	accountType := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAccountTypeMetadataKey)
	wireDrawdownAllowed := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnWireDrawdownAllowedMetadataKey) == "true"

	addressLine1 := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressLine1MetadataKey)
	addressLine2 := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressLine2MetadataKey)
	city := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressCityMetadataKey)
	state := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressStateMetadataKey)
	postalCode := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressPostalCodeMetadataKey)

	phone := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnPhoneMetadataKey)
	email := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnEmailMetadataKey)
	legalID := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnLegalIDMetadataKey)
	legalType := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnLegalTypeMetadataKey)
	localBankCode := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnLocalBankCodeMetadataKey)
	localAccountNumber := models.ExtractNamespacedMetadata(ba.Metadata, client.ColumnLocalAccountNumberMetadataKey)

	address := client.ColumnAddress{}

	if addressLine1 != "" {
		address = client.ColumnAddress{
			Line1:       addressLine1,
			Line2:       addressLine2,
			City:        city,
			State:       state,
			PostalCode:  postalCode,
			CountryCode: *ba.Country,
		}
	}

	data := client.CounterPartyBankAccountRequest{
		Name:                ba.Name,
		RoutingNumber:       routingNumber,
		AccountNumber:       *ba.AccountNumber,
		RoutingNumberType:   routingNumberType,
		AccountType:         accountType,
		WireDrawdownAllowed: wireDrawdownAllowed,
		Address:             address,
		Phone:               phone,
		Email:               email,
		LegalID:             legalID,
		LegalType:           legalType,
		LocalBankCode:       localBankCode,
		LocalAccountNumber:  localAccountNumber,
	}

	bankAccount, err := p.client.CreateCounterPartyBankAccount(ctx, data)

	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	parsedTime, err := ParseColumnTimestamp(bankAccount.CreatedAt)

	if err != nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("failed to parse creation time: %w", err)
	}

	raw, err := json.Marshal(bankAccount)
	if err != nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("failed to marshal created bank account %w", err)
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: models.PSPAccount{
			Reference: bankAccount.ID,
			CreatedAt: parsedTime,
			Name:      &bankAccount.Name,
			Raw:       raw,
			Metadata: map[string]string{
				client.ColumnTypeMetadataKey:                 bankAccount.AccountType,
				client.ColumnAccountNumberMetadataKey:        bankAccount.AccountNumber,
				client.ColumnAddressCityMetadataKey:          bankAccount.Address.City,
				client.ColumnAddressCountryCodeMetadataKey:   bankAccount.Address.CountryCode,
				client.ColumnAddressLine1MetadataKey:         bankAccount.Address.Line1,
				client.ColumnAddressLine2MetadataKey:         bankAccount.Address.Line2,
				client.ColumnAddressPostalCodeMetadataKey:    bankAccount.Address.PostalCode,
				client.ColumnAddressStateMetadataKey:         bankAccount.Address.State,
				client.ColumnDescriptionMetadataKey:          bankAccount.Description,
				client.ColumnEmailMetadataKey:                bankAccount.Email,
				client.ColumnIsColumnAccountMetadataKey:      strconv.FormatBool(bankAccount.IsColumnAccount),
				client.ColumnLegalIDMetadataKey:              bankAccount.LegalID,
				client.ColumnLegalTypeMetadataKey:            bankAccount.LegalType,
				client.ColumnLocalAccountNumberMetadataKey:   bankAccount.LocalAccountNumber,
				client.ColumnLocalBankCodeMetadataKey:        bankAccount.LocalBankCode,
				client.ColumnLocalBankCountryCodeMetadataKey: bankAccount.LocalBankCountryCode,
				client.ColumnLocalBankNameMetadataKey:        bankAccount.LocalBankName,
				client.ColumnPhoneMetadataKey:                bankAccount.Phone,
				client.ColumnRoutingNumberTypeMetadataKey:    bankAccount.RoutingNumberType,
				client.ColumnUpdatedAtMetadataKey:            bankAccount.UpdatedAt,
				client.ColumnRoutingNumberMetadataKey:        bankAccount.RoutingNumber,
				client.ColumnWireDrawdownAllowedMetadataKey:  strconv.FormatBool(bankAccount.WireDrawdownAllowed),
			},
		},
	}, nil
}
