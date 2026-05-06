package routable

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
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
