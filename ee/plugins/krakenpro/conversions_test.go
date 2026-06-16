package krakenpro

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Kraken Pro fetch_conversions", func() {
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
				"BTC": 8, "USD": 2,
			},
			// Wallet refs resolve from this cache (symbol -> raw spot code).
			assetCodes:   map[string]string{"BTC": "XXBT", "USD": "ZUSD"},
			assetsLoaded: time.Now(),
		}
	})

	AfterEach(func() { ctrl.Finish() })

	It("pairs two legs sharing one refid into one PSPConversion", func(ctx SpecContext) {
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{
			Ledger: map[string]client.LedgerEntry{
				"L1": {Refid: "C1", Type: "conversion", Asset: "ZUSD", Amount: "-100.00", Time: 1.0},
				"L2": {Refid: "C1", Type: "conversion", Asset: "XXBT", Amount: "0.0036", Time: 2.0},
				"L3": {Refid: "OTHER", Type: "deposit", Asset: "ZUSD", Amount: "5.00", Time: 3.0},
			},
		}, nil)

		resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Conversions).To(HaveLen(1))
		Expect(resp.Conversions[0].Reference).To(Equal("C1"))
	})

	It("buffers a half-pair across cycles via Pending", func(ctx SpecContext) {
		// Cycle 1: only the source leg arrives.
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{
			Ledger: map[string]client.LedgerEntry{
				"L1": {Refid: "C1", Type: "conversion", Asset: "ZUSD", Amount: "-100.00", Time: 1.0},
			},
		}, nil)
		resp1, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{})
		Expect(err).To(BeNil())
		Expect(resp1.Conversions).To(BeEmpty())

		var state conversionsState
		Expect(json.Unmarshal(resp1.NewState, &state)).To(Succeed())
		Expect(state.Pending).To(HaveKey("C1"))

		// Cycle 2: the destination leg arrives and the pair completes.
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{
			Ledger: map[string]client.LedgerEntry{
				"L2": {Refid: "C1", Type: "conversion", Asset: "XXBT", Amount: "0.0036", Time: 2.0},
			},
		}, nil)
		resp2, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{State: resp1.NewState})
		Expect(err).To(BeNil())
		Expect(resp2.Conversions).To(HaveLen(1))
	})

	It("propagates GetLedgers errors", func(ctx SpecContext) {
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{}, errors.New("boom"))
		_, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{})
		Expect(err).To(HaveOccurred())
	})
})
