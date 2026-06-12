package wise

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin Payments", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m, logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false)}
	})

	AfterEach(func() {
		ctrl.Finish()
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

		It("should not lose payments when a short page is followed by a full one (EN-1087)", func(ctx SpecContext) {
			// Page 1 emits fewer than pageSize payments because one transfer has
			// an unsupported currency and is skipped. Page 2 is full, pushing the
			// accumulated total above pageSize. The previously buggy trim dropped
			// the overflow transfers while advancing the offset past them, losing
			// them permanently.
			req := models.FetchNextPaymentsRequest{
				PageSize:    3,
				FromPayload: []byte(`{"id": 0}`),
			}

			makeTransfer := func(id uint64, cur string) client.Transfer {
				return client.Transfer{
					ID:             id,
					Reference:      fmt.Sprintf("test%d", id),
					Status:         "outgoing_payment_sent",
					SourceAccount:  1,
					SourceCurrency: "USD",
					SourceValue:    "100",
					TargetAccount:  2,
					TargetCurrency: cur,
					TargetValue:    "100",
					User:           1,
					CreatedAt:      now,
				}
			}

			// ID 101 is HUF, which is unsupported and therefore skipped.
			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 0, 3).Return(
				[]client.Transfer{
					makeTransfer(100, "USD"),
					makeTransfer(101, "HUF"),
					makeTransfer(102, "USD"),
				},
				nil,
			)
			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 3, 3).Return(
				[]client.Transfer{
					makeTransfer(103, "USD"),
					makeTransfer(104, "USD"),
					makeTransfer(105, "USD"),
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())

			// All five supported transfers must be emitted; none trimmed away.
			refs := make([]string, 0, len(resp.Payments))
			for _, p := range resp.Payments {
				refs = append(refs, p.Reference)
			}
			Expect(refs).To(ConsistOf("100", "102", "103", "104", "105"))
			Expect(resp.HasMore).To(BeTrue())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// Offset advanced one page per page consumed (two pages of 3).
			Expect(state.Offset).To(Equal(6))
			// LastTransferID is the last emitted, covering the overflow page.
			Expect(state.LastTransferID).To(Equal(uint64(105)))
		})

		It("should keep the offset and stop when an over-fetch ends on a short final page (EN-1087)", func(ctx SpecContext) {
			// A full first page with a skipped (unsupported) transfer emits fewer
			// than pageSize payments; the next page is the last one and is short.
			// The accumulated total still exceeds pageSize, so ShouldFetchMore
			// reports hasMore=true via its over-fetch branch. We must not advance
			// the offset past the short final page nor report HasMore: otherwise
			// the stored offset lands beyond the data and skips transfers that
			// later fill the gap.
			req := models.FetchNextPaymentsRequest{
				PageSize:    3,
				FromPayload: []byte(`{"id": 0}`),
			}

			makeTransfer := func(id uint64, cur string) client.Transfer {
				return client.Transfer{
					ID:             id,
					Reference:      fmt.Sprintf("test%d", id),
					Status:         "outgoing_payment_sent",
					SourceAccount:  1,
					SourceCurrency: "USD",
					SourceValue:    "100",
					TargetAccount:  2,
					TargetCurrency: cur,
					TargetValue:    "100",
					User:           1,
					CreatedAt:      now,
				}
			}

			// Full first page (ID 201 is HUF and skipped) -> emits 200, 202.
			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 0, 3).Return(
				[]client.Transfer{
					makeTransfer(200, "USD"),
					makeTransfer(201, "HUF"),
					makeTransfer(202, "USD"),
				},
				nil,
			)
			// Short final page (2 < pageSize) -> emits 203, 204; total now 4 > 3.
			m.EXPECT().GetTransfers(gomock.Any(), uint64(0), 3, 3).Return(
				[]client.Transfer{
					makeTransfer(203, "USD"),
					makeTransfer(204, "USD"),
				},
				nil,
			)

			resp, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())

			refs := make([]string, 0, len(resp.Payments))
			for _, p := range resp.Payments {
				refs = append(refs, p.Reference)
			}
			Expect(refs).To(ConsistOf("200", "202", "203", "204"))
			// Short final page => end of data reached.
			Expect(resp.HasMore).To(BeFalse())

			var state paymentsState
			err = json.Unmarshal(resp.NewState, &state)
			Expect(err).To(BeNil())
			// Offset stays on the short final page (start = 3); it must NOT jump
			// to 6 and skip transfers that later land at offsets 4..5.
			Expect(state.Offset).To(Equal(3))
			Expect(state.LastTransferID).To(Equal(uint64(204)))
		})
	})
})
