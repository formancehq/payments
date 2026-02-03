package fireblocks

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Fireblocks Plugin Balances", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			logger:         logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			client:         m,
			assetDecimals:  map[string]int{"USD": 2},
			assetsLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("returns error when from payload is missing", func(ctx SpecContext) {
		resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("from payload is required"))
		Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
	})

	It("fetches balances and skips invalid entries", func(ctx SpecContext) {
		from, err := json.Marshal(models.PSPAccount{Reference: "acc-1"})
		Expect(err).To(BeNil())

		m.EXPECT().GetVaultAccount(gomock.Any(), "acc-1").Return(&client.VaultAccount{
			ID: "acc-1",
			Assets: []client.VaultAsset{
				{ID: "USD", Available: "10.50"},
				{ID: "UNKNOWN", Available: "5"},
				{ID: "USD", Available: "bad"},
			},
		}, nil)

		resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{
			FromPayload: from,
		})
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeFalse())
		Expect(resp.Balances).To(HaveLen(1))
		Expect(resp.Balances[0].AccountReference).To(Equal("acc-1"))
		Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(1050)))
		Expect(resp.Balances[0].Asset).To(Equal("USD/2"))
		Expect(resp.Balances[0].CreatedAt.IsZero()).To(BeFalse())
	})
})
