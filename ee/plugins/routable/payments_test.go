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
		Expect(state.LastSeenAt.Equal(now)).To(BeTrue())
	})

	It("emits receivables as PAYIN and resets the cycle when both phases are done", func(ctx SpecContext) {
		mock.EXPECT().ListReceivables(gomock.Any(), 1, 50, gomock.Any()).Return(&client.ListReceivablesResponse{
			Results: []client.Receivable{{
				ID:               "re_1",
				Status:           "pending",
				Amount:           "5.00",
				CurrencyCode:     "USD",
				DeliveryMethod:   "ach_standard",
				PayFromCompany:   &client.ReceivableCompany{ID: "co_42"},
				DepositToAccount: &client.ReceivableAccount{ID: "acc_99"},
				CreatedAt:        time.Now().UTC(),
			}},
		}, nil)

		req := models.FetchNextPaymentsRequest{
			PageSize: 50,
			State:    json.RawMessage(`{"phase":"receivables","page":1}`),
		}
		resp, err := plg.fetchNextPayments(ctx, req)
		Expect(err).To(BeNil())
		Expect(resp.Payments).To(HaveLen(1))
		Expect(resp.Payments[0].Type).To(Equal(models.PAYMENT_TYPE_PAYIN))
		Expect(resp.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_PENDING))
		Expect(resp.HasMore).To(BeFalse())

		var state paymentsState
		Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
		Expect(state.Phase).To(Equal(phasePayables))
		Expect(state.Page).To(Equal(1))
	})

	It("propagates the LastSeenAt watermark as status_changed_at.gte to Routable", func(ctx SpecContext) {
		from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		mock.EXPECT().ListPayables(gomock.Any(), 1, 50, from).Return(&client.ListPayablesResponse{}, nil)
		state, _ := json.Marshal(paymentsState{Page: 1, LastSeenAt: from})
		_, err := plg.fetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 50, State: state})
		Expect(err).To(BeNil())
	})
})
