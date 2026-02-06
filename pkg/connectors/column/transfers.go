package column

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createTransfer(ctx context.Context, pi connector.PSPPaymentInitiation) (*connector.PSPPayment, error) {
	if err := p.validateTransferRequest(pi); err != nil {
		return nil, err
	}

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, connector.ErrInvalidRequest)
	}

	allowOverdraft := connector.ExtractNamespacedMetadata(pi.Metadata, client.ColumnAllowOverdraftMetadataKey)
	hold := connector.ExtractNamespacedMetadata(pi.Metadata, client.ColumnHoldMetadataKey)

	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			Amount:                pi.Amount.Int64(),
			CurrencyCode:          curr,
			SenderBankAccountId:   pi.SourceAccount.Reference,
			ReceiverBankAccountId: pi.DestinationAccount.Reference,
			AllowOverdraft:        allowOverdraft == "true",
			Hold:                  hold == "true",
			Details: client.TransferRequestDetails{
				SenderName:           *pi.SourceAccount.Name,
				MerchantName:         pi.Metadata[client.ColumnMerchantNameMetadataKey],
				MerchantCategoryCode: pi.Metadata[client.ColumnMerchantCategoryCodeMetadataKey],
				AuthorizationMethod:  pi.Metadata[client.ColumnAuthorizationMethodMetadataKey],
				InternalTransferType: pi.Metadata[client.ColumnInternalTransferTypeMetadataKey],
				Website:              pi.Metadata[client.ColumnWebsiteMetadataKey],
				Address: client.ColumnAddress{
					Line1:       pi.Metadata[client.ColumnAddressLine1MetadataKey],
					Line2:       pi.Metadata[client.ColumnAddressLine2MetadataKey],
					City:        pi.Metadata[client.ColumnAddressCityMetadataKey],
					CountryCode: pi.Metadata[client.ColumnAddressCountryCodeMetadataKey],
					PostalCode:  pi.Metadata[client.ColumnAddressPostalCodeMetadataKey],
					State:       pi.Metadata[client.ColumnAddressStateMetadataKey],
				},
			},
		},
	)
	if err != nil {
		return &connector.PSPPayment{}, err
	}

	return p.transferToPayment(resp.ID, resp)
}

func (p *Plugin) transferToPayment(id string, transfer *client.TransferResponse) (*connector.PSPPayment, error) {
	raw, err := json.Marshal(transfer)
	if err != nil {
		return nil, err
	}

	status := matchPaymentStatus(transfer.Status)
	createdAt, err := time.Parse(time.RFC3339, transfer.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", transfer.CreatedAt, err)
	}

	metadata := map[string]string{
		client.ColumnUpdatedAtMetadataKey:             transfer.UpdatedAt,
		client.ColumnIdempotencyKeyMetadataKey:        transfer.IdempotencyKey,
		client.ColumnSenderBankAccountIDMetadataKey:   transfer.SenderBankAccountID,
		client.ColumnReceiverBankAccountIDMetadataKey: transfer.ReceiverBankAccountID,
		client.ColumnDescriptionMetadataKey:           transfer.Description,
		client.ColumnAllowOverdraftMetadataKey:        fmt.Sprintf("%t", transfer.AllowOverdraft),
	}

	if len(transfer.Details) > 0 {
		detailsJSON, err := json.Marshal(transfer.Details)
		if err == nil {
			metadata[client.ColumnDetailsMetadataKey] = string(detailsJSON)
		}
	}

	return &connector.PSPPayment{
		Reference:                   id,
		ParentReference:             transfer.ID,
		CreatedAt:                   createdAt,
		Type:                        connector.PAYMENT_TYPE_TRANSFER,
		Amount:                      big.NewInt(transfer.Amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.CurrencyCode),
		Scheme:                      connector.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &transfer.SenderAccountNumberID,
		DestinationAccountReference: &transfer.ReceiverAccountNumberID,
		Raw:                         raw,
		Metadata:                    metadata,
	}, nil
}

func matchPaymentStatus(status string) connector.PaymentStatus {
	switch status {
	case "COMPLETED":
		return connector.PAYMENT_STATUS_SUCCEEDED
	case "REJECTED", "CANCELED":
		return connector.PAYMENT_STATUS_FAILED
	case "HOLD":
		return connector.PAYMENT_STATUS_PENDING
	default:
		return connector.PAYMENT_STATUS_UNKNOWN
	}
}
