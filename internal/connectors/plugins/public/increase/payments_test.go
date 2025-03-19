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

			for i := 0; i < 50; i++ {
				sampleSucceededTransactions = append(sampleSucceededTransactions, &client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: "2345433",
					Amount:    100,
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Currency:  "USD",
				})
			}
			for i := 0; i < 50; i++ {
				samplePendingTransactions = append(samplePendingTransactions, &client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: "2345433",
					Amount:    100,
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Currency:  "USD",
				})
			}
			for i := 0; i < 50; i++ {
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
				500,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get pending transactions: test error : : status code: 0"))
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
				500,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get pending transactions: test error : : status code: 0"))
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
			// We fetched everything, state should be reset
			Expect(state.LastSucceededCreatedAt.IsZero()).To(BeTrue())
			Expect(state.LastPendingCreatedAt.IsZero()).To(BeTrue())
			Expect(state.LastDeclinedCreatedAt.IsZero()).To(BeTrue())
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
			// We fetched everything, state should be reset
			succeededCreatedTime, _ := time.Parse(time.RFC3339, sampleSucceededTransactions[19].CreatedAt)
			pendingCreatedTime, _ := time.Parse(time.RFC3339, samplePendingTransactions[19].CreatedAt)
			declinedCreatedTime, _ := time.Parse(time.RFC3339, sampleDeclinedTransactions[19].CreatedAt)
			Expect(state.LastSucceededCreatedAt.UTC()).To(Equal(succeededCreatedTime.UTC()))
			Expect(state.LastPendingCreatedAt.UTC()).To(Equal(pendingCreatedTime.UTC()))
			Expect(state.LastDeclinedCreatedAt.UTC()).To(Equal(declinedCreatedTime.UTC()))
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
				Data: samplePendingTransactions[:13],
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
			succeededCreatedTime, _ := time.Parse(time.RFC3339, sampleSucceededTransactions[12].CreatedAt)
			pendingCreatedTime, _ := time.Parse(time.RFC3339, samplePendingTransactions[12].CreatedAt)
			declinedCreatedTime, _ := time.Parse(time.RFC3339, sampleDeclinedTransactions[12].CreatedAt)
			Expect(state.LastSucceededCreatedAt.UTC()).To(Equal(succeededCreatedTime.UTC()))
			Expect(state.LastPendingCreatedAt.UTC()).To(Equal(pendingCreatedTime.UTC()))
			Expect(state.LastDeclinedCreatedAt.UTC()).To(Equal(declinedCreatedTime.UTC()))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			lastSucceededCreatedAt, _ := time.Parse(time.RFC3339, sampleSucceededTransactions[38].CreatedAt)
			lastPendingCreatedAt, _ := time.Parse(time.RFC3339, samplePendingTransactions[38].CreatedAt)
			lastDeclinedCreatedAt, _ := time.Parse(time.RFC3339, sampleDeclinedTransactions[38].CreatedAt)
			req := models.FetchNextPaymentsRequest{
				State:    []byte(fmt.Sprintf(`{"last_succeeded_created_at": "%s", "last_pending_created_at": "%s", "last_declined_created_at": "%s"}`, lastSucceededCreatedAt.Format(time.RFC3339Nano), lastPendingCreatedAt.Format(time.RFC3339Nano), lastDeclinedCreatedAt.Format(time.RFC3339Nano))),
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
				NextCursor: "mnbvc",
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
				Data: samplePendingTransactions[:13],
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
				NextCursor: "uytr",
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(39))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be reset
			succeededCreatedTime, _ := time.Parse(time.RFC3339, sampleSucceededTransactions[12].CreatedAt)
			pendingCreatedTime, _ := time.Parse(time.RFC3339, samplePendingTransactions[12].CreatedAt)
			declinedCreatedTime, _ := time.Parse(time.RFC3339, sampleDeclinedTransactions[12].CreatedAt)
			Expect(state.LastSucceededCreatedAt.UTC()).To(Equal(succeededCreatedTime.UTC()))
			Expect(state.LastPendingCreatedAt.UTC()).To(Equal(pendingCreatedTime.UTC()))
			Expect(state.LastDeclinedCreatedAt.UTC()).To(Equal(declinedCreatedTime.UTC()))
		})
	})
})
