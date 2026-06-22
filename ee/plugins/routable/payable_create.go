package routable

import (
	"context"
	"errors"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
	errorsutils "github.com/formancehq/payments/pkg/domain/errors"
)

// initiatePayable is shared by createPayout and createTransfer. It
// returns the parsed Payable alongside the upstream HTTP status so
// callers can branch on 201 (sync, full payable) vs 202 (async, just
// {id}); see MAPPINGS.md §5.4.
func (p *Plugin) initiatePayable(ctx context.Context, pi models.PSPPaymentInitiation) (*client.Payable, int, error) {
	// Wrap validation errors so Temporal treats them as non-retriable.
	if err := validatePaymentInitiation(pi); err != nil {
		return nil, 0, errorsutils.NewWrappedError(err, models.ErrInvalidRequest)
	}
	if pi.SourceAccount == nil || pi.SourceAccount.Reference == "" {
		return nil, 0, errorsutils.NewWrappedError(
			errors.New("missing source account reference"), models.ErrInvalidRequest)
	}
	if pi.DestinationAccount == nil || pi.DestinationAccount.Reference == "" {
		return nil, 0, errorsutils.NewWrappedError(
			errors.New("missing destination account reference"), models.ErrInvalidRequest)
	}

	currencyCode, _, err := mappers.SplitAsset(pi.Asset)
	if err != nil {
		return nil, 0, errorsutils.NewWrappedError(
			fmt.Errorf("invalid asset %q: %w", pi.Asset, err), models.ErrInvalidRequest)
	}
	precision, err := mappers.PrecisionFor(currencyCode)
	if err != nil {
		return nil, 0, errorsutils.NewWrappedError(err, models.ErrInvalidRequest)
	}
	amount := mappers.FromMinorUnits(pi.Amount, precision)

	// Routable's v1 schema marks line_items[0].description as required;
	// always populate (override > PI description > synthesized fallback).
	lineDescription := mappers.FieldOr(pi.Metadata, mappers.MetadataKeyLineDescription, pi.Description)
	if lineDescription == "" {
		lineDescription = "Payment " + pi.Reference
	}

	// SendOn nil => JSON null => "send now". An explicit YYYY-MM-DD can
	// be wired through metadata for future-dated payables.
	var sendOn *string

	req := client.CreatePayableRequest{
		Type:                mappers.FieldOr(pi.Metadata, mappers.MetadataKeyType, mappers.DefaultPayableType),
		DeliveryMethod:      mappers.FieldOr(pi.Metadata, mappers.MetadataKeyDeliveryMethod, mappers.DefaultDeliveryMethod),
		PayToCompany:        pi.DestinationAccount.Reference,
		WithdrawFromAccount: pi.SourceAccount.Reference,
		Amount:              amount,
		CurrencyCode:        currencyCode,
		LineItems: []client.PayableLineItem{{
			UnitPrice:   amount,
			Amount:      amount,
			Quantity:    1,
			Description: lineDescription,
		}},
		SendOn:           sendOn,
		ActingTeamMember: mappers.FieldOr(pi.Metadata, mappers.MetadataKeyActingTeamMember, p.config.ActingTeamMember),
		Reference:        pi.Reference,
		ExternalID:       pi.Metadata[mappers.MetadataKeyExternalID],
		Message:          pi.Metadata[mappers.MetadataKeyMessage],
		IdempotencyKey:   pi.Reference,
	}

	// Debug-level: at 200k tx/wk this fires ~20×/min. Operators triage
	// per-payout from the engine's payment-initiation record.
	p.logger.Debugf("initiating routable payable: type=%s delivery=%s amount=%s %s reference=%s",
		req.Type, req.DeliveryMethod, req.Amount, req.CurrencyCode, req.Reference)
	payable, status, err := p.client.CreatePayable(ctx, req)
	if err != nil {
		return nil, status, err
	}
	// A 2xx with no ID is a Routable contract violation; surface it
	// rather than producing an empty PollingPayoutID.
	if payable == nil || payable.ID == "" {
		return nil, status, errors.New("routable returned an empty payable")
	}
	return payable, status, nil
}

func validatePaymentInitiation(pi models.PSPPaymentInitiation) error {
	if pi.Reference == "" {
		return errors.New("missing payment initiation reference")
	}
	if pi.Amount == nil {
		return errors.New("missing payment initiation amount")
	}
	if pi.Asset == "" {
		return errors.New("missing payment initiation asset")
	}
	return nil
}
