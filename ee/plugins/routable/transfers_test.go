package routable

import (
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Routable createTransfer", func() {
	var (
		ctrl   *gomock.Controller
		mock   *client.MockClient
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mock = client.NewMockClient(ctrl)
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			name:   "routable",
			logger: logger,
			client: mock,
			config: Config{ActingTeamMember: "tm_default"},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	pi := func() models.PSPPaymentInitiation {
		return models.PSPPaymentInitiation{
			Reference:          "pi_2",
			CreatedAt:          time.Now().UTC(),
			Amount:             big.NewInt(5000), // 50.00 USD
			Asset:              "USD/2",
			SourceAccount:      &models.PSPAccount{Reference: "acc_1"},
			DestinationAccount: &models.PSPAccount{Reference: "co_1"},
		}
	}

	It("marks the response Payment as TRANSFER even though the rail is a Routable payable", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).Return(
			&client.Payable{ID: "pa_t", Status: "completed", Amount: "50.00", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			http.StatusCreated,
			nil,
		)
		resp, err := plg.createTransfer(ctx, models.CreateTransferRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil())
		Expect(resp.Payment).NotTo(BeNil())
		Expect(resp.Payment.Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
	})

	It("returns PollingTransferID for sync 201 responses with a non-terminal status", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).Return(
			&client.Payable{ID: "pa_t2", Status: "pending", Amount: "50.00", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			http.StatusCreated,
			nil,
		)
		resp, err := plg.createTransfer(ctx, models.CreateTransferRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil())
		Expect(resp.Payment).To(BeNil())
		Expect(resp.PollingTransferID).NotTo(BeNil())
		Expect(*resp.PollingTransferID).To(Equal("pa_t2"))
	})

	// Async 202 path: Routable echoes only {id}. The plugin must return
	// PollingTransferID without trying to map the half-empty payable
	// (which would error out on the missing currency / amount).
	It("returns PollingTransferID for async 202 responses with no mapping attempted", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).Return(
			&client.Payable{ID: "pa_t3"}, // 202 body: just {id}, no amount/currency
			http.StatusAccepted,
			nil,
		)
		resp, err := plg.createTransfer(ctx, models.CreateTransferRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil(), "must NOT error on the missing currency in a 202 body")
		Expect(resp.Payment).To(BeNil())
		Expect(resp.PollingTransferID).NotTo(BeNil())
		Expect(*resp.PollingTransferID).To(Equal("pa_t3"))
	})
})
