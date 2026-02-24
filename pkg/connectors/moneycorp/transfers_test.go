package moneycorp

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/moneycorp/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp *Plugin Transfers Creation", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  connector.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("create transfer", func() {
		var (
			samplePSPPaymentInitiation connector.PSPPaymentInitiation
			now                        time.Time
		)

		BeforeEach(func() {
			now = time.Now().UTC()

			samplePSPPaymentInitiation = connector.PSPPaymentInitiation{
				Reference:   "test1",
				CreatedAt:   now.UTC(),
				Description: "test1",
				SourceAccount: &connector.PSPAccount{
					Reference:    "123",
					CreatedAt:    now.Add(-time.Duration(50) * time.Minute).UTC(),
					Name:         pointer.For("acc1"),
					DefaultAsset: pointer.For("EUR/2"),
				},
				DestinationAccount: &connector.PSPAccount{
					Reference:    "321",
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
			req := connector.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.SourceAccount = nil

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("source account is required in transfer/payout request: invalid request"))
			Expect(resp).To(Equal(connector.CreateTransferResponse{}))
		})

		It("should return an error - validation error - destination account", func(ctx SpecContext) {
			req := connector.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount = nil

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("destination account is required in transfer/payout request: invalid request"))
			Expect(resp).To(Equal(connector.CreateTransferResponse{}))
		})

		It("should return an error - validation error - asset not supported", func(ctx SpecContext) {
			req := connector.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Asset = "HUF/2"

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get currency and precision from asset: HUF: missing currencies: invalid request"))
			Expect(resp).To(Equal(connector.CreateTransferResponse{}))
		})

		It("should return an error - initiate transfer error", func(ctx SpecContext) {
			req := connector.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().InitiateTransfer(gomock.Any(), &client.TransferRequest{
				IdempotencyKey:     samplePSPPaymentInitiation.Reference,
				SourceAccountID:    samplePSPPaymentInitiation.SourceAccount.Reference,
				ReceivingAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				TransferAmount:     "1.00",
				TransferCurrency:   "EUR",
				TransferReference:  samplePSPPaymentInitiation.Description,
				ClientReference:    samplePSPPaymentInitiation.Description,
			}).Return(nil, errors.New("test error"))

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(connector.CreateTransferResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := connector.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			trResponse := client.TransferResponse{
				ID: "1",
				Attributes: client.TransferAttributes{
					SendingAccountID:   123,
					ReceivingAccountID: 321,
					CreatedAt:          now.Format("2006-01-02T15:04:05.999999999"),
					TransferReference:  samplePSPPaymentInitiation.Description,
					ClientReference:    samplePSPPaymentInitiation.Description,
					TransferAmount:     "1.00",
					TransferCurrency:   "EUR",
					TransferStatus:     "Cleared",
				},
			}
			m.EXPECT().InitiateTransfer(gomock.Any(), &client.TransferRequest{
				IdempotencyKey:     samplePSPPaymentInitiation.Reference,
				SourceAccountID:    samplePSPPaymentInitiation.SourceAccount.Reference,
				ReceivingAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				TransferAmount:     "1.00",
				TransferCurrency:   "EUR",
				TransferReference:  samplePSPPaymentInitiation.Description,
				ClientReference:    samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(connector.CreateTransferResponse{
				Payment: &connector.PSPPayment{
					Reference:                   "1",
					CreatedAt:                   now,
					Type:                        connector.PAYMENT_TYPE_TRANSFER,
					Amount:                      big.NewInt(100),
					Asset:                       "EUR/2",
					Scheme:                      connector.PAYMENT_SCHEME_OTHER,
					Status:                      connector.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("123"),
					DestinationAccountReference: pointer.For("321"),
					Raw:                         raw,
				},
			}))
		})
	})
})
