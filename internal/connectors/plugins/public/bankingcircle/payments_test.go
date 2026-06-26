package bankingcircle

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("BankingCircle Plugin Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next accounts", func() {
		var (
			samplePayments []client.Payment
			now            time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			samplePayments = make([]client.Payment, 0)
			for i := 0; i < 50; i++ {
				samplePayments = append(samplePayments, client.Payment{
					PaymentID:                    fmt.Sprint(i),
					TransactionReference:         fmt.Sprintf("transaction-%d", i),
					ConcurrencyToken:             "",
					Classification:               "",
					Status:                       "Processed",
					Errors:                       nil,
					ProcessedTimestamp:           now.Add(-time.Duration(50-i) * time.Minute).UTC(),
					LatestStatusChangedTimestamp: now.Add(-time.Duration(50-i) * time.Minute).UTC(),
					DebtorInformation: client.DebtorInformation{
						AccountID: "123",
					},
					Transfer: client.Transfer{
						Amount: client.Amount{
							Currency: "EUR",
							Amount:   "120",
						},
					},
					CreditorInformation: client.CreditorInformation{
						AccountID: "321",
					},
				})
			}
		})

		It("should return an error - get payments error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetPayments(gomock.Any(), 1, 60).Return(
				[]client.Payment{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch next payments - no state no results", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetPayments(gomock.Any(), 1, 60).Return(
				[]client.Payment{},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LatestStatusChangedTimestamp.IsZero()).To(BeTrue())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetPayments(gomock.Any(), 1, 60).Return(
				samplePayments,
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(50))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LatestStatusChangedTimestamp.UTC()).To(Equal(samplePayments[49].LatestStatusChangedTimestamp.UTC()))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetPayments(gomock.Any(), 1, 40).Return(
				samplePayments[:40],
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LatestStatusChangedTimestamp.UTC()).To(Equal(samplePayments[39].LatestStatusChangedTimestamp.UTC()))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State: []byte(fmt.Sprintf(
					`{"latestStatusChangedTimestamp": "%s", "latestProcessedID": "%s"}`,
					samplePayments[38].LatestStatusChangedTimestamp.UTC().Format(time.RFC3339Nano),
					samplePayments[38].PaymentID,
				)),
				PageSize: 40,
			}

			m.EXPECT().GetPayments(gomock.Any(), 1, 40).Return(
				samplePayments[:40],
				nil,
			)

			m.EXPECT().GetPayments(gomock.Any(), 2, 40).Return(
				samplePayments[40:],
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(11))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LatestStatusChangedTimestamp.UTC()).To(Equal(samplePayments[49].LatestStatusChangedTimestamp.UTC()))
			Expect(state.LatestProcessedID).To(Equal(samplePayments[49].PaymentID))
		})

		It("keeps distinct payments that share the watermark timestamp (M-CON2)", func(ctx SpecContext) {
			ts := now.Add(-time.Hour).UTC()
			sameSecond := make([]client.Payment, 0, 3)
			for _, id := range []string{"a", "b", "c"} {
				sameSecond = append(sameSecond, client.Payment{
					PaymentID:                    id,
					Status:                       "Processed",
					ProcessedTimestamp:           ts,
					LatestStatusChangedTimestamp: ts,
					DebtorInformation:            client.DebtorInformation{AccountID: "123"},
					Transfer:                     client.Transfer{Amount: client.Amount{Currency: "EUR", Amount: "120"}},
				})
			}

			// Watermark sits exactly on the shared timestamp, with "a" already processed.
			req := models.FetchNextPaymentsRequest{
				State: []byte(fmt.Sprintf(
					`{"latestStatusChangedTimestamp": "%s", "latestProcessedID": "a"}`,
					ts.Format(time.RFC3339Nano),
				)),
				PageSize: 40,
			}

			m.EXPECT().GetPayments(gomock.Any(), 1, 40).Return(sameSecond, nil)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			// "a" is the already-processed boundary row; "b" and "c" share its
			// timestamp and must NOT be dropped.
			Expect(resp.Payments).To(HaveLen(2))
			refs := []string{resp.Payments[0].Reference, resp.Payments[1].Reference}
			Expect(refs).To(ConsistOf("b", "c"))
		})

		It("re-emits the boundary payment once when migrating state without latestProcessedID", func(ctx SpecContext) {
			// Old-format state: watermark only, no latestProcessedID. The row at
			// exactly the watermark is re-emitted once (idempotent via upsert);
			// no recrawl.
			req := models.FetchNextPaymentsRequest{
				State:    []byte(fmt.Sprintf(`{"latestStatusChangedTimestamp": "%s"}`, samplePayments[38].LatestStatusChangedTimestamp.UTC().Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().GetPayments(gomock.Any(), 1, 40).Return(samplePayments[:40], nil)
			m.EXPECT().GetPayments(gomock.Any(), 2, 40).Return(samplePayments[40:], nil)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			// Indices 38..49: the boundary (38) re-emitted plus 39..49.
			Expect(resp.Payments).To(HaveLen(12))
			Expect(resp.Payments[0].Reference).To(Equal(samplePayments[38].PaymentID))
		})
	})
})
