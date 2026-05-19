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
