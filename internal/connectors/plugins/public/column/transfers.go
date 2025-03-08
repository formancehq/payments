package column

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferRequest(pi); err != nil {
		return nil, err
	}

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	}

	allowOverdraft := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnAllowOverdraftMetadataKey)
	hold := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnHoldMetadataKey)

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
					City:        pi.Metadata[client.ColumnCityMetadataKey],
					CountryCode: pi.Metadata[client.ColumnCountryCodeMetadataKey],
					PostalCode:  pi.Metadata[client.ColumnPostalCodeMetadataKey],
					State:       pi.Metadata[client.ColumnStateMetadataKey],
				},
			},
		},
	)
	if err != nil {
		return &models.PSPPayment{}, err
	}

	return p.transferToPayment(resp)
}

func (p *Plugin) transferToPayment(transfer *client.TransferResponse) (*models.PSPPayment, error) {
	raw, err := json.Marshal(transfer)
	if err != nil {
		return nil, err
	}

	status := matchPaymentStatus(transfer.Status)
	createdAt, err := time.Parse(time.RFC3339, transfer.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", transfer.CreatedAt, err)
	}

	precision, ok := supportedCurrenciesWithDecimal[transfer.CurrencyCode]
	if !ok {
		return nil, fmt.Errorf("unsupported currency: %s", transfer.CurrencyCode)
	}

	amount, err := currency.GetAmountWithPrecisionFromString(fmt.Sprintf("%d", transfer.Amount), precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount %d: %w", transfer.Amount, err)
	}

	return &models.PSPPayment{
		Reference:                   transfer.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.CurrencyCode),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &transfer.SenderAccountNumberID,
		DestinationAccountReference: &transfer.ReceiverAccountNumberID,
		Raw:                         raw,
	}, nil
}

func matchPaymentStatus(status string) models.PaymentStatus {
	switch status {
	case "COMPLETED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "REJECTED", "CANCELED":
		return models.PAYMENT_STATUS_FAILED
	case "HOLD":
		return models.PAYMENT_STATUS_PENDING
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}
