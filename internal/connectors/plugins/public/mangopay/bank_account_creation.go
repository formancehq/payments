package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
)

func (p Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	userID := models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayUserIDMetadataKey)
	if userID == "" {
		return models.CreateBankAccountResponse{}, fmt.Errorf("missing userID in bank account metadata")
	}

	ownerAddress := client.OwnerAddress{
		AddressLine1: models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerAddressLine1MetadataKey),
		AddressLine2: models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerAddressLine2MetadataKey),
		City:         models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerCityMetadataKey),
		Region:       models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerRegionMetadataKey),
		PostalCode:   models.ExtractNamespacedMetadata(ba.Metadata, models.BankAccountOwnerPostalCodeMetadataKey),
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
			Tag: models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
		}

		var err error
		mangopayBankAccount, err = p.client.CreateIBANBankAccount(ctx, userID, req)
		if err != nil {
			return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
		}
	} else {
		if ba.Country == nil {
			ba.Country = pointer.For("")
		}
		switch *ba.Country {
		case "US":
			if ba.AccountNumber == nil {
				return models.CreateBankAccountResponse{}, models.ErrMissingAccountInRequest
			}

			req := &client.CreateUSBankAccountRequest{
				OwnerName:          ba.Name,
				OwnerAddress:       &ownerAddress,
				AccountNumber:      *ba.AccountNumber,
				ABA:                models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayABAMetadataKey),
				DepositAccountType: models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayDepositAccountTypeMetadataKey),
				Tag:                models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateUSBankAccount(ctx, userID, req)
			if err != nil {
				return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
			}

		case "CA":
			if ba.AccountNumber == nil {
				return models.CreateBankAccountResponse{}, models.ErrMissingAccountInRequest
			}
			req := &client.CreateCABankAccountRequest{
				OwnerName:         ba.Name,
				OwnerAddress:      &ownerAddress,
				AccountNumber:     *ba.AccountNumber,
				InstitutionNumber: models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayInstitutionNumberMetadataKey),
				BranchCode:        models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayBranchCodeMetadataKey),
				BankName:          models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayBankNameMetadataKey),
				Tag:               models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateCABankAccount(ctx, userID, req)
			if err != nil {
				return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
			}

		case "GB":
			if ba.AccountNumber == nil {
				return models.CreateBankAccountResponse{}, models.ErrMissingAccountInRequest
			}

			req := &client.CreateGBBankAccountRequest{
				OwnerName:     ba.Name,
				OwnerAddress:  &ownerAddress,
				AccountNumber: *ba.AccountNumber,
				SortCode:      models.ExtractNamespacedMetadata(ba.Metadata, client.MangopaySortCodeMetadataKey),
				Tag:           models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateGBBankAccount(ctx, userID, req)
			if err != nil {
				return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
			}

		default:
			if ba.AccountNumber == nil {
				return models.CreateBankAccountResponse{}, models.ErrMissingAccountInRequest
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
				Tag:     models.ExtractNamespacedMetadata(ba.Metadata, client.MangopayTagMetadataKey),
			}

			var err error
			mangopayBankAccount, err = p.client.CreateOtherBankAccount(ctx, userID, req)
			if err != nil {
				return models.CreateBankAccountResponse{}, fmt.Errorf("%w: %w", models.ErrFailedAccountCreation, err)
			}
		}
	}

	var account models.PSPAccount
	if mangopayBankAccount != nil {
		raw, err := json.Marshal(mangopayBankAccount)
		if err != nil {
			return models.CreateBankAccountResponse{}, err
		}

		account = models.PSPAccount{
			Reference: mangopayBankAccount.ID,
			CreatedAt: time.Unix(mangopayBankAccount.CreationDate, 0),
			Name:      &mangopayBankAccount.OwnerName,
			Metadata: map[string]string{
				"user_id": userID,
			},
			Raw: raw,
		}

	}

	return models.CreateBankAccountResponse{
		RelatedAccount: account,
	}, nil
}
