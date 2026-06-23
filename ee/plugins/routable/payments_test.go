package routable

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Routable fetchNextPayments", func() {
	var (
		ctrl   *gomock.Controller
		mock   *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mock = client.NewMockClient(ctrl)
		plg = &Plugin{Plugin: plugins.NewBasePlugin(), name: "routable", logger: logger, client: mock}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("emits payables as PAYOUT and switches phase to receivables when exhausted", func(ctx SpecContext) {
		now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
		mock.EXPECT().ListPayables(gomock.Any(), 1, 50, time.Time{}).Return(&client.ListPayablesResponse{
			Results: []client.Payable{{
				ID:                  "pa_1",
				Status:              "completed",
				Amount:              "10.00",
				CurrencyCode:        "USD",
				DeliveryMethod:      "ach_standard",
				PayToCompany:        &client.PayableCompany{ID: "co_1"},
				WithdrawFromAccount: &client.PayableAccount{ID: "acc_1"},
				StatusChangedAt:     &now,
				CreatedAt:           now,
			}},
		}, nil)

		resp, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50})
		Expect(err).To(BeNil())
		Expect(resp.Payments).To(HaveLen(1))
		Expect(resp.Payments[0].Reference).To(Equal("pa_1"))
		Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYOUT))
		Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
		Expect(resp.Payments[0].Scheme).To(Equal(models.PAYMENT_SCHEME_ACH))
		Expect(*resp.Payments[0].SourceAccountReference).To(Equal("acc_1"))
		Expect(*resp.Payments[0].DestinationAccountReference).To(Equal("co_1"))
		Expect(resp.HasMore).To(BeTrue()) // moves on to receivables phase

		var state paymentsState
		Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
		Expect(state.Phase).To(Equal(phaseReceivables))
		// CycleLowerBound stays put for receivables; only CycleMaxSeen advances.
		Expect(state.CycleLowerBound.IsZero()).To(BeTrue())
		Expect(state.CycleMaxSeen.Equal(now)).To(BeTrue())
	})

	It("commits CycleMaxSeen as next CycleLowerBound only after receivables exhausts", func(ctx SpecContext) {
		now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
		mock.EXPECT().ListReceivables(gomock.Any(), 1, 50, time.Time{}).Return(&client.ListReceivablesResponse{
			Results: []client.Receivable{{
				ID:               "re_1",
				Status:           "pending",
				Amount:           "5.00",
				CurrencyCode:     "USD",
				DeliveryMethod:   "ach_standard",
				PayFromCompany:   &client.ReceivableCompany{ID: "co_42"},
				DepositToAccount: &client.ReceivableAccount{ID: "acc_99"},
				CreatedAt:        now,
			}},
		}, nil)

		// Mid-cycle state: receivables phase, payables already advanced
		// CycleMaxSeen but CycleLowerBound is still zero from the very first
		// run. Receivables exhausts, so we expect the cycle to commit.
		incoming, _ := json.Marshal(paymentsState{
			Phase:        phaseReceivables,
			Page:         1,
			CycleMaxSeen: now,
		})
		resp, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: incoming})
		Expect(err).To(BeNil())
		Expect(resp.Payments).To(HaveLen(1))
		Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
		Expect(resp.HasMore).To(BeFalse())

		var state paymentsState
		Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
		Expect(state.Phase).To(Equal(phasePayables))
		Expect(state.Page).To(Equal(1))
		Expect(state.CycleLowerBound.Equal(now)).To(BeTrue(), "CycleMaxSeen must be promoted to CycleLowerBound on cycle commit")
		Expect(state.CycleMaxSeen.IsZero()).To(BeTrue(), "CycleMaxSeen must reset for the next cycle")
	})

	It("holds CycleLowerBound immutable across multi-page payables", func(ctx SpecContext) {
		floor := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		later := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)

		// Page 1 advances CycleMaxSeen to `later` and reports HasMore.
		mock.EXPECT().ListPayables(gomock.Any(), 1, 50, floor).Return(&client.ListPayablesResponse{
			Results: []client.Payable{{
				ID: "pa_p1", Status: "pending", Amount: "1.00", CurrencyCode: "USD",
				StatusChangedAt: &later, CreatedAt: later,
			}},
			Links: client.Links{Next: "/v1/payables?page=2"},
		}, nil)

		// Page 2 MUST be requested with the SAME floor as page 1 — never
		// the page-1 max. This is the regression CodeRabbit caught.
		mock.EXPECT().ListPayables(gomock.Any(), 2, 50, floor).Return(&client.ListPayablesResponse{}, nil)

		state, _ := json.Marshal(paymentsState{Page: 1, CycleLowerBound: floor})
		resp, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: state})
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeTrue())

		var nextState paymentsState
		Expect(json.Unmarshal(resp.NewState, &nextState)).To(Succeed())
		Expect(nextState.Page).To(Equal(2))
		Expect(nextState.CycleLowerBound.Equal(floor)).To(BeTrue())
		Expect(nextState.CycleMaxSeen.Equal(later)).To(BeTrue())

		// Issue page 2 with the state we just produced.
		_, err = plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: resp.NewState})
		Expect(err).To(BeNil())
	})

	It("uses the same floor for receivables as for payables in the same cycle", func(ctx SpecContext) {
		floor := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		later := time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC)

		// Receivables phase must call ListReceivables with the floor, not
		// with the CycleMaxSeen produced by the payables phase. Otherwise
		// receivables that changed between `floor` and `later` are skipped.
		mock.EXPECT().ListReceivables(gomock.Any(), 1, 50, floor).Return(&client.ListReceivablesResponse{}, nil)

		state, _ := json.Marshal(paymentsState{
			Phase:           phaseReceivables,
			Page:            1,
			CycleLowerBound: floor,
			CycleMaxSeen:    later,
		})
		_, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: state})
		Expect(err).To(BeNil())
	})

	It("preserves CycleLowerBound when both phases of a cycle return no rows", func(ctx SpecContext) {
		// Regression for the empty-cycle bug: at high write throughput a
		// 200k/wk connector may legitimately see a cycle with zero status
		// transitions for both payables and receivables. Promoting
		// CycleMaxSeen=0 to CycleLowerBound would regress the floor to
		// epoch and trigger a full historical refetch on the next cycle.
		previousFloor := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

		// Receivables phase, both pages empty: this is the cycle-end branch
		// we're guarding.
		mock.EXPECT().ListReceivables(gomock.Any(), 1, 50, previousFloor).Return(&client.ListReceivablesResponse{}, nil)

		state, _ := json.Marshal(paymentsState{
			Phase:           phaseReceivables,
			Page:            1,
			CycleLowerBound: previousFloor,
			// CycleMaxSeen left at zero — payables phase saw nothing either.
		})
		resp, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: state})
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeFalse())

		var nextState paymentsState
		Expect(json.Unmarshal(resp.NewState, &nextState)).To(Succeed())
		Expect(nextState.Phase).To(Equal(phasePayables))
		Expect(nextState.CycleLowerBound.Equal(previousFloor)).To(BeTrue(),
			"empty cycle must NOT regress CycleLowerBound to zero")
		Expect(nextState.CycleMaxSeen.IsZero()).To(BeTrue())
	})

	It("preserves CycleLowerBound when only payables saw rows but receivables didn't", func(ctx SpecContext) {
		previousFloor := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		payableSeen := time.Date(2025, 6, 3, 0, 0, 0, 0, time.UTC)

		mock.EXPECT().ListReceivables(gomock.Any(), 1, 50, previousFloor).Return(&client.ListReceivablesResponse{}, nil)

		// Payables phase already advanced CycleMaxSeen to payableSeen.
		state, _ := json.Marshal(paymentsState{
			Phase:           phaseReceivables,
			Page:            1,
			CycleLowerBound: previousFloor,
			CycleMaxSeen:    payableSeen,
		})
		resp, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: state})
		Expect(err).To(BeNil())

		var nextState paymentsState
		Expect(json.Unmarshal(resp.NewState, &nextState)).To(Succeed())
		Expect(nextState.CycleLowerBound.Equal(payableSeen)).To(BeTrue(),
			"non-zero CycleMaxSeen from payables phase must still promote at cycle end")
	})

	// Resumable: paymentsState round-trips through JSON cleanly so that a
	// worker crash mid-cycle resumes at the correct page with the same
	// CycleLowerBound. At 200k tx/wk the cycle has many pages; losing
	// state would cause both duplicates AND skips on resume.
	It("survives JSON round-trip mid-cycle without losing the floor", func(ctx SpecContext) {
		floor := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		later := time.Date(2025, 6, 4, 0, 0, 0, 0, time.UTC)

		mid := paymentsState{Phase: phaseReceivables, Page: 7, CycleLowerBound: floor, CycleMaxSeen: later}
		raw, err := json.Marshal(mid)
		Expect(err).To(BeNil())

		decoded, err := decodePaymentsState(raw)
		Expect(err).To(BeNil())
		Expect(decoded.Phase).To(Equal(phaseReceivables))
		Expect(decoded.Page).To(Equal(7))
		Expect(decoded.CycleLowerBound.Equal(floor)).To(BeTrue())
		Expect(decoded.CycleMaxSeen.Equal(later)).To(BeTrue())
	})

	// Tiebreaker / lossless: rows whose status_changed_at equals the
	// cycle floor are re-emitted at every cycle boundary because
	// Routable's gte filter is inclusive. We rely on the engine's
	// PSPPayment.Reference dedup; the cost is at-most-N replays of the
	// boundary rows, never lost rows. This test pins the contract by
	// driving two full cycles and asserting the boundary row keeps the
	// same Reference each time, so engine dedup catches the replay.
	It("re-emits boundary rows with the same Reference across cycles for engine dedup", func(ctx SpecContext) {
		floor := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
		boundaryRow := func() client.Receivable {
			return client.Receivable{
				ID:              "re_boundary",
				Status:          "completed",
				Amount:          "1.00",
				CurrencyCode:    "USD",
				DeliveryMethod:  "ach_standard",
				StatusChangedAt: &floor,
				CreatedAt:       floor,
			}
		}

		// Cycle 1 — receivables phase, ends with boundary row.
		mock.EXPECT().ListReceivables(gomock.Any(), 1, 50, floor).Return(&client.ListReceivablesResponse{
			Results: []client.Receivable{boundaryRow()},
		}, nil)

		state, _ := json.Marshal(paymentsState{Phase: phaseReceivables, Page: 1, CycleLowerBound: floor})
		c1, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: state})
		Expect(err).To(BeNil())
		Expect(c1.Payments).To(HaveLen(1))
		Expect(c1.Payments[0].Reference).To(Equal("re_boundary"))
		Expect(c1.HasMore).To(BeFalse(), "cycle ended")

		// Cycle 2 starts in payables phase (empty in this scenario).
		mock.EXPECT().ListPayables(gomock.Any(), 1, 50, floor).Return(&client.ListPayablesResponse{}, nil)
		c2payables, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: c1.NewState})
		Expect(err).To(BeNil())
		Expect(c2payables.HasMore).To(BeTrue(), "transitions to receivables phase")

		// Cycle 2 receivables — same boundary row reappears (gte-inclusive).
		mock.EXPECT().ListReceivables(gomock.Any(), 1, 50, floor).Return(&client.ListReceivablesResponse{
			Results: []client.Receivable{boundaryRow()},
		}, nil)
		c2recv, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: c2payables.NewState})
		Expect(err).To(BeNil())
		Expect(c2recv.Payments).To(HaveLen(1))
		Expect(c2recv.Payments[0].Reference).To(Equal("re_boundary"),
			"engine dedup keys on PSPPayment.Reference; boundary row must keep its identity across cycles")
	})

	It("migrates legacy LastSeenAt state to CycleLowerBound", func(ctx SpecContext) {
		legacy := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		mock.EXPECT().ListPayables(gomock.Any(), 1, 50, legacy).Return(&client.ListPayablesResponse{}, nil)

		legacyState := json.RawMessage(`{"phase":"","page":1,"lastSeenAt":"2024-12-31T00:00:00Z"}`)
		resp, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: legacyState})
		Expect(err).To(BeNil())

		var migrated paymentsState
		Expect(json.Unmarshal(resp.NewState, &migrated)).To(Succeed())
		Expect(migrated.CycleLowerBound.Equal(legacy)).To(BeTrue())
		Expect(migrated.LastSeenAt.IsZero()).To(BeTrue(), "deprecated LastSeenAt must be cleared after migration")
	})
})
