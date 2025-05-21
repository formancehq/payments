package gocardless

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"

	gocardless "github.com/gocardless/gocardless-pro-go/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Gocardless Plugin fetch next payments", func() {
	var (
		plg            *Plugin
		now            time.Time
		samplePayments []client.GocardlessPayment
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("when there are no payments", func() {
		var (
			m *client.MockClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()

			samplePayments = make([]client.GocardlessPayment, 0)

			for i := 0; i < 50; i++ {
				samplePayments = append(samplePayments, client.GocardlessPayment{
					ID:                          fmt.Sprintf("PM%d", 1+i),
					PayoutID:                    fmt.Sprintf("PM%d", 1+i),
					CreatedAt:                   now,
					Amount:                      int(10000 + i*100),
					Status:                      "pending",
					Asset:                       "EUR",
					Metadata:                    map[string]interface{}{},
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

			m.EXPECT().GetPayments(gomock.Any(), 1, "").Return(
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

			m.EXPECT().GetPayments(gomock.Any(), 60, "").Return(
				[]client.GocardlessPayment{},
				client.Cursor{},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(0))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.NewState).ToNot(BeNil())

		})

		It("should fetch next payments - no state pageSize > total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 60,
			}

			m.EXPECT().GetPayments(gomock.Any(), 60, "").Return(
				samplePayments,
				client.Cursor{
					After: "",
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
			Expect(state.After).To(Equal(samplePayments[len(samplePayments)-1].ID))

		})
		It("should fetch next payments - no state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:    []byte(`{}`),
				PageSize: 10,
			}

			m.EXPECT().GetPayments(gomock.Any(), 10, "").Return(
				samplePayments[:40],
				client.Cursor{
					After: "PM40",
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
			Expect(state.After).To(Equal("PM10"))

		})

		It("should fetch next payments - with state pageSize < total payments", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 40,
				State:    []byte(`{"after": "PM40"}`),
			}

			m.EXPECT().GetPayments(gomock.Any(), req.PageSize, "PM40").
				Return(samplePayments[:50], client.Cursor{
					After: "PM40",
				}, nil)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(40))
			Expect(resp.HasMore).To(BeTrue())
			Expect(resp.NewState).ToNot(BeNil())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			Expect(state.After).To(Equal("PM40"))

		})
	})

	Context("when there are no payments", func() {
		var (
			mockedService *client.MockGoCardlessService
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockedService = client.NewMockGoCardlessService(ctrl)

			plg.client, _ = client.New("test", "https://example.com", "access_token", true)
			plg.client.NewWithService(mockedService)
			now = time.Now().UTC()

			samplePayments = make([]client.GocardlessPayment, 0)

			for i := 0; i < 50; i++ {
				samplePayments = append(samplePayments, client.GocardlessPayment{
					ID:                          fmt.Sprintf("PM%d", i),
					CreatedAt:                   now,
					Amount:                      int(10000 + i*100),
					Status:                      "pending",
					Asset:                       "EUR",
					Metadata:                    map[string]interface{}{},
					SourceAccountReference:      fmt.Sprintf("CR%d", i),
					DestinationAccountReference: fmt.Sprintf("CU%d", i),
				})
			}
		})

		It("should fail when gocardless client returns error", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 40,
				State:    []byte(`{"after": "PM1" }`),
			}

			mockedService.EXPECT().GetGocardlessPayments(gomock.Any(), gomock.Any()).Return(nil, errors.New("test error"))

			res, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(res).To(Equal(models.FetchNextPaymentsResponse{}))

		})

		It("should return error when parsing time fails", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 60,
				State:    []byte(`{"after": "PM1" }`),
			}

			mockedService.EXPECT().GetGocardlessPayments(gomock.Any(), gomock.Any()).Return(&gocardless.PaymentListResult{Payments: []gocardless.Payment{
				{
					Amount:    5000,
					Id:        "test-id",
					CreatedAt: "invalid",
				},
			}}, nil)

			res, err := plg.FetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to parse creation time:"))
			Expect(res).To(Equal(models.FetchNextPaymentsResponse{}))

		})

		It("should return fail when fetching mandate fails", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 1,
				State:    []byte(`{"after": "PM1" }`),
			}

			mockedService.EXPECT().GetGocardlessPayments(gomock.Any(), gomock.Any()).Return(&gocardless.PaymentListResult{
				Payments: []gocardless.Payment{
					{
						Amount:    5000,
						Id:        "test-id",
						CreatedAt: "2025-02-23T14:30:15.123456789Z",
						Status:    "pending",
						Currency:  "USD",
						Metadata:  map[string]interface{}{},
						Links: &gocardless.PaymentLinks{
							Mandate: "MANDATE",
						},
					},
				},
			}, nil)

			mockedService.EXPECT().GetMandate(gomock.Any(), gomock.Any()).Return(nil, errors.New("mandate error"))

			resp, err := plg.FetchNextPayments(ctx, req)

			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("mandate error"))
			Expect(resp).To(Equal(models.FetchNextPaymentsResponse{}))
		})

		It("should return []GocardlessPayment and no error when valid CreatedAt is parsed", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				PageSize: 1,
				State:    []byte(`{"after": "PM1" }`),
			}

			mockedService.EXPECT().GetGocardlessPayments(gomock.Any(), gomock.Any()).Return(&gocardless.PaymentListResult{
				Payments: []gocardless.Payment{
					{
						Amount:    5000,
						Id:        "test-id",
						CreatedAt: "2025-02-23T14:30:15.123456789Z",
						Status:    "paid",
						Currency:  "USD",
						Metadata:  map[string]interface{}{},
						Links: &gocardless.PaymentLinks{
							Mandate: "MANDATE",
						},
						Fx: &gocardless.PaymentFx{},
					},
				},
				Meta: gocardless.PaymentListResultMeta{
					Cursors: &gocardless.PaymentListResultMetaCursors{
						After: "PM124",
					},
				},
			}, nil)

			mockedService.EXPECT().GetMandate(gomock.Any(), gomock.Any()).Return(&gocardless.Mandate{
				Links: &gocardless.MandateLinks{
					Creditor: "CR1234",
					Customer: "CU1234",
				},
			}, nil)

			resp, err := plg.FetchNextPayments(ctx, req)

			Expect(err).To(BeNil())
			Expect(resp.Payments).To(HaveLen(1))
		})
	})
})
