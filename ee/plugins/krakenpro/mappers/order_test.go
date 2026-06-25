package mappers

import (
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

var testPairs = map[string]client.AssetPair{
	"XXBTZUSD": {Altname: "XBTUSD", Wsname: "XBT/USD", Base: "XXBT", Quote: "ZUSD"},
}

var testWallets = map[string]string{
	"BTC": "BTC",
	"USD": "USD",
	"EUR": "EUR",
}

// ---------- Status matrix ----------

func TestMapKrakenOrderStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status string
		vol    string
		exec   string
		want   models.OrderStatus
		known  bool
	}{
		{"pending", "pending", "1.0", "0", models.ORDER_STATUS_PENDING, true},
		{"open no fill", "open", "1.0", "0", models.ORDER_STATUS_OPEN, true},
		{"open partial fill", "open", "1.0", "0.3", models.ORDER_STATUS_PARTIALLY_FILLED, true},
		{"closed fully filled", "closed", "1.0", "1.0", models.ORDER_STATUS_FILLED, true},
		{"closed partial fill", "closed", "1.0", "0.3", models.ORDER_STATUS_PARTIALLY_FILLED, true},
		{"closed without fill", "closed", "1.0", "0", models.ORDER_STATUS_CANCELLED, true},
		{"canceled with partial fill", "canceled", "1.0", "0.5", models.ORDER_STATUS_PARTIALLY_FILLED, true},
		{"canceled without fill", "canceled", "1.0", "0", models.ORDER_STATUS_CANCELLED, true},
		{"cancelled with two-L spelling", "cancelled", "1.0", "0", models.ORDER_STATUS_CANCELLED, true},
		{"expired", "expired", "1.0", "0", models.ORDER_STATUS_EXPIRED, true},
		{"unknown future value", "deferred", "1.0", "0", models.ORDER_STATUS_PENDING, false},
		{"textual vol/exec mismatch is normalised", "closed", "1.00000000", "1", models.ORDER_STATUS_FILLED, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, known := MapOrderStatus(c.status, c.vol, c.exec)
			if got != c.want || known != c.known {
				t.Errorf("MapOrderStatus(%q,%q,%q) = (%v,%v), want (%v,%v)",
					c.status, c.vol, c.exec, got, known, c.want, c.known)
			}
		})
	}
}

// ---------- OrderEntryToPSPOrder ----------

func filledOrder(side, ordertype string) client.OrderEntry {
	return client.OrderEntry{
		Status:  "closed",
		Opentm:  1000,
		Closetm: 2000,
		Descr: client.OrderDescr{
			Pair: "XXBTZUSD", Type: side, Ordertype: ordertype,
			Price: "27500.0",
		},
		Vol:     "1.00000000",
		VolExec: "1.00000000",
		Cost:    "27500.00",
		Fee:     "73.70",
		Price:   "27500.0",
		Trades:  []string{"T1", "T2"},
	}
}

func TestOrderEntryToPSPOrder_FilledBuy(t *testing.T) {
	t.Parallel()
	got, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "O1", Order: filledOrder("buy", "limit")})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Direction != models.ORDER_DIRECTION_BUY {
		t.Errorf("direction=%v", got.Direction)
	}
	if got.Type != models.ORDER_TYPE_LIMIT {
		t.Errorf("type=%v", got.Type)
	}
	if got.Status != models.ORDER_STATUS_FILLED {
		t.Errorf("status=%v", got.Status)
	}
	// BUY → source = quote (USD), destination = base (BTC)
	if *got.SourceAccountReference != "USD" || *got.DestinationAccountReference != "BTC" {
		t.Errorf("BUY wallets: src=%v dst=%v", *got.SourceAccountReference, *got.DestinationAccountReference)
	}
	if got.BaseQuantityOrdered.Cmp(big.NewInt(100_000_000)) != 0 {
		t.Errorf("ordered=%s", got.BaseQuantityOrdered)
	}
	if got.BaseQuantityFilled.Cmp(big.NewInt(100_000_000)) != 0 {
		t.Errorf("filled=%s", got.BaseQuantityFilled)
	}
	// testCurrencies has USD precision 2, so 27500.00 -> 2_750_000
	if got.QuoteAmount.Cmp(big.NewInt(2_750_000)) != 0 {
		t.Errorf("quote=%s want 2750000", got.QuoteAmount)
	}
	if got.Fee.Cmp(big.NewInt(7370)) != 0 {
		t.Errorf("fee=%s want 7370", got.Fee)
	}
	if got.Metadata[MetadataPrefix+"fills"] != "T1,T2" {
		t.Errorf("fills metadata missing or wrong: %v", got.Metadata)
	}
}

