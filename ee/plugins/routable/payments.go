package routable

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// paymentsState extends the shared pageState to interleave payables and
// receivables under a single watermark cursor. A cycle paginates payables
// to completion, then receivables, then advances the watermark to whichever
// last status_changed_at we observed.
type paymentsState struct {
	Phase      paymentsPhase `json:"phase"`
	Page       int           `json:"page"`
	LastSeenAt time.Time     `json:"lastSeenAt,omitempty"`
}

type paymentsPhase string

const (
	phasePayables    paymentsPhase = ""
	phaseReceivables paymentsPhase = "receivables"
)

func (s paymentsState) nextPage() int {
	if s.Page <= 0 {
		return 1
	}
	return s.Page
}

// fetchNextPayments merges Routable payables (PAYOUT) and receivables
// (PAYIN) into a single PSPPayment stream. We always page payables first
// in a cycle, then switch to receivables, then reset for the next cycle.
// LastSeenAt is the upper bound of the most recent status_changed_at we
// emitted, so the next call can pass it as status_changed_at.gte to
// restrict the API surface.
func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	state, err := decodePaymentsState(req.State)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	switch state.Phase {
	case phaseReceivables:
		return p.fetchReceivablesPage(ctx, req, state)
	default:
		return p.fetchPayablesPage(ctx, req, state)
	}
}

func (p *Plugin) fetchPayablesPage(ctx context.Context, req models.FetchNextPaymentsRequest, state paymentsState) (models.FetchNextPaymentsResponse, error) {
	resp, err := p.client.ListPayables(ctx, state.nextPage(), req.PageSize, state.LastSeenAt)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("listing payables (page=%d): %w", state.nextPage(), err)
	}

	payments, watermark := p.payablesToPSPPayments(resp.Results, state.LastSeenAt)

	next := state
	next.LastSeenAt = watermark
	hasMore := true
	if resp.Links.HasMore() {
		next.Page = state.nextPage() + 1
	} else {
		// payables exhausted for this cycle: switch to receivables on the
		// next call. The status_changed_at watermark carries over.
		next.Phase = phaseReceivables
		next.Page = 1
	}

	payload, err := json.Marshal(next)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("marshaling state: %w", err)
	}
	return models.FetchNextPaymentsResponse{Payments: payments, NewState: payload, HasMore: hasMore}, nil
}

func (p *Plugin) fetchReceivablesPage(ctx context.Context, req models.FetchNextPaymentsRequest, state paymentsState) (models.FetchNextPaymentsResponse, error) {
	resp, err := p.client.ListReceivables(ctx, state.nextPage(), req.PageSize, state.LastSeenAt)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("listing receivables (page=%d): %w", state.nextPage(), err)
	}

	payments, watermark := p.receivablesToPSPPayments(resp.Results, state.LastSeenAt)

	next := state
	next.LastSeenAt = watermark
	hasMore := true
	if resp.Links.HasMore() {
		next.Page = state.nextPage() + 1
	} else {
		// Receivables exhausted: the cycle is complete. Reset to payables
		// for the next polling tick. HasMore=false ends the current run.
		next.Phase = phasePayables
		next.Page = 1
		hasMore = false
	}

	payload, err := json.Marshal(next)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("marshaling state: %w", err)
	}
	return models.FetchNextPaymentsResponse{Payments: payments, NewState: payload, HasMore: hasMore}, nil
}

// payablesToPSPPayments maps Routable payables onto PSPPayments and tracks
// the latest status_changed_at observed so the cursor can advance.
func (p *Plugin) payablesToPSPPayments(in []client.Payable, watermark time.Time) ([]models.PSPPayment, time.Time) {
	out := make([]models.PSPPayment, 0, len(in))
	for _, pa := range in {
		payment, err := p.payableToPSPPayment(pa)
		if err != nil {
			p.logger.Infof("skipping payable %s: %v", pa.ID, err)
			continue
		}
		out = append(out, payment)
		watermark = laterOf(watermark, statusChangedAtOrCreated(pa.StatusChangedAt, pa.CreatedAt))
	}
	return out, watermark
}

