package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/increase/client"
	"github.com/formancehq/payments/pkg/connector"
)

const (
	thirdPartyFulfillmentMethod    = "third_party"
	physicalCheckFulfillmentMethod = "physical_check"
	increaseACHPayoutMethod        = "ach"
	increaseWirePaymentMethod      = "wire"
	increaseCheckPaymentMethod     = "check"
	increaseRTPPaymentMethod       = "rtp"
	increaseFedNowPaymentMethod    = "fednow"
	increaseSWIFTPaymentMethod     = "swift"
)

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (*connector.PSPPayment, error) {
	if err := p.validatePayoutRequests(pi); err != nil {
		return nil, err
	}

	payoutMethod := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey)
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
		sourceAccountNumberID := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
		fulfillmentMethod := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseFulfillmentMethodMetadataKey)
		check := &client.CheckPayoutRequest{
			AccountID:             pi.SourceAccount.Reference,
			Amount:                pi.Amount.Int64(),
			SourceAccountNumberID: sourceAccountNumberID,
			FulfillmentMethod:     fulfillmentMethod,
		}
		switch fulfillmentMethod {
		case thirdPartyFulfillmentMethod:
			check.ThirdParty = &client.ThirdParty{
				CheckNumber: connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCheckNumberMetadataKey),
			}
		case physicalCheckFulfillmentMethod:
			check.PhysicalCheck = &client.PhysicalCheck{
				MailingAddress: client.MailingAddress{
					City:       connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCityMetadataKey),
					State:      connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseStateMetadataKey),
					PostalCode: connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePostalCodeMetadataKey),
					Line1:      connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseLine1MetadataKey),
				},
				Memo: pi.Description,
			}
			if pi.DestinationAccount.Name != nil {
				check.PhysicalCheck.RecipientName = *pi.DestinationAccount.Name
			}
		default:
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
		sourceAccountNumberID := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
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
	case increaseFedNowPaymentMethod:
		sourceAccountNumberID := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseSourceAccountNumberIdMetadataKey)
		fnr := &client.FedNowPayoutRequest{
			AccountID:                         pi.SourceAccount.Reference,
			Amount:                            pi.Amount.Int64(),
			SourceAccountNumberID:             sourceAccountNumberID,
			UnstructuredRemittanceInformation: pi.Description,
		}
		if pi.DestinationAccount.Name != nil {
			fnr.CreditorName = *pi.DestinationAccount.Name
		}
		if pi.SourceAccount.Name != nil {
			fnr.DebtorName = *pi.SourceAccount.Name
		}
		debtorName := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseDebtorNameMetadataKey)
		if debtorName != "" {
			fnr.DebtorName = debtorName
		}
		resp, err := p.client.InitiateFedNowTransferPayout(
			ctx,
			fnr,
			idempotencyKey,
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	case increaseSWIFTPaymentMethod:
		bic := connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseBankIdentificationCodeMetadataKey)
		swr := &client.SWIFTPayoutRequest{
			AccountID:              pi.SourceAccount.Reference,
			Amount:                 pi.Amount.Int64(),
			BankIdentificationCode: bic,
			CreditorAddress: client.SWIFTAddress{
				Line1:   connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCreditorAddressLine1MetadataKey),
				City:    connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCreditorAddressCityMetadataKey),
				Country: connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseCreditorAddressCountryMetadataKey),
			},
			DebtorAddress: client.SWIFTAddress{
				Line1:   connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseDebtorAddressLine1MetadataKey),
				City:    connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseDebtorAddressCityMetadataKey),
				Country: connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseDebtorAddressCountryMetadataKey),
			},
			InstructedCurrency: connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreaseInstructedCurrencyMetadataKey),
		}
		if pi.DestinationAccount.Name != nil {
			swr.CreditorName = *pi.DestinationAccount.Name
		}
		if pi.SourceAccount.Name != nil {
			swr.DebtorName = *pi.SourceAccount.Name
		}
		resp, err := p.client.InitiateSWIFTTransferPayout(
			ctx,
			swr,
			idempotencyKey,
		)
		if err != nil {
			return nil, err
		}
		return p.payoutToPayment(resp)
	default:
		return nil, fmt.Errorf("invalid payout method: %s", connector.ExtractNamespacedMetadata(pi.Metadata, client.IncreasePayoutMethodMetadataKey))
	}
}

func (p *Plugin) payoutToPayment(from *client.PayoutResponse) (*connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	status := matchPaymentStatus(from.Status)

	createdAt, err := time.Parse(time.RFC3339, from.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse posted date %s: %w", from.CreatedAt, err)
	}

	pspPayment := &connector.PSPPayment{
		ParentReference:        from.ID,
		CreatedAt:              createdAt,
		Type:                   connector.PAYMENT_TYPE_PAYOUT,
		Amount:                 big.NewInt(from.Amount),
		Asset:                  currency.FormatAsset(supportedCurrenciesWithDecimal, from.Currency),
		Scheme:                 connector.PAYMENT_SCHEME_OTHER,
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

func fillPayoutReference(transfer *client.PayoutResponse, pspPayment *connector.PSPPayment) *connector.PSPPayment {
	if transfer.TransactionID != "" {
		pspPayment.Reference = transfer.TransactionID
	} else if transfer.PendingTransactionID != "" {
		pspPayment.Reference = transfer.PendingTransactionID
	} else {
		pspPayment.Reference = transfer.ID
	}

	return pspPayment
}
