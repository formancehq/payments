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
					ID:        fmt.Sprintf("pa_%d", i),
					AccountID: "2345433",
					Amount:    100,
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Currency:  "USD",
					RouteID:   "234",
					RouteType: "123",
					Source: client.Source{
						Category:   "inbound_ach_transfer",
						TransferID: "account_transfer_87654",
					},
				})
			}
			for i := range 50 {
				samplePendingTransactions = append(samplePendingTransactions, &client.Transaction{
					ID:        fmt.Sprintf("pa_%d", i),
					AccountID: "2345433",
					Amount:    100,
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category:       "check_deposit_acceptance",
						CheckDepositID: "check_deposit_12345",
					},
				})
			}
			for i := range 50 {
				sampleDeclinedTransactions = append(sampleDeclinedTransactions, &client.Transaction{
					ID:        fmt.Sprintf("pa_%d", i),
					AccountID: "2345433",
					Amount:    100,
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category:              "check_decline",
						InboundCheckDepositID: "check_deposit_12345",
					},
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
			Expect(err).To(MatchError("failed to get transactions for timeline: test error : : status code: 0"))
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
			Expect(err).To(MatchError("failed to get pending_transactions for timeline: test error : : status code: 0"))
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
			Expect(err).To(MatchError("failed to get declined_transactions for timeline: test error : : status code: 0"))
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

			// Create test transactions with negative amounts
			testTransactions := []*client.Transaction{
				{
					ID:        "succeeded_1",
					AccountID: "account_1",
					Amount:    -1000,
					CreatedAt: now.Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category: "inbound_ach_transfer",
					},
				},
				{
					ID:        "pending_1",
					AccountID: "account_2",
					Amount:    -2000,
					CreatedAt: now.Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category: "check_deposit_acceptance",
					},
				},
				{
					ID:        "declined_1",
					AccountID: "account_3",
					Amount:    -3000,
					CreatedAt: now.Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category: "check_decline",
					},
				},
			}

			// Mock initial scan for oldest records
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       []*client.Transaction{testTransactions[0]},
				NextCursor: "",
			})

			// Mock pending transactions scan
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       []*client.Transaction{testTransactions[1]},
				NextCursor: "",
			})

			// Mock declined transactions scan
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       []*client.Transaction{testTransactions[2]},
				NextCursor: "",
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).ToNot(BeEmpty())

			// Verify that all amounts are non-negative
			for _, payment := range resp.Payments {
				Expect(payment.Amount.Int64()).To(BeNumerically(">=", 0))
			}

			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
		})

		It("should fetch next payments - no empty reference when transfer_id is empty", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			// Create test transactions with different source types
			testTransactions := []*client.Transaction{
				{
					ID:        "succeeded_1",
					AccountID: "account_1",
					Amount:    1000,
					CreatedAt: now.Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category:   "inbound_ach_transfer",
						TransferID: "account_transfer_87654",
					},
				},
				{
					ID:        "pending_1",
					AccountID: "account_2",
					Amount:    2000,
					CreatedAt: now.Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category:       "check_deposit_acceptance",
						CheckDepositID: "check_deposit_12345",
					},
				},
				{
					ID:        "declined_1",
					AccountID: "account_3",
					Amount:    3000,
					CreatedAt: now.Format(time.RFC3339),
					Currency:  "USD",
					Source: client.Source{
						Category:              "check_decline",
						InboundCheckDepositID: "check_deposit_12345",
					},
				},
			}

			// Mock initial scan for oldest records
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       []*client.Transaction{testTransactions[0]},
				NextCursor: "",
			})

			// Mock pending transactions scan
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       []*client.Transaction{testTransactions[1]},
				NextCursor: "",
			})

			// Mock declined transactions scan
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ResponseWrapper[[]*client.Transaction]{
				Data:       []*client.Transaction{testTransactions[2]},
				NextCursor: "",
			})

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).ToNot(BeEmpty())

			// Verify references and parent references
			Expect(resp.Payments[0].Reference).To(Equal("succeeded_1"))
			Expect(resp.Payments[0].ParentReference).To(Equal("account_transfer_87654"))
			Expect(resp.Payments[1].Reference).To(Equal("pending_1"))
			Expect(resp.Payments[1].ParentReference).To(Equal("check_deposit_12345"))
			Expect(resp.Payments[2].Reference).To(Equal("declined_1"))
			Expect(resp.Payments[2].ParentReference).To(Equal("check_deposit_12345"))

			Expect(resp.HasMore).To(BeFalse())
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
