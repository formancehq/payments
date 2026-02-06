package column

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createBankAccount(ctx context.Context, ba connector.BankAccount) (connector.CreateBankAccountResponse, error) {
	err := p.validateExternalBankAccount(ba)
	if err != nil {
		return connector.CreateBankAccountResponse{}, err
	}

	routingNumber := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnRoutingNumberMetadataKey)
	routingNumberType := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnRoutingNumberTypeMetadataKey)
	accountType := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAccountTypeMetadataKey)
	wireDrawdownAllowed := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnWireDrawdownAllowedMetadataKey) == "true"

	addressLine1 := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressLine1MetadataKey)
	addressLine2 := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressLine2MetadataKey)
	city := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressCityMetadataKey)
	state := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressStateMetadataKey)
	postalCode := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnAddressPostalCodeMetadataKey)

	phone := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnPhoneMetadataKey)
	email := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnEmailMetadataKey)
	legalID := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnLegalIDMetadataKey)
	legalType := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnLegalTypeMetadataKey)
	localBankCode := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnLocalBankCodeMetadataKey)
	localAccountNumber := connector.ExtractNamespacedMetadata(ba.Metadata, client.ColumnLocalAccountNumberMetadataKey)

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
		return connector.CreateBankAccountResponse{}, err
	}

	parsedTime, err := ParseColumnTimestamp(bankAccount.CreatedAt)

	if err != nil {
		return connector.CreateBankAccountResponse{}, fmt.Errorf("failed to parse creation time: %w", err)
	}

	raw, err := json.Marshal(bankAccount)
	if err != nil {
		return connector.CreateBankAccountResponse{}, fmt.Errorf("failed to marshal created bank account %w", err)
	}

	return connector.CreateBankAccountResponse{
		RelatedAccount: connector.PSPAccount{
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
