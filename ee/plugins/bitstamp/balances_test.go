package bitstamp

import (
	"encoding/json"
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Bitstamp Plugin Balances", func() {
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
			logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			currencies: map[string]int{
				"USD": 2,
				"BTC": 8,
				"ETH": 18,
			},
		}
		plg.currLoaded.Store(true)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next balances", func() {
		It("should return an error when from payload is missing", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{
				FromPayload: nil,
			}

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("missing from payload"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should return an error - get balances error", func(ctx SpecContext) {
			from := models.PSPAccount{Reference: "BTC"}
			fromPayload, _ := json.Marshal(from)

			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
				nil,
				errors.New("test error"),
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
		})

		It("should fetch balance successfully", func(ctx SpecContext) {
			from := models.PSPAccount{Reference: "BTC"}
			fromPayload, _ := json.Marshal(from)

			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
				[]client.AccountBalance{
					{Currency: "btc", Total: "1.50000000", Available: "1.00000000", Reserved: "0.50000000"},
					{Currency: "usd", Total: "5000.00", Available: "4500.00", Reserved: "500.00"},
				},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.HasMore).To(BeFalse())
			Expect(resp.Balances[0].AccountReference).To(Equal("BTC"))
			Expect(resp.Balances[0].Asset).To(Equal("BTC/8"))
			// 1.00000000 BTC = 100000000 satoshis
			Expect(resp.Balances[0].Amount.Int64()).To(Equal(int64(100000000)))
		})

		It("should return empty for unknown currency", func(ctx SpecContext) {
			from := models.PSPAccount{Reference: "UNKNOWN"}
			fromPayload, _ := json.Marshal(from)

			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
				[]client.AccountBalance{
					{Currency: "btc", Total: "1.00000000", Available: "1.00000000", Reserved: "0"},
				},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(BeNil())
			Expect(resp.HasMore).To(BeFalse())
		})

		It("should handle lowercase to uppercase currency matching", func(ctx SpecContext) {
			from := models.PSPAccount{Reference: "ETH"}
			fromPayload, _ := json.Marshal(from)

			req := models.FetchNextBalancesRequest{
				FromPayload: fromPayload,
			}

			m.EXPECT().GetAccountBalances(gomock.Any()).Return(
				[]client.AccountBalance{
					{Currency: "eth", Total: "10.5", Available: "10.5", Reserved: "0"},
				},
				nil,
			)

			resp, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.Balances).To(HaveLen(1))
			Expect(resp.Balances[0].Asset).To(Equal("ETH/18"))
		})
	})
})
