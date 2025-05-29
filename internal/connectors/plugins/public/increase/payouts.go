package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

const (
	thirdPartyFulfillmentMethod    = "third_party"
	physicalCheckFulfillmentMethod = "physical_check"
	increaseACHPayoutMethod        = "ach"
	increaseWirePaymentMethod      = "wire"
	increaseCheckPaymentMethod     = "check"
	increaseRTPPaymentMethod       = "rtp"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validatePayoutRequests(pi); err != nil {
		return nil, err
	}

	payoutMethod := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey)
	idempotencyKey := p.generateIdempotencyKey(pi.Reference)

	switch strings.ToLower(payoutMethod) {
	case increaseWirePaymentMethod:
		wrp := &client.WireTransferPayoutRequest{
			AccountID:          pi.SourceAccount.Reference,
			Amount:             pi.Amount.Int64(),
			ExternalAccountID:  pi.DestinationAccount.Reference,
			MessageToRecipient: pi.Description,
		}
		if pi.DestinationAccount.Name != nil {
			wrp.BeneficiaryName = *pi.DestinationAccount.Name
		}
		resp, err := p.client.InitiateWireTransferPayout(
			ctx,
			wrp,
			idempotencyKey,
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
			Amount:                pi.Amount.Int64(),
			SourceAccountNumberID: sourceAccountNumberID,
			FulfillmentMethod:     fulfillmentMethod,
		}
		if fulfillmentMethod == thirdPartyFulfillmentMethod {
			check.ThirdParty = &client.ThirdParty{
				CheckNumber: models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCheckNumberMetadataKey),
			}
		} else if fulfillmentMethod == physicalCheckFulfillmentMethod {
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
			return nil, fmt.Errorf("invalid fulfillmentMethod %s", fulfillmentMethod)
		}
		resp, err := p.client.InitiateCheckTransferPayout(
			ctx,
			check,
			idempotencyKey,
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	case increaseRTPPaymentMethod:
		sourceAccountNumberID := models.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
		rtp := &client.RTPPayoutRequest{
			Amount:                pi.Amount.Int64(),
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
			idempotencyKey,
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	case increaseACHPayoutMethod:
		apr := &client.ACHPayoutRequest{
			AccountID:           pi.SourceAccount.Reference,
			Amount:              pi.Amount.Int64(),
			ExternalAccountID:   pi.DestinationAccount.Reference,
			StatementDescriptor: pi.Description,
		}
		if pi.DestinationAccount.Name != nil {
			apr.IndividualName = *pi.DestinationAccount.Name
		}
		resp, err := p.client.InitiateACHTransferPayout(
			ctx,
			apr,
			idempotencyKey,
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

	pspPayment := &models.PSPPayment{
		ParentReference:        "",
		Reference:              from.ID,
		CreatedAt:              createdAt,
		Type:                   models.PAYMENT_TYPE_PAYOUT,
		Amount:                 big.NewInt(from.Amount),
		Asset:                  currency.FormatAsset(supportedCurrenciesWithDecimal, from.Currency),
		Scheme:                 models.PAYMENT_SCHEME_OTHER,
		Status:                 status,
		SourceAccountReference: &from.AccountID,
		Metadata: map[string]string{
			client.IncreaseCheckNumberMetadataKey:   from.CheckNumber,
			client.IncreaseRoutingNumberMetadataKey: from.RoutingNumber,
			client.IncreaseAccountNumberMetadataKey: from.AccountNumber,
			client.IncreaseRecipientNameMetadataKey: from.RecipientName,
		},
		Raw: raw,
	}
	pspPayment = fillPayoutReference(from, pspPayment)
	if from.ExternalAccountId != "" {
		pspPayment.DestinationAccountReference = &from.ExternalAccountId
	} else {
		unknown := "Unknown"
		pspPayment.DestinationAccountReference = &unknown
	}

	return pspPayment, nil
}

func fillPayoutReference(transfer *client.PayoutResponse, pspPayment *models.PSPPayment) *models.PSPPayment {
	if transfer.TransactionID != "" {
		pspPayment.Reference = transfer.TransactionID
	} else if transfer.PendingTransactionID != "" {
		pspPayment.Reference = transfer.PendingTransactionID
	} else {
		pspPayment.Reference = transfer.ID
	}

	return pspPayment
}