// CreatedAt must stay anchored to opentm even on a closed order so the
// open->closed upsert doesn't mutate the creation timestamp; close time
// is preserved in metadata instead.
func TestOrderEntryToPSPOrder_CreatedAtFromOpentmNotClosetm(t *testing.T) {
	t.Parallel()
	got, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "O1", Order: filledOrder("buy", "limit")})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if want := FloatEpochToTime(1000); !got.CreatedAt.Equal(want) {
		t.Errorf("CreatedAt=%v want %v (opentm, not closetm)", got.CreatedAt, want)
	}
	if got := got.Metadata[MetadataPrefix+"close_time"]; got != FloatEpochToTime(2000).Format(time.RFC3339) {
		t.Errorf("close_time metadata=%q", got)
	}
}

// An order still open (closetm == 0) must not emit a close_time entry.
func TestOrderEntryToPSPOrder_NoCloseTimeWhenOpen(t *testing.T) {
	t.Parallel()
	oe := filledOrder("buy", "limit")
	oe.Status, oe.Closetm = "open", 0
	got, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "O1", Order: oe})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if _, ok := got.Metadata[MetadataPrefix+"close_time"]; ok {
		t.Errorf("close_time should be absent on an open order: %v", got.Metadata)
	}
}

func TestOrderEntryToPSPOrder_FilledSellInvertsWallets(t *testing.T) {
	t.Parallel()
	got, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "O2", Order: filledOrder("sell", "market")})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Direction != models.ORDER_DIRECTION_SELL {
		t.Errorf("direction=%v", got.Direction)
	}
	// SELL → source = base (BTC), destination = quote (USD)
	if *got.SourceAccountReference != "BTC" || *got.DestinationAccountReference != "USD" {
		t.Errorf("SELL wallets: src=%v dst=%v", *got.SourceAccountReference, *got.DestinationAccountReference)
	}
}

func TestOrderEntryToPSPOrder_OpenStatusFromVolExec(t *testing.T) {
	t.Parallel()
	oe := filledOrder("buy", "limit")
	oe.Status = "open"
	oe.VolExec = "0.30000000" // partial
	got, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "O3", Order: oe})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Status != models.ORDER_STATUS_PARTIALLY_FILLED {
		t.Errorf("expected PARTIALLY_FILLED, got %v", got.Status)
	}
	if got.BaseQuantityFilled.Cmp(big.NewInt(30_000_000)) != 0 {
		t.Errorf("filled=%s", got.BaseQuantityFilled)
	}
}

func TestOrderEntryToPSPOrder_UnknownPair(t *testing.T) {
	t.Parallel()
	oe := filledOrder("buy", "limit")
	oe.Descr.Pair = "BOGUS"
	_, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "OZ", Order: oe})
	if err == nil {
		t.Fatal("expected unknown-pair error")
	}
}

func TestOrderEntryToPSPOrder_UnresolvedWalletYieldsNilRef(t *testing.T) {
	t.Parallel()
	// Drop USD wallet; a BUY order needs USD as source. Best-effort
	// resolution: the order still emits with a nil source ref (the
	// not-currently-held case) and a resolved destination ref.
	wallets := map[string]string{"BTC": "BTC"}
	got, err := OrderEntryToPSPOrder(testCurrencies, testPairs, wallets,
		OrderEntryWithID{OrderID: "OW", Order: filledOrder("buy", "limit")})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil {
		t.Fatal("expected order, got nil")
	}
	if got.SourceAccountReference != nil {
		t.Errorf("unresolved source should be nil, got %q", *got.SourceAccountReference)
	}
	if got.DestinationAccountReference == nil || *got.DestinationAccountReference != "BTC" {
		t.Errorf("destination should resolve to BTC, got %v", got.DestinationAccountReference)
	}
}

func TestOrderEntryToPSPOrder_ClientOrderID(t *testing.T) {
	t.Parallel()
	oe := filledOrder("buy", "limit")
	oe.ClOrdID = "my-client-id-42"
	got, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "OCL", Order: oe})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ClientOrderID != "my-client-id-42" {
		t.Errorf("ClientOrderID=%q want my-client-id-42", got.ClientOrderID)
	}
	if got.Metadata[MetadataPrefix+"cl_ord_id"] != "my-client-id-42" {
		t.Errorf("cl_ord_id metadata missing: %v", got.Metadata)
	}
}

