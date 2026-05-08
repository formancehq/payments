package routable

import (
	"context"
	"errors"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// initiatePayable is the shared CreatePayable plumbing used by both
// CreateTransfer and CreatePayout. Both flows produce a Routable
// payable; the only thing that differs at the call site is which engine
// response envelope wraps the result. Returns the parsed Payable
// alongside the upstream HTTP status so callers can branch cleanly on
// 201 (sync, full payable) vs 202 (async, just {id}).
func (p *Plugin) initiatePayable(ctx context.Context, pi models.PSPPaymentInitiation) (*client.Payable, int, error) {
	// Validation lives at the choke point so neither createPayout nor
	// createTransfer needs to remember to call validatePaymentInitiation
	// first. CodeRabbit rightly flagged the previous caller-side
	// duplication.
	if err := validatePaymentInitiation(pi); err != nil {
		return nil, 0, err
	}
	if pi.SourceAccount == nil || pi.SourceAccount.Reference == "" {
		return nil, 0, errors.New("missing source account reference")
	}
	if pi.DestinationAccount == nil || pi.DestinationAccount.Reference == "" {
		return nil, 0, errors.New("missing destination account reference")
	}

	currencyCode, _, err := mappers.SplitAsset(pi.Asset)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid asset %q: %w", pi.Asset, err)
	}
	precision, err := mappers.PrecisionFor(currencyCode)
	if err != nil {
		return nil, 0, err
	}
	amount := mappers.FromMinorUnits(pi.Amount, precision)

	// Routable's v1 schema marks line_items[0].description as required,
	// so we always populate it: caller-supplied override > PI description
	// > a synthesized fallback referencing the PI reference.
	lineDescription := mappers.FieldOr(pi.Metadata, mappers.MetadataKeyLineDescription, pi.Description)
	if lineDescription == "" {
		lineDescription = "Payment " + pi.Reference
	}

	// SendOn is required by Routable's v1 schema even when sending
	// immediately. nil pointer => JSON null => "send now". An explicit
	// YYYY-MM-DD value can be supplied via metadata for future-dated
	// payables once we wire that key.
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
		IdempotencyKey:   pi.Reference,
	}

	// Per-payout init log. At 200k tx/wk this fires ~20×/min sustained;
	// keep at Debug to avoid log-volume blow-up. Operators triage
	// individual payouts via the engine's payment-initiation record
	// (which already carries pi.Reference); error paths below stay at
	// Info/Error level for the genuinely interesting events.
	p.logger.Debugf("initiating routable payable: type=%s delivery=%s amount=%s %s reference=%s",
		req.Type, req.DeliveryMethod, req.Amount, req.CurrencyCode, req.Reference)
	payable, status, err := p.client.CreatePayable(ctx, req)
	if err != nil {
		return nil, status, err
	}
	// Defensive: a non-error response with no ID is a Routable contract
	// violation. Surface explicitly rather than letting a downstream
	// nil-deref or empty PollingPayoutID propagate.
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
