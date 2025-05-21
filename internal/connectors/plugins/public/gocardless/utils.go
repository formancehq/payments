package gocardless

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"

	"github.com/formancehq/payments/internal/models"
)

func validateExternalBankAccount(newExternalBankAccount models.BankAccount) error {
	if newExternalBankAccount.AccountNumber == nil {
		return models.NewConnectorValidationError("accountNumber", ErrMissingAccountNumber)
	}

	if newExternalBankAccount.Country == nil {
		return models.NewConnectorValidationError("country", ErrorMissingCountry)
	}

	reqCurrency, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCurrencyMetadataKey)
	if err != nil {
		return models.NewConnectorValidationError(client.GocardlessCurrencyMetadataKey, ErrMissingCurrency)
	}

	_, ok := SupportedCurrenciesWithDecimal[reqCurrency]

	if !ok {

		return ErrNotSupportedCurrency
	}

	creditor, _ := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCreditorMetadataKey)
	customer, _ := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCustomerMetadataKey)

	if len(creditor) > 1 && creditor[:2] != "CR" {
		return models.NewConnectorValidationError(client.GocardlessCreditorMetadataKey, ErrInvalidCreditorID)
	}

	if len(customer) > 1 && customer[:2] != "CU" {
		return models.NewConnectorValidationError(client.GocardlessCustomerMetadataKey, ErrInvalidCustomerID)
	}

	if (customer == "" && creditor == "") || (customer != "" && creditor != "") {
		return models.NewConnectorValidationError(client.GocardlessCustomerMetadataKey+" and "+client.GocardlessCreditorMetadataKey, ErrCreditorAndCustomerIDProvided)
	}

	if *newExternalBankAccount.Country == "US" {

		swiftCode := newExternalBankAccount.SwiftBicCode

		if swiftCode == nil {
			return models.NewConnectorValidationError("swiftBicCode", ErrMissingSwiftCode)
		}

	}

	accountType, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessAccountTypeMetadataKey)

	if reqCurrency == "USD" && err != nil {

		return models.NewConnectorValidationError(client.GocardlessAccountTypeMetadataKey, ErrMissingAccountType)
	}

	if accountType != "" && accountType != "checking" && accountType != "savings" {
		return models.NewConnectorValidationError(client.GocardlessAccountTypeMetadataKey, ErrInvalidAccountType)

	}

	if reqCurrency != "USD" && accountType != "" {

		return models.NewConnectorValidationError(client.GocardlessAccountTypeMetadataKey, ErrAccountTypeProvided)
	}

	return nil
}

func externalAccountFromGocardlessData(data client.GocardlessGenericAccount) (models.PSPAccount, error) {

	raw, err := json.Marshal(data)

	if err != nil {
		return models.PSPAccount{}, err
	}

	defaultAsset := currency.FormatAsset(SupportedCurrenciesWithDecimal, data.Currency)

	metadata := extractExternalAccountMetadata(data.Metadata)

	if data.AccountType != "" {
		metadata[client.GocardlessAccountTypeMetadataKey] = data.AccountType
	}

	return models.PSPAccount{
		Reference:    data.ID,
		CreatedAt:    data.CreatedAt,
		Name:         &data.AccountHolderName,
		Metadata:     metadata,
		DefaultAsset: &defaultAsset,
		Raw:          raw,
	}, nil
}

func extractExternalAccountMetadata(externalMetadata map[string]interface{}) map[string]string {
	metadata := make(map[string]string)
	for k, v := range externalMetadata {
		var stringValue string

		valueType := reflect.TypeOf(v)
		switch valueType.Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:
			jsonBytes, err := json.Marshal(v)
			if err == nil {
				stringValue = string(jsonBytes)
			}
		default:
			stringValue = fmt.Sprintf("%v", v)
		}

		// check if the key is already prefixed with the gocardless namespace
		if strings.HasPrefix(k, client.GocardlessMetadataSpecNamespace) {
			metadata[k] = stringValue
		} else {
			gocardlessNamespaceKey := client.GocardlessMetadataSpecNamespace + k
			metadata[gocardlessNamespaceKey] = stringValue
		}
	}

	return metadata
}
