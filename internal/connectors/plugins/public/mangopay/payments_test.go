package mangopay

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin Payments", func() {
	var (
		m   *client.MockClient
		plg models.Plugin
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	Context("fetching next payments", func() {
		var (
			sampleTransactions []client.Payment
			now                time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			sampleTransactions = make([]client.Payment, 0)
			for i := 0; i < 50; i++ {
				sampleTransactions = append(sampleTransactions, client.Payment{
					Id:           fmt.Sprintf("%d", i),
					CreationDate: now.Add(-time.Duration(50-i) * time.Minute).UTC().Unix(),
					DebitedFunds: client.Funds{
						Currency: "USD",
						Amount:   "100",
					},
					Status:           "SUCCEEDED",
					Type:             "PAYIN",
					CreditedWalletID: "acc2",
					DebitedWalletID:  "acc1",
				})
			}
		})

		It("should return an error - get transactions error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 60, time.Time{}).Return(
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
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 60, time.Time{}).Return(
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
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreationDate.IsZero()).To(BeTrue())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 60, time.Time{}).Return(
				sampleTransactions,
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
			createdTime := time.Unix(sampleTransactions[49].CreationDate, 0)
			Expect(state.LastPage).To(Equal(1))
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize:    40,
				FromPayload: json.RawMessage(`{"Reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 40, time.Time{}).Return(
				sampleTransactions[:40],
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
			Expect(state.LastPage).To(Equal(1))
			createdTime := time.Unix(sampleTransactions[39].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			lastCreatedAt := time.Unix(sampleTransactions[38].CreationDate, 0)
			req := models.FetchNextPaymentsRequest{
				State:       []byte(fmt.Sprintf(`{"lastPage": 1, "lastCreationDate": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano))),
				PageSize:    10,
				FromPayload: json.RawMessage(`{"Reference": "test"}`),
			}

			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 10, lastCreatedAt.UTC()).Return(
				sampleTransactions[39:49],
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(10))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// We fetched everything, state should be resetted
			Expect(state.LastPage).To(Equal(1))
			createdTime := time.Unix(sampleTransactions[48].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})
	})
})
