package krakenpro

import (
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Krakenpro Balances", func() {
	var (
		p      *Plugin
		m      *client.MockClient
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		p = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			client: m,
			logger: logger,
			config: Config{
				APIKey: "test-api-key",
			},
			accountRef: "kraken-test12345",
		}
	})

	Context("fetch next balances", func() {
		It("should return balances for all assets", func(ctx SpecContext) {
			m.EXPECT().GetBalance(gomock.Any()).Return(
				&client.BalanceResponse{
					Error: nil,
					Result: map[string]string{
						"ZUSD": "171288.6158",
						"XXBT": "0.0120190800",
					},
				},
				nil,
			)

			req := models.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
			}

			resp, err := p.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(2))
			Expect(resp.HasMore).To(BeFalse())

			// Balances should be sorted by asset key
			// BTC (from XXBT) comes before USD (from ZUSD) alphabetically by original key
			for _, bal := range resp.Balances {
				Expect(bal.AccountReference).To(Equal("kraken-test12345"))
				Expect(bal.Amount).ToNot(BeNil())
			}
		})

		It("should skip zero balances", func(ctx SpecContext) {
			m.EXPECT().GetBalance(gomock.Any()).Return(
				&client.BalanceResponse{
					Error: nil,
					Result: map[string]string{
						"ZUSD": "0.0000",
						"XXBT": "0.0120190800",
					},
				},
				nil,
			)

			req := models.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
			}

			resp, err := p.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
		})

		It("should skip unsupported assets", func(ctx SpecContext) {
			m.EXPECT().GetBalance(gomock.Any()).Return(
				&client.BalanceResponse{
					Error: nil,
					Result: map[string]string{
						"UNKNOWN": "1.23",
						"ZUSD":    "10.00",
					},
				},
				nil,
			)

			req := models.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
			}

			resp, err := p.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.Balances[0].Asset).To(Equal("USD/2"))
		})

		It("should handle API error", func(ctx SpecContext) {
			m.EXPECT().GetBalance(gomock.Any()).Return(
				nil,
				&client.KrakenError{Errors: []string{"EAPI:Invalid key"}},
			)

			req := models.FetchNextBalancesRequest{
				State: json.RawMessage(`{}`),
			}

			_, err := p.FetchNextBalances(ctx, req)
			Expect(err).To(HaveOccurred())
		})
	})
})
