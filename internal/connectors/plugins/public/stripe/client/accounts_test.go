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

var _ = Describe("Stripe Client Accounts", func() {
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

	Context("Get Accounts", func() {
		var (
			timeline = client.Timeline{}
			pageSize = 8
		)

		It("fails when underlying calls fail", func(ctx SpecContext) {
			expectedErr := errors.New("some err")

			b.EXPECT().CallRaw("GET", "/v1/accounts", token, gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			_, _, _, err := cl.GetAccounts(
				ctx,
				timeline,
				int64(pageSize),
			)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("returns expected number of results in reverse chronological order and sets latest ID to newest entry", func(ctx SpecContext) {
			list := &stripe.AccountList{}
			expectedAccs := []*stripe.Account{
				&stripe.Account{
					ID: "someID3",
				},
				&stripe.Account{
					ID: "someID2",
				},
				&stripe.Account{
					ID: "someID1",
				},
			}

			callCount := 0
			b.EXPECT().CallRaw("GET", "/v1/accounts", token, gomock.Any(), gomock.Any(), list).MaxTimes(2).DoAndReturn(func(
				method, path, token string, p, p2 any, l *stripe.AccountList,
			) error {
				// called once by timeline scan to find oldest entry and 2nd time to fetch enough results to fill the page
				results := expectedAccs[0 : len(expectedAccs)-callCount]
				l.Data = append(l.Data, results...)
				l.ListMeta = stripe.ListMeta{HasMore: false, TotalCount: uint32(len(l.Data))}
				callCount++
				return nil
			})
			trxs, updatedTimeline, hasMore, err := cl.GetAccounts(
				ctx,
				timeline,
				int64(pageSize),
			)
			Expect(err).To(BeNil())

			Expect(hasMore).To(BeFalse())
			Expect(trxs).To(HaveLen(len(expectedAccs)))
			Expect(trxs[0].ID).To(Equal("someID3"))
			Expect(trxs[1].ID).To(Equal("someID2"))
			Expect(trxs[2].ID).To(Equal("someID1"))
			Expect(updatedTimeline.LatestID).To(Equal("someID3"))
			Expect(updatedTimeline.BacklogStartingPoint).To(Equal("someID3"))
		})
	})
})
