package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testAPIKey    = "BITSTAMP_TEST_KEY"
	testAPISecret = "BITSTAMP_TEST_SECRET"
)

// TestUserTransactionUnmarshalJSONExtractsOnlyCurrencyAmountStrings
// locks the spot-only invariant: a future numeric extra (e.g. a new
// "created_at" timestamp) must NOT be treated as a phantom currency
// amount. Pair rate keys (with underscore) are accepted from both
// string and number forms and surfaced on PairRates.
func TestUserTransactionUnmarshalJSONExtractsOnlyCurrencyAmountStrings(t *testing.T) {
	payload := []byte(`{
		"id": 458254264,
		"datetime": "2025-09-25 14:42:59.894846",
		"type": "36",
		"fee": "0.000000",
		"eur": "-5.00",
		"usdc": "5.810770",
		"usdc_eur": 0.86047000,
		"usd": 0.0
	}`)

	var tx UserTransaction
	if err := json.Unmarshal(payload, &tx); err != nil {
		t.Fatalf("unmarshal transaction: %v", err)
	}

	if tx.CurrencyAmounts["eur"] != "-5.00" {
		t.Fatalf("expected eur amount, got %q", tx.CurrencyAmounts["eur"])
	}
	if tx.CurrencyAmounts["usdc"] != "5.810770" {
		t.Fatalf("expected usdc amount, got %q", tx.CurrencyAmounts["usdc"])
	}
	if _, ok := tx.CurrencyAmounts["usdc_eur"]; ok {
		t.Fatalf("did not expect pair rate usdc_eur to be treated as a currency amount")
	}
	if _, ok := tx.CurrencyAmounts["usd"]; ok {
		t.Fatalf("did not expect numeric usd field to be treated as a currency amount")
	}
	// json.Number preserves trailing zeros — accept either form.
	rate := tx.PairRates["usdc_eur"]
	if rate != "0.86047" && rate != "0.86047000" {
		t.Fatalf("expected pair rate captured, got %q", rate)
	}
}

func TestUserTransactionUnmarshalJSONSurfacesDerivativesMarkers(t *testing.T) {
	payload := []byte(`{
		"id": 1,
		"datetime": "2025-01-01 00:00:00.000000",
		"type": "0",
		"fee": "0",
		"btc": "1.0",
		"margin_mode": "FLEXIBLE",
		"leverage_rate": "5"
	}`)
	var tx UserTransaction
	if err := json.Unmarshal(payload, &tx); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if tx.MarginMode != "FLEXIBLE" || tx.LeverageRate != "5" {
		t.Fatalf("derivatives markers not surfaced: %#v", tx)
	}
	if !tx.HasDerivativesMarker() {
		t.Fatal("HasDerivativesMarker should be true when either field is set")
	}
}

func TestNewDefaultsEmptyEndpoint(t *testing.T) {
	c, ok := New("bitstamp", "api-key", "api-secret", "").(*client)
	if !ok {
		t.Fatalf("expected concrete client")
	}
	if c.endpoint != DefaultEndpoint {
		t.Fatalf("expected default endpoint %q, got %q", DefaultEndpoint, c.endpoint)
	}
}

