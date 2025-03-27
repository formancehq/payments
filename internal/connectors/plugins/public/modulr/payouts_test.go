package modulr

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/modulr/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Modulr Plugin Payouts Creation", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create payout", func() {
		var (
			m                          *client.MockClient
			samplePSPPaymentInitiation models.PSPPaymentInitiation
			now                        time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now, _ = time.Parse("2006-01-02T15:04:05.999-0700", time.Now().UTC().Format("2006-01-02T15:04:05.999-0700"))

			samplePSPPaymentInitiation = models.PSPPaymentInitiation{
				Reference:   "test1",
				CreatedAt:   now,
				Description: "test1",
				SourceAccount: &models.PSPAccount{
					Reference:    "acc1",
					CreatedAt:    now.Add(-time.Duration(50) * time.Minute).UTC(),
					Name:         pointer.For("acc1"),
					DefaultAsset: pointer.For("EUR/2"),
				},
				DestinationAccount: &models.PSPAccount{
					Reference:    "acc2",
					CreatedAt:    now.Add(-time.Duration(49) * time.Minute).UTC(),
					Name:         pointer.For("acc2"),
					DefaultAsset: pointer.For("EUR/2"),
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

		It("should return an error - initiate payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().InitiatePayout(gomock.Any(), &client.PayoutRequest{
				IdempotencyKey:  samplePSPPaymentInitiation.Reference,
				SourceAccountID: samplePSPPaymentInitiation.SourceAccount.Reference,
				Destination: client.Destination{
					Type: "BENEFICIARY",
					ID:   samplePSPPaymentInitiation.DestinationAccount.Reference,
				},
				Currency:          "EUR",
				Amount:            "1.00",
				Reference:         samplePSPPaymentInitiation.Description,
				ExternalReference: samplePSPPaymentInitiation.Description,
			}).Return(nil, errors.New("test error"))

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			trResponse := client.PayoutResponse{
				ID:                "1",
				Status:            "PROCESSED",
				CreatedDate:       now.Format("2006-01-02T15:04:05.999-0700"),
				ExternalReference: samplePSPPaymentInitiation.Description,
				Details: client.Details{
					SourceAccountID: samplePSPPaymentInitiation.SourceAccount.Reference,
					Destination: client.Destination{
						Type: "BENEFICIARY",
						ID:   samplePSPPaymentInitiation.DestinationAccount.Reference,
					},
					Currency: "EUR",
					Amount:   "1.00",
				},
			}
			m.EXPECT().InitiatePayout(gomock.Any(), &client.PayoutRequest{
				IdempotencyKey:  samplePSPPaymentInitiation.Reference,
				SourceAccountID: samplePSPPaymentInitiation.SourceAccount.Reference,
				Destination: client.Destination{
					Type: "BENEFICIARY",
					ID:   samplePSPPaymentInitiation.DestinationAccount.Reference,
				},
				Currency:          "EUR",
				Amount:            "1.00",
				Reference:         samplePSPPaymentInitiation.Description,
				ExternalReference: samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				Payment: &models.PSPPayment{
					Reference:                   "1",
					CreatedAt:                   now,
					Type:                        models.PAYMENT_TYPE_PAYOUT,
					Amount:                      big.NewInt(100),
					Asset:                       "EUR/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("acc1"),
					DestinationAccountReference: pointer.For("acc2"),
					Raw:                         raw,
				},
			}))
		})
	})
})
