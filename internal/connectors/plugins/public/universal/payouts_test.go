package universal_test

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Universal *Plugin — payouts/transfers", func() {
	var (
		ctrl   *gomock.Controller
		mc     *client.MockClient
		plg    *universal.Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		cfg    = json.RawMessage(`{"endpoint":"https://x","apiKey":"k"}`)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mc = client.NewMockClient(ctrl)
		mc.EXPECT().SetIdempotencyHeader(gomock.Any()).AnyTimes()
		var err error
		plg, err = universal.New("universal-test", logger, cfg)
		Expect(err).To(BeNil())
		universal.InjectClient(plg, mc)
		universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_CREATE_PAYOUT, models.CAPABILITY_CREATE_TRANSFER})
	})

	AfterEach(func() { ctrl.Finish() })

	pi := func() models.PSPPaymentInitiation {
		return models.PSPPaymentInitiation{
			Reference:          "ref-1",
			CreatedAt:          time.Now().UTC(),
			Description:        "test",
			SourceAccount:      &models.PSPAccount{Reference: "acct_src"},
			DestinationAccount: &models.PSPAccount{Reference: "acct_dst"},
			Amount:             big.NewInt(1234),
			Asset:              "EUR/2",
		}
	}

	Context("CreatePayout", func() {
		It("returns terminal payment", func(ctx SpecContext) {
			mc.EXPECT().CreatePayout(gomock.Any(), "ref-1", gomock.Any()).Return(&client.PayoutResponse{
				Mode: "terminal",
				Payment: &client.Payment{
					Reference: "ref-1", CreatedAt: time.Now().UTC(),
					Type: "PAYOUT", Status: "SUCCEEDED", Amount: "1234", Asset: "EUR/2",
				},
			}, nil)
			res, err := plg.CreatePayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
			Expect(err).To(BeNil())
			Expect(res.Payment).NotTo(BeNil())
			Expect(res.Payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(res.PollingPayoutID).To(BeNil())
		})
		It("returns polling ID for async flow", func(ctx SpecContext) {
			mc.EXPECT().CreatePayout(gomock.Any(), "ref-1", gomock.Any()).Return(&client.PayoutResponse{
				Mode: "polling", PollingID: "ext-99",
			}, nil)
			res, err := plg.CreatePayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
			Expect(err).To(BeNil())
			Expect(res.Payment).To(BeNil())
			Expect(*res.PollingPayoutID).To(Equal("ext-99"))
		})
		It("rejects bad payment initiation", func(ctx SpecContext) {
			bad := pi()
			bad.Amount = big.NewInt(0)
			_, err := plg.CreatePayout(ctx, models.CreatePayoutRequest{PaymentInitiation: bad})
			Expect(err).NotTo(BeNil())
		})
	})

	Context("PollPayoutStatus", func() {
		It("returns nil/nil to keep polling when payment not yet final", func(ctx SpecContext) {
			mc.EXPECT().GetPayout(gomock.Any(), "ext-99").Return(&client.PayoutResponse{Mode: "polling"}, nil)
			res, err := plg.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{PayoutID: "ext-99"})
			Expect(err).To(BeNil())
			Expect(res.Payment).To(BeNil())
			Expect(res.Error).To(BeNil())
		})
		It("returns terminal Payment", func(ctx SpecContext) {
			mc.EXPECT().GetPayout(gomock.Any(), "ext-99").Return(&client.PayoutResponse{
				Mode: "polling",
				Payment: &client.Payment{
					Reference: "ref-1", CreatedAt: time.Now().UTC(),
					Type: "PAYOUT", Status: "SUCCEEDED", Amount: "1234", Asset: "EUR/2",
				},
			}, nil)
			res, err := plg.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{PayoutID: "ext-99"})
			Expect(err).To(BeNil())
			Expect(res.Payment).NotTo(BeNil())
			Expect(res.Payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
		})
	})

	Context("CreateTransfer", func() {
		It("returns terminal payment", func(ctx SpecContext) {
			mc.EXPECT().CreateTransfer(gomock.Any(), "ref-1", gomock.Any()).Return(&client.TransferResponse{
				Mode: "terminal",
				Payment: &client.Payment{
					Reference: "ref-1", CreatedAt: time.Now().UTC(),
					Type: "TRANSFER", Status: "SUCCEEDED", Amount: "1234", Asset: "EUR/2",
				},
			}, nil)
			res, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{PaymentInitiation: pi()})
			Expect(err).To(BeNil())
			Expect(res.Payment).NotTo(BeNil())
			Expect(res.Payment.Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
		})
	})

	Context("ReverseTransfer", func() {
		It("returns terminal payment with REFUNDED status", func(ctx SpecContext) {
			mc.EXPECT().ReverseTransfer(gomock.Any(), "rev-1", "init-1", gomock.Any()).Return(&client.TransferResponse{
				Payment: &client.Payment{
					Reference: "p1", CreatedAt: time.Now().UTC(),
					Type: "TRANSFER", Status: "REFUNDED", Amount: "100", Asset: "EUR/2",
				},
			}, nil)
			res, err := plg.ReverseTransfer(ctx, models.ReverseTransferRequest{PaymentInitiationReversal: revFixture()})
			Expect(err).To(BeNil())
			Expect(res.Payment.Status).To(Equal(models.PAYMENT_STATUS_REFUNDED))
		})
		It("errors when counterparty omits the payment", func(ctx SpecContext) {
			mc.EXPECT().ReverseTransfer(gomock.Any(), "rev-1", "init-1", gomock.Any()).Return(&client.TransferResponse{}, nil)
			_, err := plg.ReverseTransfer(ctx, models.ReverseTransferRequest{PaymentInitiationReversal: revFixture()})
			Expect(err).NotTo(BeNil())
		})
	})
})

// revFixture is shared between payouts_test.go and bank_accounts_test.go.
func revFixture() models.PSPPaymentInitiationReversal {
	return models.PSPPaymentInitiationReversal{
		Reference:                "rev-1",
		CreatedAt:                time.Now().UTC(),
		Description:              "test reverse",
		RelatedPaymentInitiation: models.PSPPaymentInitiation{Reference: "init-1"},
		Amount:                   big.NewInt(100),
		Asset:                    "EUR/2",
	}
}
