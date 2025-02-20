package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

const (
	thirdPartyFufillmentMethod    = "third_party"
	physicalCheckFufillmentMethod = "physical_check"
	increaseACHPayoutMethod       = "ach"
	increaseWirePaymentMethod     = "wire"
	increaseCheckPaymentMethod    = "check"
	increaseRTPPaymentMethod      = "rtp"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validatePayoutRequests(pi); err != nil {
		return nil, err
	}

	_, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to get string amount from big int: %v: %w", err, models.ErrInvalidRequest)
	}

	switch models.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey) {
	case increaseWirePaymentMethod:
		resp, err := p.client.InitiateWireTransferPayout(
			ctx,
			&client.WireTransferPayoutRequest{
				AccountID:          pi.SourceAccount.Reference,
				Amount:             json.Number(amount),
				ExternalAccountID:  pi.DestinationAccount.Reference,
				BeneficiaryName:    *pi.DestinationAccount.Name,
				MessageToRecipient: pi.Description,
			},
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	case increaseCheckPaymentMethod:
		check := &client.CheckPayoutRequest{
			AccountID:             pi.SourceAccount.Reference,
			Amount:                json.Number(amount),
			SourceAccountNumberID: models.ExtractNamespacedMetadata(pi.SourceAccount.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey),
			FulfillmentMethod:     models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseFulfillmentMethodMetadataKey),
		}
		fulfillmentMethod := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseFulfillmentMethodMetadataKey)
		if fulfillmentMethod == thirdPartyFufillmentMethod {
			check.ThirdParty = struct {
				CheckNumber string "json:\"check_number\""
			}{
				CheckNumber: models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCheckNumberMetadataKey),
			}
		} else {
			check.PhysicalCheck.MailingAddress = client.MailingAddress{
				City:       models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCityMetadataKey),
				State:      models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseStateMetadataKey),
				PostalCode: models.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePostalCodeMetadataKey),
				Line1:      models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseLine1MetadataKey),
			}
			check.PhysicalCheck.Memo = pi.Description
			check.PhysicalCheck.RecipientName = *pi.DestinationAccount.Name
		}
		resp, err := p.client.InitiateCheckTransferPayout(
			ctx,
			check,
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	case increaseRTPPaymentMethod:
		resp, err := p.client.InitiateRTPTransferPayout(
			ctx,
			&client.RTPPayoutRequest{
				Amount:                json.Number(amount),
				ExternalAccountID:     pi.DestinationAccount.Reference,
				RemittanceInformation: pi.Description,
				CreditorName:          *pi.DestinationAccount.Name,
				SourceAccountNumberID: models.ExtractNamespacedMetadata(pi.SourceAccount.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey),
			},
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	default:
		resp, err := p.client.InitiateACHTransferPayout(
			ctx,
			&client.ACHPayoutRequest{
				AccountID:           pi.SourceAccount.Reference,
				Amount:              json.Number(amount),
				ExternalAccountID:   pi.DestinationAccount.Reference,
				StatementDescriptor: pi.Description,
				IndividualName:      *pi.DestinationAccount.Name,
			},
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	}
}

func (p *Plugin) payoutToPayment(from *client.PayoutResponse) (*models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	status := matchPaymentStatus(from.Status)

	createdAt, err := time.Parse(time.RFC3339, from.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", from.CreatedAt, err)
	}

	precision, ok := supportedCurrenciesWithDecimal[from.Currency]
	if !ok {
		return nil, fmt.Errorf("unsupported currency: %s", from.Currency)
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount %s: %w", from.Amount, err)
	}

	return &models.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      &from.AccountID,
		DestinationAccountReference: &from.ExternalAccountId,
		Metadata: map[string]string{
			client.IncreaseCheckNumberMetadataKey:   from.CheckNumber,
			client.IncreaseRoutingNumberMetadataKey: from.RoutingNumber,
			client.IncreaseAccountNumberMetadataKey: from.AccountNumber,
			client.IncreaseRecipientNameMetadataKey: from.RecipientName,
		},
		Raw: raw,
	}, nil
}
