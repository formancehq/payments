package wise

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin Payments", func() {
	var (
		plg *Plugin
		m   *client.MockClient
	)

	BeforeEach(func() {
		plg = &Plugin{}

		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg.SetClient(m)
	})

	Context("fetch next payments", func() {
		var (
			transfers         []client.Transfer
			expectedProfileID uint64
		)

		BeforeEach(func() {
			expectedProfileID = 111
			transfers = []client.Transfer{
				{ID: 1, Reference: "ref1", TargetValue: json.Number("25"), TargetCurrency: "EUR"},
				{ID: 2, Reference: "ref2", TargetValue: json.Number("44"), TargetCurrency: "DKK"},
				{ID: 3, Reference: "ref2", TargetValue: json.Number("61"), TargetCurrency: "EEK"}, // skipped due to unsupported currency
				{ID: 4, Reference: "ref2", TargetValue: json.Number("95"), TargetCurrency: "CAD"},
			}
		})

		It("fetches payments from wise", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State:       json.RawMessage(`{}`),
				FromPayload: json.RawMessage(fmt.Sprintf(`{"ID":%d}`, expectedProfileID)),
				PageSize:    len(transfers),
			}
			m.EXPECT().GetTransfers(ctx, expectedProfileID, 0, req.PageSize).Return(
				transfers,
				nil,
			)
			m.EXPECT().GetTransfers(ctx, expectedProfileID, 4, req.PageSize).Return(
				[]client.Transfer{},
				nil,
			)

			res, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeFalse())
			Expect(res.Payments).To(HaveLen(req.PageSize - 1))
			Expect(res.Payments[0].Reference).To(Equal(fmt.Sprint(transfers[0].ID)))
			expectedAmount, err := transfers[0].TargetValue.Int64()
			Expect(err).To(BeNil())
			Expect(res.Payments[0].Amount).To(Equal(big.NewInt(expectedAmount * 100))) // after conversion to minors
			Expect(res.Payments[1].Reference).To(Equal(fmt.Sprint(transfers[1].ID)))
			Expect(res.Payments[2].Reference).To(Equal(fmt.Sprint(transfers[3].ID)))

			var state paymentsState

			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(fmt.Sprint(state.Offset)).To(Equal(res.Payments[len(res.Payments)-1].Reference))
		})
	})
})
