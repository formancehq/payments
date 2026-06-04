package krakenpro

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

// mkSpotAccount builds a spot (trading) PSPAccount as fetch_accounts
// emits it: raw-code reference + wallet_type=spot metadata. Shared by
// the payments + conversions specs whose orchestrators resolveWallets.
func mkSpotAccount(rawCode string) models.PSPAccount {
	return models.PSPAccount{
		Reference: rawCode,
		Metadata:  map[string]string{"com.krakenpro.spec/wallet_type": "spot"},
	}
}

var _ = Describe("Kraken Pro fetch_payments", func() {
	var (
		ctrl   *gomock.Controller
		m      *client.MockClient
		lookup *models.MockAccountLookup
		plg    *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		lookup = models.NewMockAccountLookup(ctrl)
		lookup.EXPECT().ListAccountsByConnector(gomock.Any()).Return([]models.PSPAccount{
			mkSpotAccount("XXBT"), mkSpotAccount("ZUSD"), mkSpotAccount("ZEUR"),
		}, nil).AnyTimes()
		plg = &Plugin{
			Plugin:        plugins.NewBasePlugin(),
			client:        m,
			logger:        logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			accountLookup: lookup,
			currencies: map[string]int{
				"BTC": 8, "USD": 2, "EUR": 2,
			},
			assetsLoaded: time.Now(),
		}
	})

	AfterEach(func() { ctrl.Finish() })

	It("emits one PSPPayment per payment-classified ledger row, skipping trades and conversions", func(ctx SpecContext) {
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{
			Ledger: map[string]client.LedgerEntry{
				"L1": {Refid: "R1", Time: 1.0, Type: "deposit", Asset: "ZEUR", Amount: "100.00"},
				"L2": {Refid: "R2", Time: 2.0, Type: "trade", Asset: "XXBT", Amount: "-0.01"},
				"L3": {Refid: "R3", Time: 3.0, Type: "conversion", Asset: "ZUSD", Amount: "-50.00"},
				"L4": {Refid: "R4", Time: 4.0, Type: "withdrawal", Asset: "XXBT", Amount: "-0.5"},
			},
		}, nil)

		resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Payments).To(HaveLen(2)) // L1 and L4
		refs := []string{resp.Payments[0].Reference, resp.Payments[1].Reference}
		Expect(refs).To(ConsistOf("L1", "L4"))
	})

	It("first cycle freezes a window end and pages ofs from 0", func(ctx SpecContext) {
		m.EXPECT().GetLedgers(gomock.Any(), gomock.AssignableToTypeOf(client.LedgersParams{})).
			DoAndReturn(func(_ any, p client.LedgersParams) (client.LedgersResponse, error) {
				Expect(p.Start).To(BeZero(), "fresh install has no watermark")
				Expect(p.End).ToNot(BeZero(), "the window end is frozen at cycle start")
				Expect(p.Offset).To(BeZero(), "first page starts at ofs=0")
				return client.LedgersResponse{Ledger: fullDepositPage()}, nil
			})
		resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{})
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeTrue(), "a full page means more of the frozen window remains")

		var state paymentsState
		Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
		Expect(state.Window.draining()).To(BeTrue())
		Expect(state.Window.Offset).To(Equal(PAGE_SIZE))
		Expect(state.Window.Watermark).To(BeZero(), "watermark only promotes once the window drains")
	})

	It("short page promotes the watermark to the frozen end and ends the drain", func(ctx SpecContext) {
		startState := paymentsState{Window: ledgerWindow{Watermark: 500, End: 1800, Offset: 50}}
		stateBytes, _ := json.Marshal(startState)

		m.EXPECT().GetLedgers(gomock.Any(), gomock.AssignableToTypeOf(client.LedgersParams{})).
			DoAndReturn(func(_ any, p client.LedgersParams) (client.LedgersResponse, error) {
				Expect(p.Start).To(BeNumerically("==", 500), "exclusive lower bound = committed watermark")
				Expect(p.End).To(BeNumerically("==", 1800), "resumes the frozen window end")
				Expect(p.Offset).To(Equal(50))
				return client.LedgersResponse{Ledger: map[string]client.LedgerEntry{
					"L1": {Time: 1700.0, Type: "deposit", Asset: "ZUSD", Amount: "1.00"},
				}}, nil
			})
		resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: stateBytes})
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeFalse())

		var state paymentsState
		Expect(json.Unmarshal(resp.NewState, &state)).To(Succeed())
		Expect(state.Window.draining()).To(BeFalse(), "window resets after drain")
		Expect(state.Window.Watermark).To(BeNumerically("==", 1800), "watermark promoted to frozen end")
	})

	It("drains a window larger than PAGE_SIZE across cycles with no skips", func(ctx SpecContext) {
		drainAndAssertAllEmitted(ctx, plg, m, distinctTimeRows(120))
	})

	It("does not skip rows that share an identical timestamp (ofs is positional)", func(ctx SpecContext) {
		// 120 rows all at the same time — a timestamp/ID cursor would
		// loop or skip here; ofs indexes position, so all drain.
		drainAndAssertAllEmitted(ctx, plg, m, sameTimeRows(120, 100.0))
	})

	It("propagates GetLedgers errors", func(ctx SpecContext) {
		m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).Return(client.LedgersResponse{}, errors.New("nope"))
		_, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{})
		Expect(err).To(HaveOccurred())
	})
})