func (p *Plugin) receivablesToPSPPayments(in []client.Receivable, watermark time.Time) ([]models.PSPPayment, time.Time) {
	out := make([]models.PSPPayment, 0, len(in))
	for _, r := range in {
		payment, err := p.receivableToPSPPayment(r)
		if err != nil {
			p.logger.Infof("skipping receivable %s: %v", r.ID, err)
			continue
		}
		out = append(out, payment)
		watermark = laterOf(watermark, statusChangedAtOrCreated(r.StatusChangedAt, r.CreatedAt))
	}
	return out, watermark
}

func (p *Plugin) payableToPSPPayment(pa client.Payable) (models.PSPPayment, error) {
	raw, err := json.Marshal(pa)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("marshaling raw: %w", err)
	}
	precision, err := precisionFor(pa.CurrencyCode)
	if err != nil {
		return models.PSPPayment{}, err
	}
	amount, err := toMinorUnits(pa.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("parsing amount: %w", err)
	}

	payment := models.PSPPayment{
		Reference: pa.ID,
		CreatedAt: pa.CreatedAt,
		Type:      models.PAYMENT_TYPE_PAYOUT,
		Amount:    amount,
		Asset:     formatAsset(pa.CurrencyCode),
		Scheme:    deliveryMethodToScheme(pa.DeliveryMethod),
		Status:    payableStatus(pa.Status),
		Metadata:  payableMetadata(pa),
		Raw:       raw,
	}
	if pa.WithdrawFromAccount != nil && pa.WithdrawFromAccount.ID != "" {
		ref := pa.WithdrawFromAccount.ID
		payment.SourceAccountReference = &ref
	}
	if pa.PayToCompany != nil && pa.PayToCompany.ID != "" {
		ref := pa.PayToCompany.ID
		payment.DestinationAccountReference = &ref
	}
	return payment, nil
}

func (p *Plugin) receivableToPSPPayment(r client.Receivable) (models.PSPPayment, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("marshaling raw: %w", err)
	}
	precision, err := precisionFor(r.CurrencyCode)
	if err != nil {
		return models.PSPPayment{}, err
	}
	amount, err := toMinorUnits(r.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("parsing amount: %w", err)
	}

	payment := models.PSPPayment{
		Reference: r.ID,
		CreatedAt: r.CreatedAt,
		Type:      models.PAYMENT_TYPE_PAYIN,
		Amount:    amount,
		Asset:     formatAsset(r.CurrencyCode),
		Scheme:    deliveryMethodToScheme(r.DeliveryMethod),
		Status:    payableStatus(r.Status),
		Metadata:  receivableMetadata(r),
		Raw:       raw,
	}
	if r.PayFromCompany != nil && r.PayFromCompany.ID != "" {
		ref := r.PayFromCompany.ID
		payment.SourceAccountReference = &ref
	}
	if r.DepositToAccount != nil && r.DepositToAccount.ID != "" {
		ref := r.DepositToAccount.ID
		payment.DestinationAccountReference = &ref
	}
	return payment, nil
}

// laterOf returns whichever of a or b is later (or zero when both are zero).
func laterOf(a, b time.Time) time.Time {
	if a.IsZero() {
		return b
	}
	if b.IsZero() || a.After(b) {
		return a
	}
	return b
}

// statusChangedAtOrCreated picks status_changed_at when set, otherwise the
// created_at — Routable can return a nil status_changed_at on draft rows.
func statusChangedAtOrCreated(statusChangedAt *time.Time, createdAt time.Time) time.Time {
	if statusChangedAt != nil && !statusChangedAt.IsZero() {
		return *statusChangedAt
	}
	return createdAt
}

func decodePaymentsState(raw json.RawMessage) (paymentsState, error) {
	var s paymentsState
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("decoding payments state: %w", err)
	}
	return s, nil
}