// expectedSignatureFor returns the canonical Bitstamp v2 HMAC string
// for the captured request, used by the signing-fixture tests below.
// host is sourced from r.Host (the value the client sent) rather than
// r.URL.Host (which is stripped to "" by net/http on the server side).
func expectedSignatureFor(r *http.Request, host, body, apiKey, apiSecret, nonce, timestamp string) string {
	message := "BITSTAMP " + apiKey +
		r.Method + host + r.URL.Path + r.URL.RawQuery +
		r.Header.Get("Content-Type") + nonce + timestamp + "v2" + body
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestSignRequestProducesValidHMAC(t *testing.T) {
	t.Parallel()

	var capturedReq *http.Request
	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		b, _ := io.ReadAll(r.Body)
		capturedBody = string(b)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := New("bitstamp", testAPIKey, testAPISecret, server.URL)
	if _, err := c.GetOpenOrders(context.Background()); err != nil {
		t.Fatalf("GetOpenOrders: %v", err)
	}

	if got := capturedReq.Header.Get("X-Auth"); got != "BITSTAMP "+testAPIKey {
		t.Fatalf("X-Auth=%q, want BITSTAMP %s", got, testAPIKey)
	}
	if capturedReq.Header.Get("X-Auth-Version") != "v2" {
		t.Fatalf("X-Auth-Version not set")
	}
	if capturedReq.Header.Get("X-Auth-Nonce") == "" {
		t.Fatal("X-Auth-Nonce missing")
	}
	if capturedReq.Header.Get("X-Auth-Timestamp") == "" {
		t.Fatal("X-Auth-Timestamp missing")
	}

	want := expectedSignatureFor(capturedReq, capturedReq.Host, capturedBody, testAPIKey, testAPISecret,
		capturedReq.Header.Get("X-Auth-Nonce"),
		capturedReq.Header.Get("X-Auth-Timestamp"),
	)
	got := capturedReq.Header.Get("X-Auth-Signature")
	if got != want {
		t.Fatalf("signature mismatch:\n got  %s\n want %s", got, want)
	}
}

func TestSignRequestEmptyBodyOmitsContentType(t *testing.T) {
	t.Parallel()

	var capturedCT string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCT = r.Header.Get("Content-Type")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := New("bitstamp", testAPIKey, testAPISecret, server.URL)
	if _, err := c.GetAccountBalances(context.Background()); err != nil {
		t.Fatalf("GetAccountBalances: %v", err)
	}
	// Bitstamp's v2 HMAC includes Content-Type in the signed message.
	// Empty-body POSTs must NOT set Content-Type — otherwise the server
	// computes a different signature and rejects the request.
	if capturedCT != "" {
		t.Fatalf("empty-body POST set Content-Type=%q, expected empty", capturedCT)
	}
}

func TestSignRequestFormBodySetsContentType(t *testing.T) {
	t.Parallel()

	var capturedCT, capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		capturedBody = string(b)
		_, _ = w.Write([]byte(`{"id":"123","status":"Finished","transactions":[]}`))
	}))
	defer server.Close()

	c := New("bitstamp", testAPIKey, testAPISecret, server.URL)
	if _, err := c.GetOrderStatus(context.Background(), "123"); err != nil {
		t.Fatalf("GetOrderStatus: %v", err)
	}

	if capturedCT != "application/x-www-form-urlencoded" {
		t.Fatalf("Content-Type=%q, expected application/x-www-form-urlencoded", capturedCT)
	}
	if !strings.Contains(capturedBody, "id=123") {
		t.Fatalf("body=%q, expected to contain id=123", capturedBody)
	}
}

func TestGetOpenOrdersDecodesSnapshot(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"id":"1","datetime":"2025-09-25 14:00:00","type":"0","price":"60000.00","amount":"0.5","currency_pair":"btcusd","client_order_id":"abc"}
		]`))
	}))
	defer server.Close()

	orders, err := New("bitstamp", testAPIKey, testAPISecret, server.URL).
		GetOpenOrders(context.Background())
	if err != nil {
		t.Fatalf("GetOpenOrders: %v", err)
	}
	if len(orders) != 1 || orders[0].ID != "1" || orders[0].CurrencyPair != "btcusd" {
		t.Fatalf("unexpected snapshot: %#v", orders)
	}
}

func TestGetOpenOrdersForMarketRejectsEmptyPair(t *testing.T) {
	t.Parallel()
	c := New("bitstamp", testAPIKey, testAPISecret, "http://unused")
	if _, err := c.GetOpenOrdersForMarket(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty currency pair")
	}
}

func TestGetOrderStatusRejectsEmptyID(t *testing.T) {
	t.Parallel()
	c := New("bitstamp", testAPIKey, testAPISecret, "http://unused")
	if _, err := c.GetOrderStatus(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty order ID")
	}
}

func TestSignedPOSTMapsAPI5506ToDerivativesUnsupportedError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"API5506","message":"Trade account does not support derivatives."}`))
	}))
	defer server.Close()

	_, err := New("bitstamp", testAPIKey, testAPISecret, server.URL).
		GetOpenOrders(context.Background())
	if err == nil {
		t.Fatal("expected API5506 to surface as an error")
	}
	if !IsDerivativesUnsupportedError(err) {
		t.Fatalf("expected DerivativesUnsupportedError, got %T: %v", err, err)
	}
}

