package coinbase

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbase/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Plugin Balances", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			client: m,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next balances", func() {
		var sampleAccounts []client.Account

		BeforeEach(func() {
			sampleAccounts = []client.Account{
				{
					ID:             "acc1",
					Currency:       "BTC",
					Balance:        "1.5",
					Available:      "1.0",
					Hold:           "0.5",
					ProfileID:      "profile1",
					TradingEnabled: true,
				},
				{
					ID:             "acc2",
					Currency:       "USD",
					Balance:        "1000.50",
					Available:      "900.00",
					Hold:           "100.50",
					ProfileID:      "profile1",
					TradingEnabled: true,
				},
			}
		})

		It("should return an error - missing from payload", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				FromPayload: nil,
			}

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing from payload"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should return an error - get accounts error", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{Reference: "acc1"})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			m.EXPECT().GetAccounts(gomock.Any()).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should fetch BTC balance with correct precision", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{Reference: "acc1"})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			m.EXPECT().GetAccounts(gomock.Any()).Return(
				sampleAccounts,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())

			// BTC has 8 decimals, so 1.5 BTC = 150000000 (1.5 * 10^8)
			Expect(resp.Balances[0].Asset).To(Equal("BTC/8"))
			Expect(resp.Balances[0].Amount.Cmp(big.NewInt(150000000))).To(Equal(0))
			Expect(resp.Balances[0].AccountReference).To(Equal("acc1"))
		})

		It("should fetch USD balance with correct precision", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{Reference: "acc2"})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			m.EXPECT().GetAccounts(gomock.Any()).Return(
				sampleAccounts,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))

			// USD has 2 decimals, so 1000.50 USD = 100050 (1000.50 * 10^2)
			Expect(resp.Balances[0].Asset).To(Equal("USD/2"))
			Expect(resp.Balances[0].Amount.Cmp(big.NewInt(100050))).To(Equal(0))
		})

		It("should return empty balances for unsupported currency", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{Reference: "acc-unknown"})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			unsupportedAccounts := []client.Account{
				{
					ID:       "acc-unknown",
					Currency: "UNKNOWN_CURRENCY",
					Balance:  "100",
				},
			}

			m.EXPECT().GetAccounts(gomock.Any()).Return(
				unsupportedAccounts,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(0))
		})

		It("should return empty balances for non-existent account", func(ctx SpecContext) {
			fromPayload, _ := json.Marshal(models.PSPAccount{Reference: "non-existent"})
			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			m.EXPECT().GetAccounts(gomock.Any()).Return(
				sampleAccounts,
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(0))
		})
	})
})
