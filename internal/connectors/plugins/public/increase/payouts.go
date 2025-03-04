package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

	amount := pi.Amount.String() // increase uses minor units
	payoutMethod := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey)

	switch strings.ToLower(payoutMethod) {
	case increaseWirePaymentMethod:
		wrp := &client.WireTransferPayoutRequest{
			AccountID:          pi.SourceAccount.Reference,
			Amount:             json.Number(amount),
			ExternalAccountID:  pi.DestinationAccount.Reference,
			MessageToRecipient: pi.Description,
		}
		if pi.DestinationAccount.Name != nil {
			wrp.BeneficiaryName = *pi.DestinationAccount.Name
		}
		resp, err := p.client.InitiateWireTransferPayout(
			ctx,
			wrp,
			fmt.Sprintf("wire%s%s", pi.SourceAccount.Reference, pi.DestinationAccount.Reference),
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	case increaseCheckPaymentMethod:
		sourceAccountNumberID := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
		fulfillmentMethod := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseFulfillmentMethodMetadataKey)
		check := &client.CheckPayoutRequest{
			AccountID:             pi.SourceAccount.Reference,
			Amount:                json.Number(amount),
			SourceAccountNumberID: models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey),
			FulfillmentMethod:     fulfillmentMethod,
		}
		if fulfillmentMethod == thirdPartyFufillmentMethod {
			check.ThirdParty = &struct {
				CheckNumber string `json:"check_number"`
			}{
				CheckNumber: models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCheckNumberMetadataKey),
			}
		} else if fulfillmentMethod == physicalCheckFufillmentMethod {
			check.PhysicalCheck = &client.PhysicalCheck{
				MailingAddress: client.MailingAddress{
					City:       models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCityMetadataKey),
					State:      models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseStateMetadataKey),
					PostalCode: models.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePostalCodeMetadataKey),
					Line1:      models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseLine1MetadataKey),
				},
				Memo: pi.Description,
			}
			if pi.DestinationAccount.Name != nil {
				check.PhysicalCheck.RecipientName = *pi.DestinationAccount.Name
			}
		} else {
			return nil, fmt.Errorf("invalid fufillmentMethod %s", fulfillmentMethod)
		}
		resp, err := p.client.InitiateCheckTransferPayout(
			ctx,
			check,
			fmt.Sprintf("check%s%s", sourceAccountNumberID, pi.DestinationAccount.Reference),
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	case increaseRTPPaymentMethod:
		sourceAccountNumberID := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
		rtp := &client.RTPPayoutRequest{
			Amount:                json.Number(amount),
			ExternalAccountID:     pi.DestinationAccount.Reference,
			RemittanceInformation: pi.Description,
			SourceAccountNumberID: sourceAccountNumberID,
		}
		if pi.DestinationAccount.Name != nil {
			rtp.CreditorName = *pi.DestinationAccount.Name
		}
		resp, err := p.client.InitiateRTPTransferPayout(
			ctx,
			rtp,
			fmt.Sprintf("rtp%s%s", sourceAccountNumberID, pi.DestinationAccount.Reference),
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	case increaseACHPayoutMethod:
		apr := &client.ACHPayoutRequest{
			AccountID:           pi.SourceAccount.Reference,
			Amount:              json.Number(amount),
			ExternalAccountID:   pi.DestinationAccount.Reference,
			StatementDescriptor: pi.Description,
		}
		if pi.DestinationAccount.Name != nil {
			apr.IndividualName = *pi.DestinationAccount.Name
		}
		resp, err := p.client.InitiateACHTransferPayout(
			ctx,
			apr,
			fmt.Sprintf("ach%s%s", pi.SourceAccount.Reference, pi.DestinationAccount.Reference),
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	default:
		return nil, fmt.Errorf("invalid payout method: %s", models.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey))
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
