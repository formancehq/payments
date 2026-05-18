package universal_test

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Universal *Plugin — fetch primitives", func() {
	var (
		ctrl   *gomock.Controller
		mc     *client.MockClient
		plg    *universal.Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		cfg    = json.RawMessage(`{"endpoint":"https://x","apiKey":"k"}`)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mc = client.NewMockClient(ctrl)
		mc.EXPECT().SetIdempotencyHeader(gomock.Any()).AnyTimes()
		var err error
		plg, err = universal.New(models.ConnectorID{}, "universal-test", logger, cfg)
		Expect(err).To(BeNil())
		universal.InjectClient(plg, mc)
	})

	AfterEach(func() { ctrl.Finish() })

	Context("FetchNextAccounts", func() {
		It("translates wire to PSPAccount and roundtrips state", func(ctx SpecContext) {
			universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_ACCOUNTS})
			now := time.Now().UTC().Truncate(time.Second)
			mc.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Return(&client.AccountsPage{
				Items: []client.Account{
					{Reference: "a1", CreatedAt: now, Name: pStr("Op EUR")},
				},
				NextCursor: "c2",
				HasMore:    true,
			}, nil)
			res, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{PageSize: 10})
			Expect(err).To(BeNil())
			Expect(res.Accounts).To(HaveLen(1))
			Expect(res.Accounts[0].Reference).To(Equal("a1"))
			Expect(res.HasMore).To(BeTrue())
			var st struct {
				NextCursor    string    `json:"nextCursor"`
				LastUpdatedAt time.Time `json:"lastUpdatedAt"`
			}
			Expect(json.Unmarshal(res.NewState, &st)).To(BeNil())
			Expect(st.NextCursor).To(Equal("c2"))
			Expect(st.LastUpdatedAt).To(Equal(now), "LastUpdatedAt must advance to CreatedAt for incremental polling")
		})
	})

	Context("FetchNextPayments", func() {
		It("threads updatedAt through state", func(ctx SpecContext) {
			universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_PAYMENTS})
			t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			t2 := time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC)
			mc.EXPECT().ListPayments(gomock.Any(), gomock.Any()).Return(&client.PaymentsPage{
				Items: []client.Payment{
					{Reference: "p1", CreatedAt: t1, UpdatedAt: t1, Type: "PAYIN", Status: "SUCCEEDED", Amount: "100", Asset: "EUR/2"},
					{Reference: "p2", CreatedAt: t2, UpdatedAt: t2, Type: "PAYIN", Status: "PENDING", Amount: "200", Asset: "EUR/2"},
				},
				HasMore: false,
			}, nil)
			res, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 100})
			Expect(err).To(BeNil())
			Expect(res.Payments).To(HaveLen(2))
			Expect(res.Payments[0].Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			var st struct {
				LastUpdatedAt time.Time `json:"lastUpdatedAt"`
			}
			Expect(json.Unmarshal(res.NewState, &st)).To(BeNil())
			Expect(st.LastUpdatedAt).To(Equal(t2))
		})
	})

	Context("FetchNextOrders", func() {
		It("maps order direction + status and advances LastUpdatedAt", func(ctx SpecContext) {
			universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_ORDERS})
			upd := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
			mc.EXPECT().ListOrders(gomock.Any(), gomock.Any()).Return(&client.OrdersPage{
				Items: []client.Order{
					{
						Reference: "o1", CreatedAt: time.Now().UTC(), UpdatedAt: upd,
						Direction: "BUY", Type: "MARKET", Status: "FILLED",
						SourceAsset: "EUR/2", DestinationAsset: "BTC/8",
						BaseQuantityOrdered: "1000",
					},
				},
				HasMore: false,
			}, nil)
			res, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{PageSize: 10})
			Expect(err).To(BeNil())
			Expect(res.Orders).To(HaveLen(1))
			Expect(res.Orders[0].Direction).To(Equal(models.ORDER_DIRECTION_BUY))
			Expect(res.Orders[0].Status).To(Equal(models.ORDER_STATUS_FILLED))
			var st struct {
				LastUpdatedAt time.Time `json:"lastUpdatedAt"`
			}
			Expect(json.Unmarshal(res.NewState, &st)).To(BeNil())
			Expect(st.LastUpdatedAt).To(Equal(upd))
		})
	})

	Context("FetchNextConversions", func() {
		It("translates conversion fields and advances LastUpdatedAt", func(ctx SpecContext) {
			universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_CONVERSIONS})
			created := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
			mc.EXPECT().ListConversions(gomock.Any(), gomock.Any()).Return(&client.ConversionsPage{
				Items: []client.Conversion{
					{Reference: "c1", CreatedAt: created, Status: "COMPLETED",
						SourceAsset: "USDC/6", DestinationAsset: "USD/2", SourceAmount: "1000000"},
				},
			}, nil)
			res, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{PageSize: 10})
			Expect(err).To(BeNil())
			Expect(res.Conversions).To(HaveLen(1))
			Expect(res.Conversions[0].Status).To(Equal(models.CONVERSION_STATUS_COMPLETED))
			var st struct {
				LastUpdatedAt time.Time `json:"lastUpdatedAt"`
			}
			Expect(json.Unmarshal(res.NewState, &st)).To(BeNil())
			Expect(st.LastUpdatedAt).To(Equal(created))
		})
	})

	Context("FetchNextOthers", func() {
		It("rejects empty Name with ErrInvalidRequest", func(ctx SpecContext) {
			universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_OTHERS})
			_, err := plg.FetchNextOthers(ctx, models.FetchNextOthersRequest{Name: "  "})
			Expect(err).NotTo(BeNil())
			Expect(errors.Is(err, models.ErrInvalidRequest)).To(BeTrue())
		})
	})

	Context("fetchPaginated — atomic state advancement", func() {
		It("does not advance LastUpdatedAt when conversion fails mid-batch", func(ctx SpecContext) {
			universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_FETCH_PAYMENTS})
			t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			t2 := time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC)
			mc.EXPECT().ListPayments(gomock.Any(), gomock.Any()).Return(&client.PaymentsPage{
				Items: []client.Payment{
					{Reference: "p1", CreatedAt: t1, UpdatedAt: t1, Type: "PAYIN", Status: "SUCCEEDED", Amount: "100", Asset: "EUR/2"},
					// Second row is unmappable (bad amount) — the
					// whole batch MUST fail and the returned
					// (nil) state forces the engine to retry from
					// the previous watermark.
					{Reference: "p2", CreatedAt: t2, UpdatedAt: t2, Type: "PAYIN", Status: "PENDING", Amount: "not-a-number", Asset: "EUR/2"},
				},
			}, nil)
			res, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{PageSize: 100})
			Expect(err).NotTo(BeNil())
			Expect(res.NewState).To(BeNil(), "partial-failure must NOT surface a state — engine should retry with prior watermark")
		})
	})
})

func pStr(s string) *string { return &s }
