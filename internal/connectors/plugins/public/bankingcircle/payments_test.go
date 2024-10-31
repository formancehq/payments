package bankingcircle

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("BankingCircle Plugin Payments", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next accounts", func() {
		var (
			m              *client.MockClient
			samplePayments []client.Payment
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
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

			m.EXPECT().GetPayments(ctx, 1, 60).Return(
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

			m.EXPECT().GetPayments(ctx, 1, 60).Return(
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

			m.EXPECT().GetPayments(ctx, 1, 60).Return(
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

			m.EXPECT().GetPayments(ctx, 1, 40).Return(
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
				State:    []byte(fmt.Sprintf(`{"latestStatusChangedTimestamp": "%s"}`, samplePayments[38].LatestStatusChangedTimestamp.UTC().Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().GetPayments(ctx, 1, 40).Return(
				samplePayments[:40],
				nil,
			)

			m.EXPECT().GetPayments(ctx, 2, 40).Return(
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
		})
	})
})
