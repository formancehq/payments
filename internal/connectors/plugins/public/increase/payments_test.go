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

	Context("fetching next accounts", func() {
		var (
			m                           *client.MockClient
			sampleSucceededTransactions []*client.Transaction
			samplePendingTransactions   []*client.Transaction
			sampleDeclinedTransactions  []*client.Transaction
			now                         time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()
			sampleSucceededTransactions = make([]*client.Transaction, 0)
			samplePendingTransactions = make([]*client.Transaction, 0)
			sampleDeclinedTransactions = make([]*client.Transaction, 0)

			for i := 0; i < 50; i++ {
				sampleSucceededTransactions = append(sampleSucceededTransactions, &client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: "2345433",
					Amount:    "100.01",
					Source: struct {
						DestinationAccountID string "json:\"destination_account_id\""
						SourceAccountID      string "json:\"source_account_id\""
						TransactionID        string "json:\"transaction_id\""
					}{
						DestinationAccountID: "2345432",
						SourceAccountID:      "09876543",
						TransactionID:        "123467898",
					},
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
					Currency:  "USD",
				})
			}
			for i := 0; i < 50; i++ {
				samplePendingTransactions = append(samplePendingTransactions, &client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: "2345433",
					Amount:    "100.01",
					Source: struct {
						DestinationAccountID string "json:\"destination_account_id\""
						SourceAccountID      string "json:\"source_account_id\""
						TransactionID        string "json:\"transaction_id\""
					}{
						DestinationAccountID: "2345432",
						SourceAccountID:      "09876543",
						TransactionID:        "123467898",
					},
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
					Currency:  "USD",
				})
			}
			for i := 0; i < 50; i++ {
				sampleDeclinedTransactions = append(sampleDeclinedTransactions, &client.Transaction{
					ID:        fmt.Sprintf("%d", i),
					AccountID: "2345433",
					Amount:    "100.01",
					Source: struct {
						DestinationAccountID string "json:\"destination_account_id\""
						SourceAccountID      string "json:\"source_account_id\""
						TransactionID        string "json:\"transaction_id\""
					}{
						DestinationAccountID: "2345432",
						SourceAccountID:      "09876543",
						TransactionID:        "123467898",
					},
					CreatedAt: now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
					Date:      now.Add(-time.Duration(50-i) * time.Minute).UTC().Format("2006-01-02T15:04:05.999-0700"),
					Currency:  "USD",
				})
			}
		})

		It("should return an error - get transactions error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 60, "").Return(
				[]*client.Transaction{},
				"",
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

			m.EXPECT().GetTransactions(gomock.Any(), 60, "").Return(
				[]*client.Transaction{},
				"",
				nil,
			)

			m.EXPECT().GetPendingTransactions(gomock.Any(), 60, "").Return(
				[]*client.Transaction{},
				"",
				nil,
			)

			m.EXPECT().GetDeclinedTransactions(gomock.Any(), 60, "").Return(
				[]*client.Transaction{},
				"",
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
			Expect(state.NextSucceededCursor).To(BeEmpty())
			Expect(state.NextPendingCursor).To(BeEmpty())
			Expect(state.NextDeclinedCursor).To(BeEmpty())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 60, "").Return(
				sampleSucceededTransactions,
				"",
				nil,
			)

			m.EXPECT().GetPendingTransactions(gomock.Any(), 60, "").Return(
				samplePendingTransactions[:1],
				"",
				nil,
			)

			m.EXPECT().GetDeclinedTransactions(gomock.Any(), 60, "").Return(
				sampleDeclinedTransactions[:1],
				"",
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(52))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())
			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.NextSucceededCursor).To(BeEmpty())
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 40, "").Return(
				sampleSucceededTransactions[:40],
				"qwerty",
				nil,
			)

			m.EXPECT().GetPendingTransactions(gomock.Any(), 40, "").Return(
				samplePendingTransactions[:40],
				"uiop",
				nil,
			)

			m.EXPECT().GetDeclinedTransactions(gomock.Any(), 40, "").Return(
				sampleDeclinedTransactions[:40],
				"asdfg",
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
			Expect(state.NextSucceededCursor).To(Equal("qwerty"))
			Expect(state.NextPendingCursor).To(Equal("uiop"))
			Expect(state.NextDeclinedCursor).To(Equal("asdfg"))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{"next_succeeded_cursor": "qwerty", "next_pending_cursor": "uiop", "next_declined_cursor": "asdfg"}`),
				PageSize: 40,
			}

			m.EXPECT().GetTransactions(gomock.Any(), 40, "qwerty").Return(
				sampleSucceededTransactions[:40],
				"mnbvc",
				nil,
			)
			m.EXPECT().GetPendingTransactions(gomock.Any(), 40, "uiop").Return(
				samplePendingTransactions[:40],
				"lkjh",
				nil,
			)
			m.EXPECT().GetDeclinedTransactions(gomock.Any(), 40, "asdfg").Return(
				sampleDeclinedTransactions[:40],
				"uytr",
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
			// We fetched everything, state should be resetted
			Expect(state.NextSucceededCursor).To(Equal("mnbvc"))
			Expect(state.NextPendingCursor).To(Equal("lkjh"))
			Expect(state.NextDeclinedCursor).To(Equal("uytr"))
		})
	})
})
