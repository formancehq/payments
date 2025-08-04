package client_test

import (
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v79"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Client Payments", func() {
	var (
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		cl     client.Client
		ctrl   *gomock.Controller
		b      *client.MockBackend
		token  string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		b = client.NewMockBackend(ctrl)
		token = "dummy"
		cl = client.New("test", logger, b, token)
	})

	Context("Get Payments", func() {
		var (
			accountID = "someAccount"
			timeline  = client.Timeline{}
			pageSize  = 8
		)

		It("fails when underlying calls fail", func(ctx SpecContext) {
			expectedErr := errors.New("some err")

			b.EXPECT().CallRaw("GET", "/v1/balance_transactions", token, gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			_, _, _, err := cl.GetPayments(
				ctx,
				accountID,
				timeline,
				int64(pageSize),
			)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("returns expected number of results in chronological order and sets latest ID to newest entry", func(ctx SpecContext) {
			list := &stripe.BalanceTransactionList{}
			expectedBTs := []*stripe.BalanceTransaction{
				&stripe.BalanceTransaction{
					ID:     "someID3",
					Source: &stripe.BalanceTransactionSource{},
				},
				&stripe.BalanceTransaction{
					ID:     "someID2",
					Source: &stripe.BalanceTransactionSource{},
				},
				&stripe.BalanceTransaction{
					ID:     "someID1",
					Source: &stripe.BalanceTransactionSource{},
				},
			}

			callCount := 0
			b.EXPECT().CallRaw("GET", "/v1/balance_transactions", token, gomock.Any(), gomock.Any(), list).MaxTimes(2).DoAndReturn(func(
				method, path, token string, p, p2 any, l *stripe.BalanceTransactionList,
			) error {
				// called once by timeline scan to find oldest entry and 2nd time to fetch enough results to fill the page
				results := expectedBTs[0 : len(expectedBTs)-callCount]
				l.Data = append(l.Data, results...)
				l.ListMeta = stripe.ListMeta{HasMore: false, TotalCount: uint32(len(l.Data))}
				callCount++
				return nil
			})
			trxs, updatedTimeline, hasMore, err := cl.GetPayments(
				ctx,
				accountID,
				timeline,
				int64(pageSize),
			)
			Expect(err).To(BeNil())

			Expect(hasMore).To(BeFalse())
			Expect(trxs).To(HaveLen(len(expectedBTs)))
			Expect(trxs[0].ID).To(Equal("someID1"))
			Expect(trxs[1].ID).To(Equal("someID2"))
			Expect(trxs[2].ID).To(Equal("someID3"))
			Expect(updatedTimeline.LatestID).To(Equal("someID3"))
		})
	})
})