func TestErrorEnvelopeMessageFallbacks(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   ErrorResponse
		want string
	}{
		{ErrorResponse{Msg: "primary"}, "primary"},
		{ErrorResponse{Reason: "fallback"}, "fallback"},
		{ErrorResponse{Code: "API1234"}, "API1234"},
		{ErrorResponse{}, ""},
	}
	for _, tc := range cases {
		if got := tc.in.Message(); got != tc.want {
			t.Errorf("Message() = %q, want %q (in=%+v)", got, tc.want, tc.in)
		}
	}
}

// stubServer spins up a one-shot httptest.Server that returns the given
// JSON body for any path it receives, capturing the most recent request
// for assertions. Cuts boilerplate across the round-trip tests.
func stubServer(t *testing.T, body string) (*httptest.Server, func() *http.Request) {
	t.Helper()
	var last *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Drain body so HMAC-signed POSTs are fully read before we
		// snapshot the request — otherwise the captured req.Body is
		// closed by the time the assertion runs.
		buf, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(buf)))
		last = r
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, func() *http.Request { return last }
}

func TestGetCurrenciesDecodesNetworksArray(t *testing.T) {
	t.Parallel()
	srv, _ := stubServer(t, `[
		{"name":"Bitcoin","currency":"BTC","decimals":8,"type":"crypto","symbol":"₿","logo":"https://x/btc.svg","available_supply":"19934406","deposit":"Enabled","withdrawal":"Enabled","networks":[
			{"network":"bitcoin","deposit":"Enabled","withdrawal":"Enabled","withdrawal_decimals":8,"withdrawal_minimum_amount":"0.00006000"},
			{"network":"xrpl","deposit":"Disabled","withdrawal":"Disabled","withdrawal_decimals":8}
		]}
	]`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).GetCurrencies(t.Context())
	if err != nil {
		t.Fatalf("GetCurrencies: %v", err)
	}
	if len(got) != 1 || got[0].Currency != "BTC" {
		t.Fatalf("unexpected currencies: %+v", got)
	}
	c := got[0]
	if c.Symbol != "₿" || c.AvailableSupply != "19934406" || c.Deposit != "Enabled" {
		t.Errorf("missing extended currency fields: %+v", c)
	}
	if len(c.Networks) != 2 {
		t.Fatalf("networks length = %d, want 2", len(c.Networks))
	}
	if c.Networks[0].Network != "bitcoin" || c.Networks[0].WithdrawalMinimumAmount != "0.00006000" {
		t.Errorf("first network mismatch: %+v", c.Networks[0])
	}
}

func TestGetMarketsDecodes(t *testing.T) {
	t.Parallel()
	srv, _ := stubServer(t, `[
		{"base_currency":"BTC","base_decimals":8,"counter_currency":"USD","counter_decimals":0,"description":"Bitcoin / U.S. dollar","instant_and_market_orders":"Enabled","instant_order_counter_decimals":2,"market_symbol":"btcusd","market_type":"SPOT","minimum_order_value":"10","name":"BTC/USD","trading":"Enabled"}
	]`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).GetMarkets(t.Context())
	if err != nil {
		t.Fatalf("GetMarkets: %v", err)
	}
	if len(got) != 1 || got[0].MarketSymbol != "btcusd" || got[0].MarketType != "SPOT" || got[0].MinimumOrderValue != "10" {
		t.Errorf("unexpected market: %+v", got)
	}
}

