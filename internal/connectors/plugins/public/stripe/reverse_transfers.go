package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/stripe/stripe-go/v80"
)

const (
	transferIDMetadataKey = "com.stripe.spec/transfer_id"
)

func validateReverseTransferRequest(pir models.PSPPaymentInitiationReversal) error {
	_, ok := pir.Metadata[transferIDMetadataKey]
	if !ok {
		return errorsutils.NewWrappedError(
			fmt.Errorf("transfer id is required in metadata of transfer reversal request"),
			models.ErrInvalidRequest,
		)
	}
	return nil
}
func (p *Plugin) reverseTransfer(ctx context.Context, pir models.PSPPaymentInitiationReversal) (models.PSPPayment, error) {
	if err := validateReverseTransferRequest(pir); err != nil {
		return models.PSPPayment{}, err
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
		return models.PSPPayment{}, err
	}
	payment, err := fromTransferReversalToPayment(resp, account, &pir.RelatedPaymentInitiation.DestinationAccount.Reference)
	if err != nil {
		return models.PSPPayment{}, err
	}
	return payment, nil
}
func fromTransferReversalToPayment(from *stripe.TransferReversal, source, destination *string) (models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return models.PSPPayment{}, err
	}
	return models.PSPPayment{
		ParentReference:             from.Transfer.BalanceTransaction.ID,
		Reference:                   from.BalanceTransaction.ID,
		CreatedAt:                   time.Unix(from.Created, 0),
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      big.NewInt(from.Amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, strings.ToUpper(string(from.Currency))),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_REFUNDED,
		SourceAccountReference:      source,
		DestinationAccountReference: destination,
		Metadata:                    from.Metadata,
		Raw:                         raw,
	}, nil
}
