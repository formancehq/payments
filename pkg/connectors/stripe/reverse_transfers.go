package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/stripe/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/stripe/stripe-go/v80"
)

const (
	transferIDMetadataKey = "com.stripe.spec/transfer_id"
)

func validateReverseTransferRequest(pir connector.PSPPaymentInitiationReversal) error {
	_, ok := pir.Metadata[transferIDMetadataKey]
	if !ok {
		return connector.NewWrappedError(
			fmt.Errorf("transfer id is required in metadata of transfer reversal request"),
			connector.ErrInvalidRequest,
		)
	}
	return nil
}
func (p *Plugin) reverseTransfer(ctx context.Context, pir connector.PSPPaymentInitiationReversal) (connector.PSPPayment, error) {
	if err := validateReverseTransferRequest(pir); err != nil {
		return connector.PSPPayment{}, err
	}
	var account *string = nil
	if pir.RelatedPaymentInitiation.SourceAccount != nil && pir.RelatedPaymentInitiation.SourceAccount.Reference != p.client.GetRootAccountID() {
		account = &pir.RelatedPaymentInitiation.SourceAccount.Reference
	}
	resp, err := p.client.ReverseTransfer(
		ctx,
		client.ReverseTransferRequest{
			IdempotencyKey:   pir.Reference,
			StripeTransferID: pir.Metadata[transferIDMetadataKey],
			Account:          account,
			Amount:           pir.Amount.Int64(),
			Description:      pir.Description,
			Metadata:         pir.Metadata,
		},
	)
	if err != nil {
		return connector.PSPPayment{}, err
	}
	payment, err := fromTransferReversalToPayment(resp, account, &pir.RelatedPaymentInitiation.DestinationAccount.Reference)
	if err != nil {
		return connector.PSPPayment{}, err
	}
	return payment, nil
}
func fromTransferReversalToPayment(from *stripe.TransferReversal, source, destination *string) (connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return connector.PSPPayment{}, err
	}
	return connector.PSPPayment{
		ParentReference:             from.Transfer.BalanceTransaction.ID,
		Reference:                   from.BalanceTransaction.ID,
		CreatedAt:                   time.Unix(from.Created, 0),
		Type:                        connector.PAYMENT_TYPE_TRANSFER,
		Amount:                      big.NewInt(from.Amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, strings.ToUpper(string(from.Currency))),
		Scheme:                      connector.PAYMENT_SCHEME_OTHER,
		Status:                      connector.PAYMENT_STATUS_REFUNDED,
		SourceAccountReference:      source,
		DestinationAccountReference: destination,
		Metadata:                    from.Metadata,
		Raw:                         raw,
	}, nil
}
