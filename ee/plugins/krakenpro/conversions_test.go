package krakenpro

import (
	"encoding/json"
	"errors"
	"fmt"
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
		// Account refs are each leg's own raw Kraken code.
		Expect(*resp.Conversions[0].SourceAccountReference).To(Equal("ZUSD"))
		Expect(*resp.Conversions[0].DestinationAccountReference).To(Equal("XXBT"))
	})

	It("buffers a half-pair across cycles of a still-draining window", func(ctx SpecContext) {
		// Cycle 1: a full page (so the window keeps draining, watermark
		// unchanged) carrying only the source leg → buffered in Pending.
		full := map[string]client.LedgerEntry{
			"L1": {Refid: "C1", Type: "conversion", Asset: "ZUSD", Amount: "-100.00", Time: 1.0},
		}
		for i := 0; i < PAGE_SIZE-1; i++ {
			full[fmt.Sprintf("T%02d", i)] = client.LedgerEntry{Type: "trade", Asset: "XXBT", Amount: "0.1", Time: 1.0}
		}
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{Ledger: full}, nil)

		resp1, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{})
		Expect(err).To(BeNil())
		Expect(resp1.Conversions).To(BeEmpty())
		Expect(resp1.HasMore).To(BeTrue(), "full page keeps the window draining")

		var state conversionsState
		Expect(json.Unmarshal(resp1.NewState, &state)).To(Succeed())
		Expect(state.Pending).To(HaveKey("C1"))

		// Cycle 2: a short page with the destination leg completes the pair.
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{
			Ledger: map[string]client.LedgerEntry{
				"L2": {Refid: "C1", Type: "conversion", Asset: "XXBT", Amount: "0.0036", Time: 2.0},
			},
		}, nil)
		resp2, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{State: resp1.NewState})
		Expect(err).To(BeNil())
		Expect(resp2.Conversions).To(HaveLen(1))
	})

	It("refreshes the cache when a leg's asset is unknown, then emits", func(ctx SpecContext) {
		// Cache lacks DOGE; the arriving leg references XXDG. A pending
		// known-asset source leg is already buffered.
		plg.currencies = map[string]int{"BTC": 8}
		plg.assetsLoaded = time.Now()

		startState := conversionsState{Pending: map[string]client.LedgerEntry{
			"C1": {ID: "LA", Refid: "C1", Time: 1.0, Type: "conversion", Asset: "XXBT", Amount: "-0.01"},
		}}
		stateBytes, _ := json.Marshal(startState)

		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{
			Ledger: map[string]client.LedgerEntry{
				"LB": {Refid: "C1", Type: "conversion", Asset: "XXDG", Amount: "300.0", Time: 2.0},
			},
		}, nil)
		// Forced refresh sees both assets so the pair maps after refresh.
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
		Expect(st.Pending).To(BeEmpty(), "pending cleared after the pair emits")
	})

	It("drops a row whose asset stays unknown after refresh, never buffering it", func(ctx SpecContext) {
		plg.currencies = map[string]int{"BTC": 8}
		plg.assetsLoaded = time.Now()

		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{
			Ledger: map[string]client.LedgerEntry{
				"LB": {Refid: "C9", Type: "conversion", Asset: "NOPE", Amount: "1.0", Time: 2.0},
			},
		}, nil)
		// Refresh still doesn't know NOPE.
		m.EXPECT().GetAssets(gomock.Any()).Return(map[string]client.AssetInfo{
			"XXBT": {Altname: "XBT", Decimals: 8},
		}, nil)
		m.EXPECT().GetAssetPairs(gomock.Any()).Return(map[string]client.AssetPair{}, nil)

		resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Conversions).To(BeEmpty())
		var st conversionsState
		Expect(json.Unmarshal(resp.NewState, &st)).To(Succeed())
		Expect(st.Pending).To(BeEmpty(), "unknown-asset rows must never enter pending")
	})

	It("prunes a stale half-pair once its window fully drains", func(ctx SpecContext) {
		// A pending leg whose time is below the window end: when the short
		// page promotes the watermark past it, the orphan is pruned.
		startState := conversionsState{
			Window:  ledgerWindow{End: 5000, Offset: 0},
			Pending: map[string]client.LedgerEntry{"OLD": {ID: "LO", Refid: "OLD", Time: 1.0, Type: "conversion", Asset: "XXBT", Amount: "-1"}},
		}
		stateBytes, _ := json.Marshal(startState)
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).
			Return(client.LedgersResponse{Ledger: map[string]client.LedgerEntry{}}, nil)

		resp, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{State: stateBytes})
		Expect(err).To(BeNil())
		var st conversionsState
		Expect(json.Unmarshal(resp.NewState, &st)).To(Succeed())
		Expect(st.Window.Watermark).To(BeNumerically("==", 5000), "watermark promoted to frozen end")
		Expect(st.Pending).To(BeEmpty(), "orphan pruned once watermark passes its time")
	})

	It("propagates GetLedgers errors", func(ctx SpecContext) {
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{}, errors.New("boom"))
		_, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{})
		Expect(err).To(HaveOccurred())
	})
})
