package gocardless

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Gocardless Plugin fetch next payments", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("when there are no payments", func() {
		var (
			m              *client.MockClient
			samplePayments []client.GocardlessPayment
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			samplePayments = make([]client.GocardlessPayment, 0)

			for i := 0; i < 50; i++ {
				samplePayments = append(samplePayments, client.GocardlessPayment{
					ID:                          fmt.Sprintf("PM%d", i),
					CreatedAt:                   now.Add(-time.Duration(50-i) * time.Minute).Unix(),
					Amount:                      int(10000 + i*100),
					Status:                      "pending",
					Asset:                       "EUR",
					Metadata:                    map[string]string{},
					SourceAccountReference:      fmt.Sprintf("CR%d", i),
					DestinationAccountReference: fmt.Sprintf("CU%d", i),
				})
			}
		})

		It("should return an error - get payments error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 1,
			}

			m.EXPECT().GetPayments(gomock.Any(), client.PaymentPayload{}, 1, "", "").Return(
				[]client.GocardlessPayment{},
				client.Cursor{},
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

			m.EXPECT().GetPayments(gomock.Any(), client.PaymentPayload{}, 60, "", "").Return(
				[]client.GocardlessPayment{},
				client.Cursor{},
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
			Expect(state.LastCreationDate).To(BeZero())
		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetPayments(gomock.Any(), client.PaymentPayload{}, 60, "", "").Return(
				samplePayments,
				client.Cursor{
					After:  "PM50",
					Before: "PM1",
				},
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
			Expect(state.After).To(Equal("PM50"))
			Expect(state.Before).To(Equal("PM1"))

		})
		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetPayments(gomock.Any(), client.PaymentPayload{}, 10, "", "").Return(
				samplePayments[:40],
				client.Cursor{
					After:  "PM40",
					Before: "PM1",
				},
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
			Expect(state.After).To(Equal("PM40"))
			Expect(state.Before).To(Equal("PM1"))
		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			lastCreationDate := time.Unix(samplePayments[38].CreatedAt, 0)

			req := models.FetchNextPaymentsRequest{
				PageSize: 40,
				State:    []byte(`{"after": "PM1", "before": "PM40", "lastCreationDate": "` + lastCreationDate.Format(time.RFC3339Nano) + `"}`),
			}

			m.EXPECT().GetPayments(gomock.Any(), client.PaymentPayload{}, req.PageSize, "PM1", "PM40").
				Return(samplePayments[:40], client.Cursor{
					After:  "PM40",
					Before: "PM1",
				}, nil)

			m.EXPECT().GetPayments(gomock.Any(), client.PaymentPayload{}, req.PageSize, "PM40", "PM1").
				Return(samplePayments[41:], client.Cursor{
					After:  "PM50",
					Before: "PM41",
				}, nil)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(10))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal("PM50"))
			Expect(state.Before).To(Equal("PM41"))

		})
	})
})
