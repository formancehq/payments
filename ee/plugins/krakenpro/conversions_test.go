package krakenpro

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/plugins"
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

	It("refreshes + retries when a leg's asset is unknown, then emits and clears pending", func(ctx SpecContext) {
		// Cache lacks DOGE; the arriving leg references XXDG.
		plg.currencies = map[string]int{"BTC": 8}
		plg.assetCodes = map[string]string{"BTC": "XXBT"}
		plg.assetsLoaded = time.Now()

		startState := conversionsState{Pending: map[string]pendingLeg{
			"C1": {LedgerID: "LA", Time: 1.0, Type: "conversion", Asset: "XXBT", Amount: "-0.01"},
		}}
		stateBytes, _ := json.Marshal(startState)

		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{
			Ledger: map[string]client.LedgerEntry{
				"LB": {Refid: "C1", Type: "conversion", Asset: "XXDG", Amount: "300.0", Time: 2.0},
			},
		}, nil)
		// Forced refresh sees both assets so the pair maps on retry.
		m.EXPECT().GetAssets(gomock.Any()).Return(map[string]client.AssetInfo{
			"XXBT": {Altname: "XBT", Decimals: 8},
			"XXDG": {Altname: "XDG", Decimals: 8},
		}, nil)
		m.EXPECT().GetAssetPairs(gomock.Any()).Return(map[string]client.AssetPair{}, nil)

		resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{State: stateBytes})
		Expect(err).To(BeNil())
		Expect(resp.Conversions).To(HaveLen(1), "leg recovered after refresh, not lost")
		var st conversionsState
		Expect(json.Unmarshal(resp.NewState, &st)).To(Succeed())
		Expect(st.Pending).To(BeEmpty(), "pending cleared only after a successful emit")
	})

	It("propagates GetLedgers errors", func(ctx SpecContext) {
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{}, errors.New("boom"))
		_, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{})
		Expect(err).To(HaveOccurred())
	})
})
