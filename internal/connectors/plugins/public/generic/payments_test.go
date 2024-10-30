package generic

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Generic Plugin Payments", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next payments", func() {
		var (
			m              *client.MockClient
			samplePayments []genericclient.Transaction
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			samplePayments = make([]genericclient.Transaction, 0)
			for i := 0; i < 50; i++ {
				samplePayments = append(samplePayments, genericclient.Transaction{
					Id:                   fmt.Sprint(i),
					CreatedAt:            now.Add(-time.Duration(50-i) * time.Minute).UTC(),
					UpdatedAt:            now.Add(-time.Duration(50-i) * time.Minute).UTC(),
					Currency:             "EUR",
					Type:                 genericclient.PAYIN,
					Status:               genericclient.SUCCEEDED,
					Amount:               "1000",
					SourceAccountID:      pointer.For("acc1"),
					DestinationAccountID: pointer.For("acc2"),
					Metadata:             map[string]string{"foo": "bar"},
				})
			}
		})

		It("should return an error - get payments error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().ListTransactions(ctx, int64(0), int64(60), time.Time{}).Return(
				[]genericclient.Transaction{},
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

			m.EXPECT().ListTransactions(ctx, int64(0), int64(60), time.Time{}).Return(
				[]genericclient.Transaction{},
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
			Expect(state.LastUpdatedAtFrom.IsZero()).To(BeTrue())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().ListTransactions(ctx, int64(0), int64(60), time.Time{}).Return(
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
			// We fetched everything, state should be resetted
			Expect(state.LastUpdatedAtFrom.UTC()).To(Equal(samplePayments[49].UpdatedAt.UTC()))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 40,
			}

			m.EXPECT().ListTransactions(ctx, int64(0), int64(40), time.Time{}).Return(
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
			Expect(state.LastUpdatedAtFrom.UTC()).To(Equal(samplePayments[39].UpdatedAt.UTC()))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(fmt.Sprintf(`{"lastUpdatedAtFrom": "%s"}`, samplePayments[38].UpdatedAt.Format(time.RFC3339Nano))),
				PageSize: 40,
			}

			m.EXPECT().ListTransactions(ctx, int64(0), int64(40), samplePayments[38].UpdatedAt.UTC()).Return(
				samplePayments[:40],
				nil,
			)

			m.EXPECT().ListTransactions(ctx, int64(1), int64(40), samplePayments[38].UpdatedAt.UTC()).Return(
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
			Expect(state.LastUpdatedAtFrom.UTC()).To(Equal(samplePayments[49].UpdatedAt.UTC()))
		})
	})
})
