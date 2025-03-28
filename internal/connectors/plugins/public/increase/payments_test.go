package increase

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Increase Plugin Payments", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next payments", func() {
		var (
			mockHTTPClient              *client.MockHTTPClient
			sampleSucceededTransactions []*client.Transaction
			samplePendingTransactions   []*client.Transaction
			sampleDeclinedTransactions  []*client.Transaction
			now                         time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com", "we5432345")
			plg.client.SetHttpClient(mockHTTPClient) // Inject the mock HTTP client
			now = time.Now().UTC()

			sampleSucceededTransactions = make([]*client.Transaction, 0)
			samplePendingTransactions = make([]*client.Transaction, 0)
			sampleDeclinedTransactions = make([]*client.Transaction, 0)

			for i := range 50 {
				sampleSucceededTransactions = append(sampleSucceededTransactions, &client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: "2345433",
					Amount:    100,
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Currency:  "USD",
					RouteID:   "234",
					RouteType: "123",
					Source: client.Source{
						Category: "inbound_ach_transfer",
					},
				})
			}
			for i := range 50 {
				samplePendingTransactions = append(samplePendingTransactions, &client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: "2345433",
					Amount:    100,
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category: "check_deposit_acceptance",
					},
				})
			}
			for i := range 50 {
				sampleDeclinedTransactions = append(sampleDeclinedTransactions, &client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: "2345433",
					Amount:    100,
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Currency:  "USD",
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
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data: samplePendingTransactions[:20],
			})

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
			Expect(err).To(MatchError("failed to get transactions: test error : : status code: 0"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should return an error - get pending transactions error", func(ctx SpecContext) {
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
			Expect(err).To(MatchError("failed to get pending transactions: test error : : status code: 0"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should return an error - get declined transactions error", func(ctx SpecContext) {
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
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data: samplePendingTransactions[:20],
			})

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data: sampleSucceededTransactions[:20],
			})

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
			Expect(err).To(MatchError("failed to get declined transactions: test error : : status code: 0"))
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
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data: []*client.Transaction{},
			}).Times(3)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
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
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data: sampleSucceededTransactions[:20],
			})

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data: samplePendingTransactions[:20],
			})

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data: sampleDeclinedTransactions[:20],
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(60))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
		})

		It("should fetch next payments - amount should always be non-negative", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			sampleSucceededTransactions[0].Amount = -1000
			samplePendingTransactions[0].Amount = -2000
			sampleDeclinedTransactions[0].Amount = -3000
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       sampleSucceededTransactions[:13],
				NextCursor: "qwerty",
			})

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       samplePendingTransactions[:13],
				NextCursor: "asdfg",
			})

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       sampleDeclinedTransactions[:13],
				NextCursor: "zxcvb",
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments[0].Amount.Int64()).To(BeNumerically(">=", 0))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
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
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       sampleSucceededTransactions[:13],
				NextCursor: "qwerty",
			})

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       samplePendingTransactions[:13],
				NextCursor: "zxcvb",
			})

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       sampleDeclinedTransactions[:13],
				NextCursor: "asdfg",
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(39))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
		})

		It("should fetch next payments - with stop state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{"next_succeeded_cursor": "", "next_pending_cursor": "", "next_declined_cursor": "", "stop_succeeded": true, "stop_pending": true, "stop_declined": true}`),
				PageSize: 40,
			}

			// Since the state indicates to stop fetching succeeded, pending, and declined transactions,
			// no HTTP calls should be made. We expect the response to have no payments and HasMore to be false.

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
		})
	})
})