// ledgerRow pairs a ledger id with its entry for ordered ofs slicing.
type ledgerRow struct {
	id string
	e  client.LedgerEntry
}

func fullDepositPage() map[string]client.LedgerEntry {
	out := map[string]client.LedgerEntry{}
	for i := 0; i < PAGE_SIZE; i++ {
		out[fmt.Sprintf("L%03d", i)] = client.LedgerEntry{
			Time: float64(1000 + i), Type: "deposit", Asset: "ZUSD", Amount: "1.00",
		}
	}
	return out
}

func distinctTimeRows(n int) []ledgerRow {
	rows := make([]ledgerRow, n)
	for i := 0; i < n; i++ {
		rows[i] = ledgerRow{
			id: fmt.Sprintf("L%04d", i),
			e:  client.LedgerEntry{Time: float64(1000 + i), Type: "deposit", Asset: "ZUSD", Amount: "1.00"},
		}
	}
	return rows
}

func sameTimeRows(n int, ts float64) []ledgerRow {
	rows := make([]ledgerRow, n)
	for i := 0; i < n; i++ {
		rows[i] = ledgerRow{
			id: fmt.Sprintf("L%04d", i),
			e:  client.LedgerEntry{Time: ts, Type: "deposit", Asset: "ZUSD", Amount: "1.00"},
		}
	}
	return rows
}

// drainAndAssertAllEmitted serves `rows` as ofs-paged windows and drives
// FetchNextPayments across cycles, asserting every row is emitted exactly
// once and the frozen window end never shifts during the drain.
func drainAndAssertAllEmitted(ctx SpecContext, plg *Plugin, m *client.MockClient, rows []ledgerRow) {
	var frozenEnd float64
	m.EXPECT().GetLedgers(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, p client.LedgersParams) (client.LedgersResponse, error) {
			Expect(p.End).ToNot(BeZero())
			if frozenEnd == 0 {
				frozenEnd = p.End
			} else {
				Expect(p.End).To(BeNumerically("==", frozenEnd), "End must stay frozen across the drain")
			}
			ledger := map[string]client.LedgerEntry{}
			for i := p.Offset; i < p.Offset+PAGE_SIZE && i < len(rows); i++ {
				ledger[rows[i].id] = rows[i].e
			}
			return client.LedgersResponse{Ledger: ledger}, nil
		}).AnyTimes()

	emitted := map[string]int{}
	var stateBytes json.RawMessage
	for cycle := 0; cycle < 20; cycle++ {
		resp, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: stateBytes})
		Expect(err).To(BeNil())
		for _, pmt := range resp.Payments {
			emitted[pmt.Reference]++
		}
		stateBytes = resp.NewState
		if !resp.HasMore {
			break
		}
	}
	Expect(emitted).To(HaveLen(len(rows)), "every row drained, none skipped")
	for ref, count := range emitted {
		Expect(count).To(Equal(1), "row %s emitted exactly once", ref)
	}
}
