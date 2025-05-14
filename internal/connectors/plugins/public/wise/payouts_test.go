package wise

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Wise Plugin Payouts Creation", func() {
	var (
		m   *client.MockClient
		plg models.Plugin
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	Context("create payout", func() {
		var (
			samplePSPPaymentInitiation models.PSPPaymentInitiation
			now                        time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			samplePSPPaymentInitiation = models.PSPPaymentInitiation{
				Reference:   "test1",
				CreatedAt:   now.UTC(),
				Description: "test1",
				SourceAccount: &models.PSPAccount{
					Reference:    "acc1",
					CreatedAt:    now.Add(-time.Duration(50) * time.Minute).UTC(),
					Name:         pointer.For("acc1"),
					DefaultAsset: pointer.For("EUR/2"),
					Metadata: map[string]string{
						"profile_id": "1",
					},
				},
				DestinationAccount: &models.PSPAccount{
					Reference:    "acc2",
					CreatedAt:    now.Add(-time.Duration(49) * time.Minute).UTC(),
					Name:         pointer.For("acc2"),
					DefaultAsset: pointer.For("EUR/2"),
					Metadata: map[string]string{
						"profile_id": "2",
					},
				},
				Amount: big.NewInt(100),
				Asset:  "EUR/2",
				Metadata: map[string]string{
					"foo": "bar",
				},
			}
		})

		It("should return an error - validation error - source account", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.SourceAccount = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("source account is required in transfer/payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - destination account", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("destination account is required in transfer/payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - missing source account profile id in metadata", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.SourceAccount.Metadata = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("source account metadata with profile id is required in transfer/payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - invalid source account profile id in metadata", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.SourceAccount.Metadata["profile_id"] = "invalid"

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("source account metadata with profile id is required as an integer in transfer/payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - missing destination account profile id in metadata", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount.Metadata = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("destination account metadata with profile id is required in transfer/payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - invalid destination account profile id in metadata", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount.Metadata["profile_id"] = "invalid"

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("destination account metadata with profile id is required as an integer in transfer/payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - asset not supported", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Asset = "HUF/2"

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get currency and precision from asset: HUF: missing currencies: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - create quote error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().CreateQuote(gomock.Any(), "1", "EUR", json.Number("1.00")).Return(client.Quote{}, errors.New("test error"))

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - create transfer error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			quote := client.Quote{
				ID: uuid.New(),
			}
			m.EXPECT().CreateQuote(gomock.Any(), "1", "EUR", json.Number("1.00")).Return(quote, nil)
			m.EXPECT().CreatePayout(gomock.Any(), quote, uint64(2), "test1").Return(nil, errors.New("test error"))

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			trResponse := client.Payout{
				ID:              123,
				Status:          "outgoing_payment_sent",
				TargetAccount:   2,
				TargetCurrency:  "EUR",
				TargetValue:     "1.00",
				SourceBalanceID: 1,
				CreatedAt:       now,
			}
			quote := client.Quote{
				ID: uuid.New(),
			}
			m.EXPECT().CreateQuote(gomock.Any(), "1", "EUR", json.Number("1.00")).Return(quote, nil)
			m.EXPECT().CreatePayout(gomock.Any(), quote, uint64(2), "test1").Return(&trResponse, nil)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				Payment: &models.PSPPayment{
					Reference:                   "123",
					CreatedAt:                   now,
					Type:                        models.PAYMENT_TYPE_PAYOUT,
					Amount:                      big.NewInt(100),
					Asset:                       "EUR/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("1"),
					DestinationAccountReference: pointer.For("2"),
					Raw:                         raw,
				},
			}))
		})
	})
})
