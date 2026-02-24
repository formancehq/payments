package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/mangopay/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createBankAccount(ctx context.Context, ba connector.BankAccount) (connector.CreateBankAccountResponse, error) {
	userID := connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayUserIDMetadataKey)
	if userID == "" {
		return connector.CreateBankAccountResponse{}, connector.NewConnectorValidationError(client.MangopayUserIDMetadataKey, connector.ErrMissingConnectorMetadata)
	}

	ownerAddress := client.OwnerAddress{
		AddressLine1: connector.ExtractNamespacedMetadata(ba.Metadata, connector.BankAccountOwnerAddressLine1MetadataKey),
		AddressLine2: connector.ExtractNamespacedMetadata(ba.Metadata, connector.BankAccountOwnerAddressLine2MetadataKey),
		City:         connector.ExtractNamespacedMetadata(ba.Metadata, connector.BankAccountOwnerCityMetadataKey),
		Region:       connector.ExtractNamespacedMetadata(ba.Metadata, connector.BankAccountOwnerRegionMetadataKey),
		PostalCode:   connector.ExtractNamespacedMetadata(ba.Metadata, connector.BankAccountOwnerPostalCodeMetadataKey),
		Country: func() string {
			if ba.Country == nil {
				return ""
			}
			return *ba.Country
		}(),
	}

	var mangopayBankAccount *client.BankAccount
	if ba.IBAN != nil {
		req := &client.CreateIBANBankAccountRequest{
			OwnerName:    ba.Name,
			OwnerAddress: &ownerAddress,
			IBAN:         *ba.IBAN,
			BIC: func() string {
				if ba.SwiftBicCode == nil {
					return ""
				}
				return *ba.SwiftBicCode
			}(),
			Tag: connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
		}

		var err error
		mangopayBankAccount, err = p.client.CreateIBANBankAccount(ctx, userID, req)
		if err != nil {
			return connector.CreateBankAccountResponse{}, connector.NewWrappedError(
				fmt.Errorf("failed to create IBAN bank account: %w", err),
				connector.ErrFailedAccountCreation,
			)
		}
	} else {
		if ba.Country == nil {
			ba.Country = pointer.For("")
		}
		switch *ba.Country {
		case "US":
			if ba.AccountNumber == nil {
				return connector.CreateBankAccountResponse{}, connector.ErrMissingAccountInRequest
			}

			req := &client.CreateUSBankAccountRequest{
				OwnerName:          ba.Name,
				OwnerAddress:       &ownerAddress,
				AccountNumber:      *ba.AccountNumber,
				ABA:                connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayABAMetadataKey),
				DepositAccountType: connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayDepositAccountTypeMetadataKey),
				Tag:                connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateUSBankAccount(ctx, userID, req)
			if err != nil {
				return connector.CreateBankAccountResponse{}, connector.NewWrappedError(
					fmt.Errorf("failed to create US bank account: %w", err),
					connector.ErrFailedAccountCreation,
				)
			}

		case "CA":
			if ba.AccountNumber == nil {
				return connector.CreateBankAccountResponse{}, connector.ErrMissingAccountInRequest
			}
			req := &client.CreateCABankAccountRequest{
				OwnerName:         ba.Name,
				OwnerAddress:      &ownerAddress,
				AccountNumber:     *ba.AccountNumber,
				InstitutionNumber: connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayInstitutionNumberMetadataKey),
				BranchCode:        connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayBranchCodeMetadataKey),
				BankName:          connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayBankNameMetadataKey),
				Tag:               connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateCABankAccount(ctx, userID, req)
			if err != nil {
				return connector.CreateBankAccountResponse{}, connector.NewWrappedError(
					fmt.Errorf("failed to create CA bank account: %w", err),
					connector.ErrFailedAccountCreation,
				)
			}

		case "GB":
			if ba.AccountNumber == nil {
				return connector.CreateBankAccountResponse{}, connector.ErrMissingAccountInRequest
			}

			req := &client.CreateGBBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &ownerAddress,
				AccountNumber: *ba.AccountNumber,
				SortCode:      connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopaySortCodeMetadataKey),
				Tag:           connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateGBBankAccount(ctx, userID, req)
			if err != nil {
				return connector.CreateBankAccountResponse{}, connector.NewWrappedError(
					fmt.Errorf("failed to create GB bank account: %w", err),
					connector.ErrFailedAccountCreation,
				)
			}

		default:
			if ba.AccountNumber == nil {
				return connector.CreateBankAccountResponse{}, connector.ErrMissingAccountInRequest
			}

			req := &client.CreateOtherBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &ownerAddress,
				AccountNumber: *ba.AccountNumber,
				BIC: func() string {
					if ba.SwiftBicCode == nil {
						return ""
					}
					return *ba.SwiftBicCode
				}(),
				Country: *ba.Country,
				Tag:     connector.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateOtherBankAccount(ctx, userID, req)
			if err != nil {
				return connector.CreateBankAccountResponse{}, connector.NewWrappedError(
					fmt.Errorf("failed to create other bank account: %w", err),
					connector.ErrFailedAccountCreation,
				)
			}
		}
	}

	var account connector.PSPAccount
	if mangopayBankAccount != nil {
		raw, err := json.Marshal(mangopayBankAccount)
		if err != nil {
			return connector.CreateBankAccountResponse{}, err
		}

		account = connector.PSPAccount{
			Reference: mangopayBankAccount.ID,
			CreatedAt: time.Unix(mangopayBankAccount.CreationDate, 0),
			Name:      &mangopayBankAccount.OwnerName,
			Metadata: map[string]string{
				"user_id": userID,
			},
			Raw: raw,
		}

	}

	return connector.CreateBankAccountResponse{
		RelatedAccount: account,
	}, nil
}
