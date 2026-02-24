package column

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Column Plugin Payments", func() {
	var (
		ctrl           *gomock.Controller
		mockHTTPClient *client.MockHTTPClient
		plg            connector.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockHTTPClient = client.NewMockHTTPClient(ctrl)
		c := client.New("test", "aseplye", "https://test.com")
		c.SetHttpClient(mockHTTPClient)
		plg = &Plugin{client: c}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next payments", func() {
		var (
			sampleTransactions []*client.Transaction
			now                time.Time
		)

		BeforeEach(func() {
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
			req := connector.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			expectedErr := errors.New("test error")
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				expectedErr,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(expectedErr))
			Expect(resp).To(Equal(connector.FetchNextPaymentsResponse{}))
		})

		It("should fetch next payments - no state no results", func(ctx SpecContext) {
			req := connector.FetchNextPaymentsRequest{
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
			req := connector.FetchNextPaymentsRequest{
				State:    []byte(`{"timeline":{"backlog_cursor":"123"}}`),
				PageSize: 60,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				NewRequestMatcher("limit=60&starting_after=123"),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.TransactionResponseWrapper[[]*client.Transaction]{
				Transfers: sampleTransactions[:20],
			})
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				NewRequestMatcher("ending_before=19&limit=60"),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.TransactionResponseWrapper[[]*client.Transaction]{
				Transfers: sampleTransactions[:19],
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
			Expect(state.Timeline.LastSeenID).To(Equal(sampleTransactions[0].ID))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := connector.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				NewRequestMatcher("limit=40"),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.TransactionResponseWrapper[[]*client.Transaction]{
				Transfers: sampleTransactions[:13],
				HasMore:   false,
			})
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				NewRequestMatcher("ending_before=12&limit=40"),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.TransactionResponseWrapper[[]*client.Transaction]{
				Transfers: sampleTransactions[:12],
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
			Expect(state.Timeline.LastSeenID).To(Equal(sampleTransactions[0].ID))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			lastIDCreated := sampleTransactions[38].ID
			req := connector.FetchNextPaymentsRequest{
				State:    []byte(fmt.Sprintf(`{"timeline":{"last_seen_id":"%s"}}`, lastIDCreated)),
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
			Expect(state.Timeline.LastSeenID).To(Equal(sampleTransactions[0].ID))
		})
	})
})

type HTTPRequestMatcher struct {
	got           *http.Request
	ExpectedQuery string
}

func NewRequestMatcher(query string) gomock.Matcher {
	return &HTTPRequestMatcher{ExpectedQuery: query}
}

func (m *HTTPRequestMatcher) Matches(x interface{}) bool {
	actual, ok := x.(*http.Request)
	if !ok {
		return false
	}
	m.got = actual

	//nolint:gosimple
	if actual.URL.RawQuery == m.ExpectedQuery {
		return true
	}
	return false
}

func (m *HTTPRequestMatcher) String() string {
	if m.got == nil {
		return "not a valid *http.Reqeust"
	}
	return fmt.Sprintf("%s did not match expected %s", m.got.URL.RawQuery, m.ExpectedQuery)
}
