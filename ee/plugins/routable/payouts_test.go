package routable

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Routable createPayout / pollPayableStatus", func() {
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
			Reference:          "pi_1",
			CreatedAt:          time.Now().UTC(),
			Description:        "rent",
			Amount:             big.NewInt(12345), // 123.45 USD
			Asset:              "USD/2",
			SourceAccount:      &models.PSPAccount{Reference: "acc_1"},
			DestinationAccount: &models.PSPAccount{Reference: "co_1"},
		}
	}

	It("returns PollingPayoutID for non-terminal payables", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, error) {
			Expect(req.Type).To(Equal(defaultPayableType))
			Expect(req.DeliveryMethod).To(Equal(defaultDeliveryMethod))
			Expect(req.PayToCompany).To(Equal("co_1"))
			Expect(req.WithdrawFromAccount).To(Equal("acc_1"))
			Expect(req.Amount).To(Equal("123.45"))
			Expect(req.CurrencyCode).To(Equal("USD"))
			Expect(req.ActingTeamMember).To(Equal("tm_default"))
			Expect(req.IdempotencyKey).To(Equal("pi_1"))
			return &client.Payable{ID: "pa_1", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, nil
		})

		resp, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil())
		Expect(resp.Payment).To(BeNil())
		Expect(resp.PollingPayoutID).NotTo(BeNil())
		Expect(*resp.PollingPayoutID).To(Equal("pa_1"))
	})

	It("returns the Payment immediately when the response is terminal", func(ctx SpecContext) {
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).Return(
			&client.Payable{ID: "pa_2", Status: "completed", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			nil,
		)
		resp, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: pi()})
		Expect(err).To(BeNil())
		Expect(resp.PollingPayoutID).To(BeNil())
		Expect(resp.Payment).NotTo(BeNil())
		Expect(resp.Payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
	})

	It("respects metadata overrides for type, delivery_method, and acting_team_member", func(ctx SpecContext) {
		piWithOverrides := pi()
		piWithOverrides.Metadata = map[string]string{
			MetadataKeyType:             "wire",
			MetadataKeyDeliveryMethod:   "wire",
			MetadataKeyActingTeamMember: "tm_override",
			MetadataKeyExternalID:       "ext_42",
			MetadataKeyMemo:             "Q1 invoice",
		}
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, error) {
			Expect(req.Type).To(Equal("wire"))
			Expect(req.DeliveryMethod).To(Equal("wire"))
			Expect(req.ActingTeamMember).To(Equal("tm_override"))
			Expect(req.ExternalID).To(Equal("ext_42"))
			Expect(req.Memo).To(Equal("Q1 invoice"))
			return &client.Payable{ID: "pa_3", Status: "processing", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, nil
		})
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: piWithOverrides})
		Expect(err).To(BeNil())
	})

	It("rejects payment initiations with no source/destination", func(ctx SpecContext) {
		bad := pi()
		bad.SourceAccount = nil
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: bad})
		Expect(err).To(HaveOccurred())
	})

	It("falls back to the per-request metadata acting_team_member when the config is empty", func(ctx SpecContext) {
		plg.config = Config{} // no connector-level default
		piWithTM := pi()
		piWithTM.Metadata = map[string]string{MetadataKeyActingTeamMember: "tm_from_metadata"}
		mock.EXPECT().CreatePayable(gomock.Any(), gomock.Any()).DoAndReturn(func(_ any, req client.CreatePayableRequest) (*client.Payable, error) {
			Expect(req.ActingTeamMember).To(Equal("tm_from_metadata"))
			return &client.Payable{ID: "pa_md", Status: "pending", Amount: "123.45", CurrencyCode: "USD", CreatedAt: time.Now().UTC()}, nil
		})
		_, err := plg.createPayout(ctx, models.CreatePayoutRequest{PaymentInitiation: piWithTM})
		Expect(err).To(BeNil())
	})

	It("polls and returns the Payment when the payable is terminal", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_1").Return(
			&client.Payable{ID: "pa_1", Status: "completed", Amount: "10.00", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			nil,
		)
		resp, err := plg.pollPayableStatus(ctx, "pa_1")
		Expect(err).To(BeNil())
		Expect(resp.Payment).NotTo(BeNil())
		Expect(resp.Error).To(BeNil())
	})

	It("returns an Error string for failed terminal states", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_x").Return(
			&client.Payable{ID: "pa_x", Status: "failed", Amount: "10.00", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			nil,
		)
		resp, err := plg.pollPayableStatus(ctx, "pa_x")
		Expect(err).To(BeNil())
		Expect(resp.Payment).To(BeNil())
		Expect(resp.Error).NotTo(BeNil())
		Expect(*resp.Error).To(ContainSubstring("FAILED"))
	})

	It("returns an empty response (engine retries later) on 404 ErrPayableNotFound", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_pending").Return(nil, client.ErrPayableNotFound)
		resp, err := plg.pollPayableStatus(ctx, "pa_pending")
		Expect(err).To(BeNil())
		Expect(resp.Payment).To(BeNil())
		Expect(resp.Error).To(BeNil())
	})

	It("propagates other client errors", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_y").Return(nil, errors.New("boom"))
		_, err := plg.pollPayableStatus(ctx, "pa_y")
		Expect(err).To(HaveOccurred())
	})

	It("returns nothing while still pending", func(ctx SpecContext) {
		mock.EXPECT().GetPayable(gomock.Any(), "pa_z").Return(
			&client.Payable{ID: "pa_z", Status: "pending", Amount: "10.00", CurrencyCode: "USD", CreatedAt: time.Now().UTC()},
			nil,
		)
		resp, err := plg.pollPayableStatus(ctx, "pa_z")
		Expect(err).To(BeNil())
		Expect(resp.Payment).To(BeNil())
		Expect(resp.Error).To(BeNil())
	})
})
