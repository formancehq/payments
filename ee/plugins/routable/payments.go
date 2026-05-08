package routable

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// paymentsState carries the cycle cursor across fetcher invocations.
// The three invariants it enforces — CycleLowerBound immutable for the
// duration of a cycle, CycleMaxSeen as a write-only watermark, and
// boundary-row replay tolerated via engine-side Reference dedup — are
// documented in MAPPINGS.md §3.6. The previous mid-cycle mutation
// pattern silently dropped rows; the test suite pins the regression.
//
// LastSeenAt is kept on the wire only to migrate persisted state from
// the previous design — on first decode it is promoted to
// CycleLowerBound and zeroed.
type paymentsState struct {
	Phase           paymentsPhase `json:"phase"`
	Page            int           `json:"page"`
	CycleLowerBound time.Time     `json:"cycleLowerBound,omitempty"`
	CycleMaxSeen    time.Time     `json:"cycleMaxSeen,omitempty"`

	// Deprecated: pre-cycle-immutable state. See decodePaymentsState.
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
		// Switch to receivables on the next call. CycleLowerBound
		// stays put: receivables must use the SAME floor as payables.
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
		// the next cycle. Empty-cycle guard: if both phases saw no
		// rows, maxSeen is zero — preserve the previous floor instead
		// of regressing to epoch. (Pinned by the empty-cycle test.)
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
	// Migrate legacy LastSeenAt → CycleLowerBound on first decode.
	if s.CycleLowerBound.IsZero() && !s.LastSeenAt.IsZero() {
		s.CycleLowerBound = s.LastSeenAt
	}
	s.LastSeenAt = time.Time{}
	return s, nil
}
