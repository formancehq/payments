package column

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Column Plugin Payments", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next payments", func() {
		var (
			mockHTTPClient     *client.MockHTTPClient
			sampleTransactions []*client.Transaction
			now                time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com")
			plg.client.SetHttpClient(mockHTTPClient)
			now = time.Now().UTC()
			sampleTransactions = make([]*client.Transaction, 0)
			statuses := []string{
				"pending",          // PAYMENT_STATUS_PENDING
				"completed",        // PAYMENT_STATUS_SUCCEEDED
				"canceled",         // PAYMENT_STATUS_CANCELLED
				"failed",           // PAYMENT_STATUS_FAILED
				"returned",         // PAYMENT_STATUS_REFUNDED
				"return_contested", // PAYMENT_STATUS_REFUND_REVERSED
				"first_return",     // PAYMENT_STATUS_REFUNDED_FAILURE
				"manual_review",    // PAYMENT_STATUS_AUTHORISATION
				"hold",             // PAYMENT_STATUS_CAPTURE
			}
			transactionTypes := []string{
				"book",
				"swift",
				"realtime",
				"wire",
				"ach_debit",
				"ach_credit",
			}
			for i := range 50 {
				sampleTransactions = append(sampleTransactions, &client.Transaction{
					ID:           fmt.Sprintf("%d", i),
					Amount:       100,
					Status:       statuses[i%len(statuses)],
					CreatedAt:    now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					CompletedAt:  now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					CurrencyCode: "USD",
					Type:         transactionTypes[i%len(transactionTypes)],
				})
			}
		})

		It("should return an error - get transactions error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get transactions: test error : "))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch next payments - no state no results", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.TransactionResponseWrapper[[]*client.Transaction]{
				Transfers: []*client.Transaction{},
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(BeEmpty())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.TransactionResponseWrapper[[]*client.Transaction]{
				Transfers: sampleTransactions[:20],
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(20))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(Equal("19"))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.TransactionResponseWrapper[[]*client.Transaction]{
				Transfers: sampleTransactions[:13],
				HasMore:   true,
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(13))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())
			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastIDCreated).To(Equal("12"))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			lastIDCreated := sampleTransactions[38].ID
			req := models.FetchNextPaymentsRequest{
				State:    []byte(fmt.Sprintf(`{"lastIDCreated": "%s"}`, lastIDCreated)),
				PageSize: 40,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.TransactionResponseWrapper[[]*client.Transaction]{
				Transfers: sampleTransactions[:13],
				HasMore:   true,
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(13))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())
			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			Expect(state.LastIDCreated).To(Equal("12"))
		})
	})
})