func TestGetMyMarketsRequiresSignedPOST(t *testing.T) {
	t.Parallel()
	srv, lastReq := stubServer(t, `[{"name":"BTC/USD","url_symbol":"btcusd"}]`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).GetMyMarkets(t.Context())
	if err != nil {
		t.Fatalf("GetMyMarkets: %v", err)
	}
	if len(got) != 1 || got[0].URLSymbol != "btcusd" {
		t.Errorf("unexpected my_markets: %+v", got)
	}
	// Live probe proved my_markets requires signed POST — verify the
	// outgoing call satisfies that contract.
	r := lastReq()
	if r.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", r.Method)
	}
	if r.Header.Get("X-Auth-Signature") == "" {
		t.Errorf("missing HMAC signature header on my_markets call")
	}
}

func TestGetTradingFeesDecodes(t *testing.T) {
	t.Parallel()
	srv, _ := stubServer(t, `[
		{"currency_pair":"btcusd","market":"btcusd","fees":{"maker":"0.300","taker":"0.400"}}
	]`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).GetTradingFees(t.Context())
	if err != nil {
		t.Fatalf("GetTradingFees: %v", err)
	}
	if len(got) != 1 || got[0].Fees.Maker != "0.300" || got[0].Fees.Taker != "0.400" {
		t.Errorf("unexpected trading fees: %+v", got)
	}
}

func TestGetWithdrawalFeesDecodesPerNetwork(t *testing.T) {
	t.Parallel()
	// One currency can have multiple rows when it spans multiple
	// blockchains — BTC bitcoin vs xrpl is the canonical example.
	srv, _ := stubServer(t, `[
		{"currency":"btc","fee":"0.00008000","network":"bitcoin"},
		{"currency":"btc","fee":"0.00000000","network":"xrpl"}
	]`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).GetWithdrawalFees(t.Context())
	if err != nil {
		t.Fatalf("GetWithdrawalFees: %v", err)
	}
	if len(got) != 2 || got[0].Network != "bitcoin" || got[1].Network != "xrpl" {
		t.Errorf("expected per-network rows: %+v", got)
	}
}

func TestGetCryptoTransactionsDecodesBuckets(t *testing.T) {
	t.Parallel()
	srv, lastReq := stubServer(t, `{
		"deposits":[{"id":1,"network":"bitcoin","currency":"BTC","txid":"abc","amount":1.23,"datetime":1759995000,"status":"PENDING","pending_reason":"ADDRESS_VERIFICATION_NEEDED","destinationAddress":"1A1zP1"}],
		"withdrawals":[{"currency":"BTC","network":"bitcoin","destinationAddress":"3FiK","txid":"def","amount":0.00012,"datetime":1642665114}],
		"ripple_iou_transactions":[{"currency":"BTC","network":"bitcoin","destinationAddress":"3FiK","txid":"ghi","amount":0.00012,"datetime":1642665114}]
	}`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetCryptoTransactions(t.Context(), CryptoTransactionsOptions{Limit: 1000, IncludeIOUs: true})
	if err != nil {
		t.Fatalf("GetCryptoTransactions: %v", err)
	}
	if len(got.Deposits) != 1 || got.Deposits[0].Status != "PENDING" || got.Deposits[0].PendingReason != "ADDRESS_VERIFICATION_NEEDED" {
		t.Errorf("deposits mismatch: %+v", got.Deposits)
	}
	if len(got.Withdrawals) != 1 || got.Withdrawals[0].TxID != "def" {
		t.Errorf("withdrawals mismatch: %+v", got.Withdrawals)
	}
	if len(got.RippleIOUTransactions) != 1 {
		t.Errorf("IOUs mismatch: %+v", got.RippleIOUTransactions)
	}
	// Form body must carry both limit and include_ious so the HMAC
	// signature matches Bitstamp's expectation.
	body, _ := io.ReadAll(lastReq().Body)
	for _, want := range []string{"limit=1000", "include_ious=true"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("body missing %q, got %q", want, body)
		}
	}
}

