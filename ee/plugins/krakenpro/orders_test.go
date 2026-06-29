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

var _ = Describe("Kraken Pro fetch_orders", func() {
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
			assetPairs: map[string]client.AssetPair{
				"XXBTZUSD": {Altname: "XBTUSD", Wsname: "XBT/USD", Base: "XXBT", Quote: "ZUSD"},
			},
			assetsLoaded: time.Now(),
		}
	})

	AfterEach(func() { ctrl.Finish() })

	closedFilledOrder := func() client.OrderEntry {
		return client.OrderEntry{
			Status:  "closed",
			Opentm:  100,
			Closetm: 200,
			Descr:   client.OrderDescr{Pair: "XXBTZUSD", Type: "buy", Ordertype: "limit", Price: "27500"},
			Vol:     "1.00000000", VolExec: "1.00000000",
			Cost:   "27500.00",
			Fee:    "73.70",
			Price:  "27500",
			Trades: []string{"T1", "T2"},
		}
	}

	Describe("window pagination", func() {
		It("drains ClosedOrders via the frozen window and resolves refs from pair codes", func(ctx SpecContext) {
			// First window page — ofs=0, no Start (fresh), frozen End,
			// closetime="close". Short page → window drains.
			m.EXPECT().GetClosedOrders(gomock.Any(), gomock.AssignableToTypeOf(client.ClosedOrdersParams{})).
				DoAndReturn(func(_ any, p client.ClosedOrdersParams) (client.ClosedOrdersResponse, error) {
					Expect(p.Trades).To(BeTrue())
					Expect(p.Offset).To(BeZero())
					Expect(p.Start).To(BeZero())
					Expect(p.End).ToNot(BeZero(), "window end is frozen")
					Expect(p.Closetime).To(Equal(client.ClosetimeClose))
					return client.ClosedOrdersResponse{
						Closed: map[string]client.OrderEntry{"OCLOSED1": closedFilledOrder()},
					}, nil
				})

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(err).To(BeNil())
			Expect(resp.HasMore).To(BeFalse(), "short closed page drains the window")
			Expect(resp.Orders).To(HaveLen(1))
			// BUY: source = quote (ZUSD), destination = base (XXBT).
			Expect(*resp.Orders[0].SourceAccountReference).To(Equal("ZUSD"))
			Expect(*resp.Orders[0].DestinationAccountReference).To(Equal("XXBT"))

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.Closed.draining()).To(BeFalse(), "window promoted after short page")
			Expect(state.Closed.Watermark).ToNot(BeZero(), "watermark promoted to frozen end")
		})

		It("resumes a frozen ClosedOrders window with start=watermark", func(ctx SpecContext) {
			startState := ordersState{Closed: ledgerWindow{Watermark: 1234.5, End: 4000, Offset: 50}}
			stateBytes, _ := json.Marshal(startState)
			m.EXPECT().GetClosedOrders(gomock.Any(), gomock.AssignableToTypeOf(client.ClosedOrdersParams{})).
				DoAndReturn(func(_ any, p client.ClosedOrdersParams) (client.ClosedOrdersResponse, error) {
					Expect(p.Start).To(BeNumerically("==", 1234.5))
					Expect(p.End).To(BeNumerically("==", 4000))
					Expect(p.Offset).To(Equal(50))
					return client.ClosedOrdersResponse{Closed: map[string]client.OrderEntry{}}, nil
				})

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: stateBytes})
			Expect(err).To(BeNil())
			Expect(resp.HasMore).To(BeFalse())

			var state ordersState
			Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
			Expect(state.Closed.Watermark).To(BeNumerically("==", 4000), "promoted to frozen end")
			Expect(state.Closed.draining()).To(BeFalse())
		})

		It("drains a ClosedOrders window larger than PAGE_SIZE with no skips", func(ctx SpecContext) {
			const n = 117
			ids := make([]string, n)
			for i := range ids {
				ids[i] = fmt.Sprintf("OC%04d", i)
			}
			var frozenEnd float64
			m.EXPECT().GetClosedOrders(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ any, p client.ClosedOrdersParams) (client.ClosedOrdersResponse, error) {
					Expect(p.End).ToNot(BeZero())
					if frozenEnd == 0 {
						frozenEnd = p.End
					} else {
						Expect(p.End).To(BeNumerically("==", frozenEnd), "End frozen across drain")
					}
					closed := map[string]client.OrderEntry{}
					for i := p.Offset; i < p.Offset+PAGE_SIZE && i < n; i++ {
						closed[ids[i]] = closedFilledOrder()
					}
					return client.ClosedOrdersResponse{Closed: closed}, nil
				}).AnyTimes()

			emitted := map[string]int{}
			var stateBytes json.RawMessage
			for cycle := 0; cycle < 20; cycle++ {
				resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: stateBytes})
				Expect(err).To(BeNil())
				for _, o := range resp.Orders {
					emitted[o.Reference]++
				}
				stateBytes = resp.NewState
				if !resp.HasMore {
					break
				}
			}
			Expect(emitted).To(HaveLen(n), "every closed order drained, none skipped")
		})
	})

	Describe("error handling", func() {
		It("skips an order whose asset precision is unknown without failing the page", func(ctx SpecContext) {
			plg.currencies = map[string]int{"BTC": 8} // USD (quote) precision absent
			m.EXPECT().GetClosedOrders(gomock.Any(), gomock.Any()).
				Return(client.ClosedOrdersResponse{
					Closed: map[string]client.OrderEntry{"OHIST": closedFilledOrder()},
				}, nil)

			resp, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(err).To(BeNil(), "an unmappable row must not fail the page")
			Expect(resp.Orders).To(BeEmpty())
		})

		It("propagates GetClosedOrders errors", func(ctx SpecContext) {
			m.EXPECT().GetClosedOrders(gomock.Any(), gomock.Any()).
				Return(client.ClosedOrdersResponse{}, errors.New("kaboom"))
			_, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(err).To(HaveOccurred())
		})
	})
})
