package mangopay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/ce/plugins/mangopay/client"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
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

			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 10, lastCreatedAt.UTC().Add(-time.Second)).Return(
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
			createdTime := time.Unix(sampleTransactions[48].CreationDate, 0)
			Expect(state.LastCreationDate.UTC()).To(Equal(createdTime.UTC()))
		})

		It("accumulates the processed-ID set when the batch stays in the watermark second", func(ctx SpecContext) {
			// Given: the watermark already sits on the shared second, and the next
			// batch is entirely within it.
			lastCreatedAt := time.Unix(sampleTransactions[4].CreationDate, 0).UTC()

			req := models.FetchNextPaymentsRequest{
				State:       []byte(fmt.Sprintf(`{"lastCreationDate": "%s"}`, lastCreatedAt.UTC().Format(time.RFC3339Nano))),
				PageSize:    5,
				FromPayload: json.RawMessage(`{"Reference": "test"}`),
			}

			// Set all transactions to have the same CreationDate as the watermark.
			for i := range sampleTransactions {
				sampleTransactions[i].CreationDate = sampleTransactions[4].CreationDate
			}
			transactionsReturnedByClient := sampleTransactions[5:10]
			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 5, lastCreatedAt.Add(-time.Second)).Times(1).Return(
				transactionsReturnedByClient,
				nil,
			)

			// When
			resp, err := plg.FetchNextPayments(ctx, req)

			// Then
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(5))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.LastCreationDate.UTC()).To(Equal(lastCreatedAt))
			// Watermark second unchanged -> the emitted IDs are tracked so the next
			// cycle skips them.
			Expect(state.LastProcessedIDs).To(ConsistOf("5", "6", "7", "8", "9"))
		})

		It("includes a transaction at exactly the watermark second (M-CON3)", func(ctx SpecContext) {
			watermark := time.Unix(sampleTransactions[10].CreationDate, 0).UTC()
			req := models.FetchNextPaymentsRequest{
				State:       []byte(fmt.Sprintf(`{"lastPage": 1, "lastCreationDate": "%s"}`, watermark.Format(time.RFC3339Nano))),
				PageSize:    60,
				FromPayload: json.RawMessage(`{"Reference": "test"}`),
			}

			// A transaction created in the SAME second as the watermark would be
			// dropped by an exclusive AfterDate=watermark filter. The fix queries
			// AfterDate=watermark-1s, so the server returns the watermark second and
			// this row is emitted instead of lost.
			atWatermark := client.Payment{
				Id:               "same-second",
				CreationDate:     watermark.Unix(),
				DebitedFunds:     client.Funds{Currency: "USD", Amount: "100"},
				Status:           "SUCCEEDED",
				Type:             "PAYIN",
				CreditedWalletID: "acc2",
				DebitedWalletID:  "acc1",
			}
			m.EXPECT().GetTransactions(gomock.Any(), "test", 1, 60, watermark.Add(-time.Second)).Return(
				[]client.Payment{atWatermark},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
			Expect(resp.Payments[0].Reference).To(Equal("same-second"))
		})

		It("walks a same-second group larger than PageSize across cycles without stalling", func(ctx SpecContext) {
			ts := time.Now().UTC().Add(-time.Hour)
			mk := func(id string) client.Payment {
				return client.Payment{
					Id:               id,
					CreationDate:     ts.Unix(),
					DebitedFunds:     client.Funds{Currency: "USD", Amount: "100"},
					Status:           "SUCCEEDED",
					Type:             "PAYIN",
					CreditedWalletID: "acc2",
					DebitedWalletID:  "acc1",
				}
			}
			all := []client.Payment{mk("t0"), mk("t1"), mk("t2"), mk("t3"), mk("t4")}
			// Mangopay rescans from page 1 each cycle (AfterDate re-includes the
			// watermark second); serve the list page by page so the processed-ID set
			// has to skip already-emitted siblings to make progress.
			m.EXPECT().GetTransactions(gomock.Any(), "test", gomock.Any(), 2, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, page, _ int, _ time.Time) ([]client.Payment, error) {
					start := (page - 1) * 2
					if start >= len(all) {
						return []client.Payment{}, nil
					}
					end := start + 2
					if end > len(all) {
						end = len(all)
					}
					return all[start:end], nil
				},
			).AnyTimes()
			refs := func(ps []models.PSPPayment) []string {
				out := make([]string, len(ps))
				for i := range ps {
					out[i] = ps[i].Reference
				}
				return out
			}
			fromPayload := json.RawMessage(`{"Reference": "test"}`)

			// Cycle 1: page 1 -> t0, t1.
			resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: []byte(`{}`), PageSize: 2, FromPayload: fromPayload})
			Expect(err).To(BeNil())
			Expect(refs(resp.Payments)).To(Equal([]string{"t0", "t1"}))

			// Cycle 2: rescan skips t0,t1 (in set), page 2 -> t2, t3.
			resp, err = plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: resp.NewState, PageSize: 2, FromPayload: fromPayload})
			Expect(err).To(BeNil())
			Expect(refs(resp.Payments)).To(ConsistOf("t2", "t3"))

			// Cycle 3: page 3 -> t4 (group fully drained, no stall).
			resp, err = plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: resp.NewState, PageSize: 2, FromPayload: fromPayload})
			Expect(err).To(BeNil())
			Expect(refs(resp.Payments)).To(ContainElement("t4"))
		})
	})
})