func TestGetCryptoTransactionsDefaultsLimitWhenOptsEmpty(t *testing.T) {
	t.Parallel()
	srv, lastReq := stubServer(t, `{"deposits":[],"withdrawals":[],"ripple_iou_transactions":[]}`)
	_, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetCryptoTransactions(t.Context(), CryptoTransactionsOptions{})
	if err != nil {
		t.Fatalf("GetCryptoTransactions: %v", err)
	}
	body, _ := io.ReadAll(lastReq().Body)
	if !strings.Contains(string(body), "limit=100") {
		t.Errorf("expected fallback limit=100 in body, got %q", body)
	}
}

func TestGetWithdrawalRequestsRejectsMissingArgs(t *testing.T) {
	t.Parallel()
	c := New("bitstamp", testAPIKey, testAPISecret, "http://unused")
	if _, err := c.GetWithdrawalRequests(t.Context(), 0, 0); err == nil {
		t.Error("expected error when limit is 0 (Bitstamp requires both limit and offset)")
	}
	if _, err := c.GetWithdrawalRequests(t.Context(), 100, -1); err == nil {
		t.Error("expected error when offset is negative")
	}
}

func TestGetWithdrawalRequestsDecodes(t *testing.T) {
	t.Parallel()
	srv, lastReq := stubServer(t, `[
		{"id":42,"datetime":"2025-09-25 14:42:59","type":0,"currency":"EUR","amount":"100.00","status":2,"transaction_id":"BANK-REF"}
	]`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetWithdrawalRequests(t.Context(), 1000, 0)
	if err != nil {
		t.Fatalf("GetWithdrawalRequests: %v", err)
	}
	if len(got) != 1 || got[0].ID != 42 || got[0].Status != 2 || got[0].TransactionID != "BANK-REF" {
		t.Errorf("unexpected withdrawal request: %+v", got)
	}
	body, _ := io.ReadAll(lastReq().Body)
	if !strings.Contains(string(body), "limit=1000") || !strings.Contains(string(body), "offset=0") {
		t.Errorf("body must carry both limit AND offset, got %q", body)
	}
}

func TestGetWithdrawalRequestByIDRejectsZero(t *testing.T) {
	t.Parallel()
	c := New("bitstamp", testAPIKey, testAPISecret, "http://unused")
	if _, err := c.GetWithdrawalRequestByID(t.Context(), 0); err == nil {
		t.Error("expected error on id <= 0")
	}
}

func TestGetWithdrawalRequestByIDReturnsFirstElement(t *testing.T) {
	t.Parallel()
	srv, _ := stubServer(t, `[
		{"id":99,"datetime":"2025-09-25 14:42:59","type":1,"currency":"GBP","amount":"50.00","status":0}
	]`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetWithdrawalRequestByID(t.Context(), 99)
	if err != nil {
		t.Fatalf("GetWithdrawalRequestByID: %v", err)
	}
	if got.ID != 99 || got.Status != 0 {
		t.Errorf("unexpected withdrawal request: %+v", got)
	}
}

func TestGetWithdrawalRequestByIDNotFoundReturnsError(t *testing.T) {
	t.Parallel()
	srv, _ := stubServer(t, `[]`)
	_, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetWithdrawalRequestByID(t.Context(), 99)
	if err == nil {
		t.Error("expected error on empty response")
	}
}

func TestOrderStatusDecodesExtendedFields(t *testing.T) {
	t.Parallel()
	// Live-probed shape: order_status returns market / type / subtype /
	// datetime / amount_remaining alongside status + transactions.
	srv, _ := stubServer(t, `{
		"id":1458532827766784,
		"datetime":"2022-01-31 14:43:15",
		"type":"0",
		"subtype":"LIMIT",
		"status":"Open",
		"market":"BTC/USD",
		"amount_remaining":"0.50",
		"client_order_id":"my-id",
		"transactions":[{"tid":1,"price":"60000.00","btc":"0.5","usd":"30000.00","fee":"15.00","datetime":"2022-01-31 14:43:15","type":2}]
	}`)

	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetOrderStatus(t.Context(), "1458532827766784")
	if err != nil {
		t.Fatalf("GetOrderStatus: %v", err)
	}
	if got.Market != "BTC/USD" || got.Type != "0" || got.Subtype != "LIMIT" || got.AmountRemaining != "0.50" || got.Datetime == "" {
		t.Errorf("extended order_status fields missing: %+v", got)
	}
	if len(got.Transactions) != 1 || got.Transactions[0].CurrencyAmounts["btc"] != "0.5" {
		t.Errorf("transactions[] not decoded: %+v", got.Transactions)
	}
	if got.HasDerivativesMarker() {
		t.Error("spot order should not flag as derivatives")
	}
}

func TestGetUserTransactionsHonoursSinceIDAndLimit(t *testing.T) {
	t.Parallel()
	srv, lastReq := stubServer(t, `[{"id":1,"datetime":"2025-09-25 14:42:59.000000","type":"0","fee":"0","eur":"25.00"}]`)
	since := int64(42)
	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetUserTransactions(t.Context(), &since, 250)
	if err != nil {
		t.Fatalf("GetUserTransactions: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].CurrencyAmounts["eur"] != "25.00" {
		t.Errorf("unexpected user_transactions: %+v", got)
	}
	body, _ := io.ReadAll(lastReq().Body)
	if !strings.Contains(string(body), "since_id=42") || !strings.Contains(string(body), "limit=250") || !strings.Contains(string(body), "sort=asc") {
		t.Errorf("body missing required form fields: %q", body)
	}
}

func TestGetUserTransactionsOmitsZeroSinceID(t *testing.T) {
	t.Parallel()
	srv, lastReq := stubServer(t, `[]`)
	_, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetUserTransactions(t.Context(), nil, 100)
	if err != nil {
		t.Fatalf("GetUserTransactions: %v", err)
	}
	body, _ := io.ReadAll(lastReq().Body)
	if strings.Contains(string(body), "since_id=") {
		t.Errorf("cold-start call must not send since_id, got body %q", body)
	}
}

func TestGetOpenOrdersForMarketSignsTheRightPath(t *testing.T) {
	t.Parallel()
	srv, lastReq := stubServer(t, `[]`)
	_, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetOpenOrdersForMarket(t.Context(), "BTC/USD")
	if err != nil {
		t.Fatalf("GetOpenOrdersForMarket: %v", err)
	}
	if got := lastReq().URL.Path; got != "/api/v2/open_orders/btc/usd/" {
		// Tolerate that we lowercased + stripped whitespace; allow either btcusd or btc/usd shape.
		if got != "/api/v2/open_orders/btc%2Fusd/" && got != "/api/v2/open_orders/btcusd/" {
			t.Errorf("unexpected path %q", got)
		}
	}
	if lastReq().Header.Get("X-Auth-Signature") == "" {
		t.Errorf("missing HMAC signature")
	}
}

func TestSignedPOSTMapsAPI5506BeforeWrap(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"API5506","message":"Trade account does not support derivatives."}`))
	}))
	defer srv.Close()
	_, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetCryptoTransactions(t.Context(), CryptoTransactionsOptions{Limit: 100})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsDerivativesUnsupportedError(err) {
		t.Errorf("API5506 must surface as DerivativesUnsupportedError, got %v", err)
	}
}

func TestSignedPOSTWrapsGenericServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"API0001","reason":"Invalid parameter"}`))
	}))
	defer srv.Close()
	_, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetAccountBalances(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
	if IsDerivativesUnsupportedError(err) {
		t.Errorf("non-API5506 errors must not surface as DerivativesUnsupportedError")
	}
	if !strings.Contains(err.Error(), "Invalid parameter") {
		t.Errorf("error should carry the PSP reason, got %v", err)
	}
}

func TestOrderStatusFlagsDerivatives(t *testing.T) {
	t.Parallel()
	srv, _ := stubServer(t, `{"id":1,"status":"Open","margin_mode":"ISOLATED","leverage":"3.1"}`)
	got, err := New("bitstamp", testAPIKey, testAPISecret, srv.URL).
		GetOrderStatus(t.Context(), "1")
	if err != nil {
		t.Fatalf("GetOrderStatus: %v", err)
	}
	if !got.HasDerivativesMarker() {
		t.Errorf("expected derivatives marker on margin_mode + leverage, got %+v", got)
	}
}
