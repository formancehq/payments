package routable

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// paymentsState pages through payables and then receivables under a single
// cycle. The two timestamps capture the cursor invariant the previous
// design got wrong:
//
//   - CycleLowerBound is the status_changed_at.gte floor used by every
//     payables and receivables request in the current cycle. It is held
//     IMMUTABLE for the cycle's full duration. Mutating it mid-cycle (the
//     old behaviour) caused page=2 to use a tighter lower bound than
//     page=1, so any row that landed between the two timestamps but only
//     appeared on a later page would be silently dropped.
//   - CycleMaxSeen accumulates the latest status_changed_at observed
//     across every page of the cycle. It is never used to drive a
//     request. Once receivables exhausts, it is promoted to
//     CycleLowerBound for the next cycle and reset.
//
// Routable's status_changed_at.gte filter is inclusive, so rows whose
// timestamp equals the cycle floor get re-emitted at every cycle boundary.
// The engine framework dedupes by PSPPayment.Reference, so this is wasted
// traffic but never a correctness problem. A `(timestamp, id)` tiebreaker
// would eliminate the replay; out of scope for this PR.
//
// The legacy LastSeenAt field is kept on the wire so existing persisted
// state migrates cleanly: on first load we promote it to CycleLowerBound.
type paymentsState struct {
	Phase           paymentsPhase `json:"phase"`
	Page            int           `json:"page"`
	CycleLowerBound time.Time     `json:"cycleLowerBound,omitempty"`
	CycleMaxSeen    time.Time     `json:"cycleMaxSeen,omitempty"`

	// Deprecated: pre-cycle-immutable state. Promoted to CycleLowerBound
	// on first decode after the upgrade and zeroed thereafter.
	LastSeenAt time.Time `json:"lastSeenAt,omitempty"`
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
// (PAYIN) into a single PSPPayment stream. A cycle paginates payables to
// completion, then receivables to completion, then commits the cycle's
// max-seen timestamp as the next cycle's lower bound.
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
	resp, err := p.client.ListPayables(ctx, state.nextPage(), req.PageSize, state.CycleLowerBound)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("listing payables (page=%d): %w", state.nextPage(), err)
	}

	payments, maxSeen := mappers.PayablesToPSPPayments(resp.Results, state.CycleMaxSeen, func(id string, err error) {
		p.logger.Infof("skipping payable %s: %v", id, err)
	})

	next := state
	next.CycleMaxSeen = maxSeen
	if resp.Links.HasMore() {
		next.Page = state.nextPage() + 1
	} else {
		// payables exhausted for this cycle: switch to receivables on
		// the next call. CycleLowerBound stays put — receivables must
		// use the SAME floor as payables did, otherwise we skip
		// receivables that changed between the cycle's start and the
		// latest payable seen.
		next.Phase = phaseReceivables
		next.Page = 1
	}

	payload, err := json.Marshal(next)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("marshaling state: %w", err)
	}
	return models.FetchNextPaymentsResponse{Payments: payments, NewState: payload, HasMore: true}, nil
}

func (p *Plugin) fetchReceivablesPage(ctx context.Context, req models.FetchNextPaymentsRequest, state paymentsState) (models.FetchNextPaymentsResponse, error) {
	resp, err := p.client.ListReceivables(ctx, state.nextPage(), req.PageSize, state.CycleLowerBound)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("listing receivables (page=%d): %w", state.nextPage(), err)
	}

	payments, maxSeen := mappers.ReceivablesToPSPPayments(resp.Results, state.CycleMaxSeen, func(id string, err error) {
		p.logger.Infof("skipping receivable %s: %v", id, err)
	})

	next := state
	next.CycleMaxSeen = maxSeen
	hasMore := true
	if resp.Links.HasMore() {
		next.Page = state.nextPage() + 1
	} else {
		// Cycle complete: promote CycleMaxSeen to CycleLowerBound for
		// the next cycle, reset Phase and Page. HasMore=false ends the
		// run.
		// Empty-cycle guard: if we saw no rows in BOTH phases this
		// cycle, maxSeen is zero. Promoting that would regress the
		// floor and trigger a full historical refetch on the next
		// cycle. Preserve the previous CycleLowerBound instead.
		nextLowerBound := maxSeen
		if nextLowerBound.IsZero() {
			nextLowerBound = state.CycleLowerBound
		}
		next = paymentsState{
			Phase:           phasePayables,
			Page:            1,
			CycleLowerBound: nextLowerBound,
		}
		hasMore = false
	}

	payload, err := json.Marshal(next)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("marshaling state: %w", err)
	}
	return models.FetchNextPaymentsResponse{Payments: payments, NewState: payload, HasMore: hasMore}, nil
}

func decodePaymentsState(raw json.RawMessage) (paymentsState, error) {
	var s paymentsState
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("decoding payments state: %w", err)
	}
	// Migrate legacy state (pre cycle-immutable cursor): promote
	// LastSeenAt to CycleLowerBound on first decode and zero it. Drops
	// the deprecated field from subsequent serializations without losing
	// the watermark.
	if s.CycleLowerBound.IsZero() && !s.LastSeenAt.IsZero() {
		s.CycleLowerBound = s.LastSeenAt
	}
	s.LastSeenAt = time.Time{}
	return s, nil
}
