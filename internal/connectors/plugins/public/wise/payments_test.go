package wise

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin Payments", func() {
	var (
		m   *client.MockClient
		plg models.Plugin
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	Context("fetching next accounts", func() {
		var (
			sampleTransfers []client.Transfer
			now             time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleTransfers = make([]client.Transfer, 0)
			for i := 0; i < 50; i++ {
				sampleTransfers = append(sampleTransfers, client.Transfer{
					ID:                   uint64(i),
					Reference:            fmt.Sprintf("test%d", i),
					Status:               "outgoing_payment_sent",
					SourceAccount:        1,
					SourceCurrency:       "USD",
					SourceValue:          "100",
					TargetAccount:        2,
					TargetCurrency:       "USD",
					TargetValue:          "100",
					User:                 1,
					SourceBalanceID:      123,
					DestinationBalanceID: 321,
					CreatedAt:            now.Add(-time.Duration(50-i) * time.Minute).UTC(),
				})
			}
		})

		It("should return an error - missing from payload", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 60,
			}

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing from payload in request"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should return an error - get transactions error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 0, 60).Return(
				[]client.Transfer{},
				errors.New("test error"),
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should fetch next payments - no state no results", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 0, 60).Return(
				[]client.Transfer{},
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
			Expect(state.Offset).To(Equal(0))
			Expect(state.LastTransferID).To(Equal(uint64(0)))
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize:    60,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 0, 60).Return(
				sampleTransfers,
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
			// Offset should not be increased as we did not attain the pageSize
			Expect(state.Offset).To(Equal(0))
			Expect(state.LastTransferID).To(Equal(uint64(49)))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize:    40,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 0, 40).Return(
				sampleTransfers[:40],
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
			// We attain the pageSize, offset should be increased
			Expect(state.Offset).To(Equal(40))
			Expect(state.LastTransferID).To(Equal(uint64(39)))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       []byte(`{"offset": 0, "lastTransferID": 38}`),
				PageSize:    40,
				FromPayload: []byte(`{"id": 0}`),
			}

			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 0, 40).Return(
				sampleTransfers[:40],
				nil,
			)

			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 40, 40).Return(
				sampleTransfers[40:],
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
			Expect(state.Offset).To(Equal(40))
			Expect(state.LastTransferID).To(Equal(uint64(49)))
		})
	})
})
