package client_test

import (
	logging "github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ce/plugins/stripe/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v80"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Client Reversals", func() {
	var (
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		cl     client.Client
		ctrl   *gomock.Controller
		b      *client.MockBackend
		token  string
		err    error
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		b = client.NewMockBackend(ctrl)
		token = "dummy"
		b.EXPECT().Call("GET", "/v1/account", token, nil, &stripe.Account{}).DoAndReturn(
			func(_, _, _ string, _ any, account *stripe.Account) error {
				account.ID = "rootID"
				return nil
			})
		cl, err = client.New("test", logger, b, token)
		Expect(err).To(BeNil())
	})

	Context("Reverse Transfer", func() {
		// Regression: New left transferReversalClient uninitialized (nil
		// backend, empty key), so every ReverseTransfer call panicked with a
		// nil pointer dereference inside the Stripe SDK. Found by the contract
		// test against the live test-mode API.
		It("calls the reversal endpoint through an initialized sub-client", func(ctx SpecContext) {
			b.EXPECT().Call("POST", "/v1/transfers/tr_123/reversals", token, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_, _, _ string, _ any, v *stripe.TransferReversal) error {
					v.ID = "trr_123"
					return nil
				})

			resp, err := cl.ReverseTransfer(ctx, client.ReverseTransferRequest{
				StripeTransferID: "tr_123",
				Amount:           1,
				Description:      "test",
			})
			Expect(err).To(BeNil())
			Expect(resp.ID).To(Equal("trr_123"))
		})
	})
})