func TestOrderEntryToPSPOrder_UnknownDirection(t *testing.T) {
	t.Parallel()
	oe := filledOrder("???", "limit")
	_, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "OD", Order: oe})
	if err == nil {
		t.Fatal("expected direction error")
	}
}

func TestOrderEntryToPSPOrder_BadVolReturnsError(t *testing.T) {
	t.Parallel()
	oe := filledOrder("buy", "limit")
	oe.Vol = "not-a-number"
	_, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "OB", Order: oe})
	if err == nil {
		t.Fatal("expected vol parse error")
	}
}

func TestOrderEntryToPSPOrder_BlankFeeIsZero(t *testing.T) {
	t.Parallel()
	oe := filledOrder("sell", "market")
	oe.Status = "canceled"
	oe.Fee = ""
	got, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "OC", Order: oe})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Fee == nil || got.Fee.Sign() != 0 {
		t.Fatalf("blank fee → zero: got %v", got.Fee)
	}
}

func TestDynamicOrderPricePrecision_RespectsCap(t *testing.T) {
	t.Parallel()
	overlyPrecise := "1.12345678901234567890" // 20 fractional digits
	got := dynamicOrderPricePrecision(overlyPrecise)
	if got != maxPricePrecisionCap {
		t.Fatalf("precision = %d, want cap %d", got, maxPricePrecisionCap)
	}
}

func TestParseOptionalPrice_ZeroOrEmpty(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "0", "0.0000"} {
		got, err := parseOptionalPrice(s, 4)
		if err != nil {
			t.Fatalf("parseOptionalPrice(%q): err %v", s, err)
		}
		if got != nil {
			t.Fatalf("parseOptionalPrice(%q): expected nil, got %v", s, got)
		}
	}
}

func TestMapOrderType_AdditionalCoverage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want models.OrderType
		ok   bool
	}{
		{"stop-loss", models.ORDER_TYPE_STOP, true},
		{"trailing-stop-limit", models.ORDER_TYPE_TRAILING_STOP_LIMIT, true},
		{"take-profit-limit", models.ORDER_TYPE_TAKE_PROFIT_LIMIT, true},
		{"limit-maker", models.ORDER_TYPE_LIMIT_MAKER, true},
	}
	for _, c := range cases {
		got, ok := MapOrderType(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("MapOrderType(%q) = (%v, %v) want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestResolvePair(t *testing.T) {
	t.Parallel()
	res, ok := ResolvePair(testPairs, "XXBTZUSD")
	if !ok {
		t.Fatal("expected resolution")
	}
	if res.BaseSymbol != "BTC" || res.QuoteSymbol != "USD" {
		t.Errorf("base=%q quote=%q", res.BaseSymbol, res.QuoteSymbol)
	}
}

func TestResolvePair_FallbackForms(t *testing.T) {
	t.Parallel()
	// Each of these non-primary-key forms must resolve via the fallback:
	// altname, wsname (with slash), and a slash-stripped wsname.
	for _, in := range []string{"XBTUSD", "XBT/USD", "XBTUSD ", "xbt/usd"} {
		res, ok := ResolvePair(testPairs, in)
		if !ok {
			t.Fatalf("expected fallback resolution for %q", in)
		}
		if res.BaseSymbol != "BTC" || res.QuoteSymbol != "USD" {
			t.Errorf("%q → base=%q quote=%q", in, res.BaseSymbol, res.QuoteSymbol)
		}
		if res.Pair != "XXBTZUSD" {
			t.Errorf("%q → Pair=%q want canonical code XXBTZUSD", in, res.Pair)
		}
	}
	if _, ok := ResolvePair(testPairs, "NOPENOPE"); ok {
		t.Error("unknown pair must not resolve")
	}
}

func TestOrderEntryToPSPOrder_NonEmptyInvalidFeeErrors(t *testing.T) {
	t.Parallel()
	oe := filledOrder("sell", "market")
	oe.Fee = "not-a-number"
	if _, err := OrderEntryToPSPOrder(testCurrencies, testPairs, testWallets,
		OrderEntryWithID{OrderID: "OFEE", Order: oe}); err == nil {
		t.Fatal("a non-empty unparseable fee must return an error, not silently zero")
	}
}
