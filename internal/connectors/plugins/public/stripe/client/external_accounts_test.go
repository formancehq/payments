package client_test

import (
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v80"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Client External Accounts", func() {
	var (
		logger   = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		cl       client.Client
		ctrl     *gomock.Controller
		b        *client.MockBackend
		timeline client.Timeline
		token    string
		err      error
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		b = client.NewMockBackend(ctrl)
		token = "dummy"
		timeline = client.Timeline{}
		b.EXPECT().Call("GET", "/v1/account", token, nil, &stripe.Account{}).DoAndReturn(
			func(_, _, _ string, _ any, account *stripe.Account) error {
				account.ID = "rootID"
				return nil
			})
		cl, err = client.New("test", logger, b, token)
		Expect(err).To(BeNil())
	})

	Context("Get External Accounts", func() {
		var (
			accountID = "accountID"
			pageSize  = 8
		)

		It("does not make external accounts call when underlying account matches the root account", func(ctx SpecContext) {
			_, _, _, err := cl.GetExternalAccounts(
				ctx,
				"rootID",
				timeline,
				int64(pageSize),
			)
			Expect(err).To(BeNil())
		})

		It("fails when underlying calls fail", func(ctx SpecContext) {
			expectedErr := errors.New("some err")

			b.EXPECT().CallRaw("GET", "/v1/accounts/accountID/external_accounts", token, gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			_, _, _, err := cl.GetExternalAccounts(
				ctx,
				accountID,
				timeline,
				int64(pageSize),
			)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("returns expected number of results in reverse chronological order and sets latest ID to newest entry", func(ctx SpecContext) {
			list := &stripe.BankAccountList{}
			expectedAccs := []*stripe.BankAccount{
				&stripe.BankAccount{
					ID: "someID3",
				},
				&stripe.BankAccount{
					ID: "someID2",
				},
				&stripe.BankAccount{
					ID: "someID1",
				},
			}

			callCount := 0
			b.EXPECT().CallRaw("GET", "/v1/accounts/accountID/external_accounts", token, gomock.Any(), gomock.Any(), list).MaxTimes(2).DoAndReturn(func(
				method, path, token string, p, p2 any, l *stripe.BankAccountList,
			) error {
				// called once by timeline scan to find oldest entry and 2nd time to fetch enough results to fill the page
				results := expectedAccs[0 : len(expectedAccs)-callCount]
				l.Data = append(l.Data, results...)
				l.ListMeta = stripe.ListMeta{HasMore: false, TotalCount: uint32(len(l.Data))}
				callCount++
				return nil
			})
			trxs, updatedTimeline, hasMore, err := cl.GetExternalAccounts(
				ctx,
				accountID,
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
