package routable

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/routable/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency/precision from asset: %w", err)
	}
	amountStr, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to format amount: %w", err)
	}

	vendorID := pi.DestinationAccount.Reference
	sourceAccountID := pi.SourceAccount.Reference

	ptype := "ach"
	delivery := "ach_standard"
	if v, ok := pi.Metadata["spec.formance.com/routable.delivery_method"]; ok && v != "" {
		delivery = v
	}
	if v, ok := pi.Metadata["spec.formance.com/routable.type"]; ok && v != "" {
		ptype = v
	}
	payToPaymentMethod := pi.Metadata["spec.formance.com/routable.pay_to_payment_method"]

	payload := &client.PayoutRequest{
		Type:                ptype,
		ActingTeamMember:    p.actingTeamMemberID,
		PayToCompany:        vendorID,
		WithdrawFromAccount: sourceAccountID,
		CurrencyCode:        curr,
		Amount:              amountStr,
		SendOn:              nil, // ready_to_send
		DeliveryMethod:      delivery,
		PayToPaymentMethod:  payToPaymentMethod,
		LineItems: []client.NewPayableLineItem{{
			UnitPrice:   amountStr,
			Description: pi.Description,
			Quantity:    "1",
			Amount:      amountStr,
		}},
	}

	resp, err := p.client.InitiatePayout(ctx, payload)
	if err != nil {
		return nil, err
	}

	return payoutToPayment(resp, pi)
}

func mapRoutableStatusToPaymentStatus(s string) models.PaymentStatus {
	switch s {
	case "completed":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "failed", "canceled":
		return models.PAYMENT_STATUS_FAILED
	default:
		return models.PAYMENT_STATUS_PENDING
	}
}

func payoutToPayment(from *client.PayoutResponse, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	raw, _ := json.Marshal(from)
	createdAt := from.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	status := mapRoutableStatusToPaymentStatus(from.Status)
	pay := &models.PSPPayment{
		ParentReference:             "",
		Reference:                   from.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      pi.Amount,
		Asset:                       pi.Asset,
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      status,
		SourceAccountReference:      nil,
		DestinationAccountReference: nil,
		Metadata:                    map[string]string{"spec.formance.com/generic_provider": ProviderName},
		Raw:                         raw,
	}
	if pi.SourceAccount != nil {
		pay.SourceAccountReference = &pi.SourceAccount.Reference
	}
	if pi.DestinationAccount != nil {
		ref := pi.DestinationAccount.Reference
		pay.DestinationAccountReference = &ref
	}
	return pay, nil
}
