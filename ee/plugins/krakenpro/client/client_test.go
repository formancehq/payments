package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/formancehq/payments/pkg/domain/httpwrapper"
)

const testSecret = "YWJjZA==" // "abcd"

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New("krakenpro-test", "key", testSecret, srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv, c
}

func TestPublicCallDecodesEnvelope(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/0/public/Assets" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"error":[],"result":{"XXBT":{"altname":"XBT","decimals":8}}}`)
	})

	assets, err := c.GetAssets(context.Background())
	if err != nil {
		t.Fatalf("GetAssets: %v", err)
	}
	if a, ok := assets["XXBT"]; !ok || a.Decimals != 8 {
		t.Errorf("assets[XXBT] = %#v", a)
	}
}

func TestPrivateCallSignsCorrectly(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method=%s", r.Method)
		}
		if r.Header.Get("api-key") != "key" {
			t.Errorf("api-key header missing")
		}
		nonce := r.Header.Get("api-nonce")
		if nonce == "" {
			t.Fatal("api-nonce missing")
		}
		body, _ := io.ReadAll(r.Body)
		// Re-derive the signature locally and compare.
		secret, _ := base64.StdEncoding.DecodeString(testSecret)
		sha := sha256.New()
		sha.Write([]byte(nonce))
		sha.Write(body)
		mac := hmac.New(sha512.New, secret)
		mac.Write([]byte(r.URL.Path))
		mac.Write(sha.Sum(nil))
		want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		if got := r.Header.Get("api-sign"); got != want {
			t.Errorf("api-sign mismatch:\n got %s\nwant %s", got, want)
		}
		// Body must contain the same nonce.
		if !strings.Contains(string(body), `"nonce":"`+nonce+`"`) {
			t.Errorf("nonce missing from body: %s", string(body))
		}
		_, _ = io.WriteString(w, `{"error":[],"result":{"XXBT":{"balance":"1.0","hold_trade":"0"}}}`)
	})

	res, err := c.GetBalanceEx(context.Background())
	if err != nil {
		t.Fatalf("GetBalanceEx: %v", err)
	}
	if res["XXBT"].Balance != "1.0" {
		t.Errorf("unexpected response: %#v", res)
	}
}

func TestPrivateCallSurfacesAPIError(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"error":["EAPI:Invalid key"]}`)
	})

	_, err := c.GetBalanceEx(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsAPIError(err) {
		t.Errorf("expected APIError, got %T", err)
	}
	if !IsFatalAuthError(err) {
		t.Errorf("EAPI:Invalid key should be fatal")
	}
}

func TestLedgersBuildsParams(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		if got["type"] != "conversion" {
			t.Errorf("type=%v want conversion", got["type"])
		}
		if got["without_count"] != true {
			t.Errorf("without_count=%v want true", got["without_count"])
		}
		if got["start"] == nil {
			t.Errorf("start missing")
		}
		_, _ = io.WriteString(w, `{"error":[],"result":{"ledger":{}}}`)
	})

	_, err := c.GetLedgers(context.Background(), LedgersParams{
		Start:        100.5,
		Type:         "conversion",
		WithoutCount: true,
	})
	if err != nil {
		t.Fatalf("GetLedgers: %v", err)
	}
}

func TestNewRejectsBadSecret(t *testing.T) {
	t.Parallel()
	if _, err := New("krakenpro-test", "key", "@@@not-base64@@@", "http://x"); err == nil {
		t.Fatal("expected base64 error")
	}
}

func TestIsFatalAuthErrorOnNonAuthCode(t *testing.T) {
	t.Parallel()
	err := &APIError{Endpoint: "/0/private/Balance", Code: "EService:Unavailable", All: []string{"EService:Unavailable"}}
	if IsFatalAuthError(err) {
		t.Error("EService should not be fatal")
	}
}

func TestIsFatalAuthErrorMatrix(t *testing.T) {
	t.Parallel()
	for _, code := range []string{
		"EAPI:Invalid key",
		"EAPI:Invalid signature",
		"EAPI:Bad request",
		"EGeneral:Permission denied",
		"EGeneral:Unknown method",
	} {
		if !IsFatalAuthError(&APIError{Code: code, All: []string{code}}) {
			t.Errorf("%q must be fatal — Temporal must stop retrying", code)
		}
	}
}

func TestInvalidNonceIsRetriableNotFatal(t *testing.T) {
	t.Parallel()
	// A shared key across worker pods makes the odd out-of-order nonce a
	// transient race: it must retry (with backoff), not hard-fail.
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error":["EAPI:Invalid nonce"]}`)
	})
	_, err := c.GetBalanceEx(context.Background())
	if IsFatalAuthError(err) {
		t.Fatal("EAPI:Invalid nonce must not be fatal")
	}
	if !IsRetriableError(err) {
		t.Fatal("EAPI:Invalid nonce must be retriable")
	}
	// Invalid nonce maps to TooEarly (stronger backoff than rate-limit)
	// because too many invalid-nonce errors temp-lock the key.
	if !errors.Is(err, httpwrapper.ErrStatusCodeTooEarly) {
		t.Fatalf("invalid nonce must map to the TooEarly backoff path, got %v", err)
	}
}

func TestPublicCallSendsNoAuthHeaders(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		for _, h := range []string{"api-key", "api-nonce", "api-sign"} {
			if r.Header.Get(h) != "" {
				t.Errorf("public request must not carry %s", h)
			}
		}
		_, _ = io.WriteString(w, `{"error":[],"result":{}}`)
	})
	if _, err := c.GetAssets(context.Background()); err != nil {
		t.Fatalf("GetAssets: %v", err)
	}
}

func TestGetAssetPairsDecodes(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/0/public/AssetPairs" {
			t.Errorf("path=%q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method=%s", r.Method)
		}
		_, _ = io.WriteString(w, `{"error":[],"result":{"XXBTZUSD":{"altname":"XBTUSD","wsname":"XBT/USD","base":"XXBT","quote":"ZUSD","pair_decimals":1,"lot_decimals":8}}}`)
	})
	pairs, err := c.GetAssetPairs(context.Background())
	if err != nil {
		t.Fatalf("GetAssetPairs: %v", err)
	}
	p, ok := pairs["XXBTZUSD"]
	if !ok {
		t.Fatal("missing XXBTZUSD")
	}
	if p.Wsname != "XBT/USD" || p.Base != "XXBT" || p.Quote != "ZUSD" {
		t.Fatalf("decoded pair: %+v", p)
	}
}

func TestGetClosedOrdersPassesAllParams(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/0/private/ClosedOrders" {
			t.Errorf("path=%q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		if got["trades"] != true {
			t.Errorf("trades=%v", got["trades"])
		}
		if got["start"] != 100.5 {
			t.Errorf("start=%v", got["start"])
		}
		if got["end"] != 200.5 {
			t.Errorf("end=%v", got["end"])
		}
		if got["ofs"] != float64(50) {
			t.Errorf("ofs=%v", got["ofs"])
		}
		if got["closetime"] != "both" {
			t.Errorf("closetime=%v", got["closetime"])
		}
		if got["without_count"] != true {
			t.Errorf("without_count=%v", got["without_count"])
		}
		_, _ = io.WriteString(w, `{"error":[],"result":{"closed":{}}}`)
	})

	_, err := c.GetClosedOrders(context.Background(), ClosedOrdersParams{
		Trades: true, Start: 100.5, End: 200.5, Offset: 50, Closetime: "both", WithoutCount: true,
	})
	if err != nil {
		t.Fatalf("GetClosedOrders: %v", err)
	}
}

func TestGetLedgersPassesEndAndType(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		if got["end"] != 200.5 {
			t.Errorf("end=%v", got["end"])
		}
		if got["ofs"] != float64(50) {
			t.Errorf("ofs=%v", got["ofs"])
		}
		_, _ = io.WriteString(w, `{"error":[],"result":{"ledger":{}}}`)
	})
	_, err := c.GetLedgers(context.Background(), LedgersParams{End: 200.5, Offset: 50})
	if err != nil {
		t.Fatalf("GetLedgers: %v", err)
	}
}

func TestPublicCallSurfacesAPIError(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error":["EService:Unavailable"]}`)
	})
	if _, err := c.GetAssets(context.Background()); !IsAPIError(err) {
		t.Fatalf("expected APIError, got %v", err)
	}
}

func TestPrivateCallSurfacesAPIErrorFromHTTPError(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":["EAPI:Invalid key"]}`)
	})
	_, err := c.GetBalanceEx(context.Background())
	if !IsAPIError(err) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if !IsFatalAuthError(err) {
		t.Fatal("EAPI:Invalid key should be fatal even on 4xx")
	}
}

func TestGetBalanceExDecodesCredit(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error":[],"result":{"ZUSD":{"balance":"100.0","hold_trade":"5.0","credit":"50.0","credit_used":"10.0"}}}`)
	})
	res, err := c.GetBalanceEx(context.Background())
	if err != nil {
		t.Fatalf("GetBalanceEx: %v", err)
	}
	e := res["ZUSD"]
	if e.Credit != "50.0" || e.CreditUsed != "10.0" {
		t.Fatalf("credit fields not decoded: %#v", e)
	}
}

func TestWarningOnlyEnvelopeReturnsResult(t *testing.T) {
	t.Parallel()
	// A W-prefixed warning can ride alongside a valid result; it must not
	// discard the result nor be treated as an error.
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error":["WGeneral:Notice"],"result":{"XXBT":{"balance":"2.5","hold_trade":"0"}}}`)
	})
	res, err := c.GetBalanceEx(context.Background())
	if err != nil {
		t.Fatalf("warning-only array must not fail the call, got %v", err)
	}
	if res["XXBT"].Balance != "2.5" {
		t.Fatalf("result not decoded: %#v", res)
	}
}

func TestErrorClassifiedPastLeadingWarning(t *testing.T) {
	t.Parallel()
	// A warning preceding a real error must not mask it: classification
	// picks the first E-prefixed code, so this stays a fatal-auth error.
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error":["WGeneral:Notice","EAPI:Invalid key"],"result":{}}`)
	})
	_, err := c.GetBalanceEx(context.Background())
	if !IsFatalAuthError(err) {
		t.Fatalf("expected fatal-auth classification past the leading warning, got %v", err)
	}
}

func TestErrorWithResultStillFails(t *testing.T) {
	t.Parallel()
	// An E code present means the request failed even if a (stub) result
	// object is echoed back — the error must win, not be swallowed.
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error":["EAPI:Invalid key"],"result":{"XXBT":{"balance":"2.5"}}}`)
	})
	_, err := c.GetBalanceEx(context.Background())
	if !IsFatalAuthError(err) {
		t.Fatalf("error alongside a result must still fail, got %v", err)
	}
}

func TestRateLimitEnvelopeMapsToTooManyRequests(t *testing.T) {
	t.Parallel()
	// Kraken signals rate limits in the error array, often on HTTP 200.
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error":["EAPI:Rate limit exceeded"]}`)
	})
	_, err := c.GetBalanceEx(context.Background())
	if !errors.Is(err, httpwrapper.ErrStatusCodeTooManyRequests) {
		t.Fatalf("expected ErrStatusCodeTooManyRequests mapping, got %v", err)
	}
	if !IsRateLimitError(err) {
		t.Fatal("IsRateLimitError should report true")
	}
	if !IsAPIError(err) {
		t.Fatal("underlying APIError must remain detectable via errors.As")
	}
}

func TestThrottledPrefixMapsToTooManyRequests(t *testing.T) {
	t.Parallel()
	_, c := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error":["EService:Throttled: 1700000000"]}`)
	})
	_, err := c.GetBalanceEx(context.Background())
	if !errors.Is(err, httpwrapper.ErrStatusCodeTooManyRequests) {
		t.Fatalf("expected rate-limit mapping for EService:Throttled, got %v", err)
	}
}

func TestNonceStrictlyIncreasesAcrossCalls(t *testing.T) {
	t.Parallel()
	// The nonce is a stateless UnixNano (no stored counter). Across
	// sequential signed calls it must be a strictly-increasing integer —
	// Kraken requires that per key.
	var nonces []int64
	_, c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.ParseInt(r.Header.Get("api-nonce"), 10, 64)
		if err != nil {
			t.Errorf("api-nonce not an integer: %q", r.Header.Get("api-nonce"))
		}
		nonces = append(nonces, n)
		_, _ = io.WriteString(w, `{"error":[],"result":{}}`)
	})
	for i := 0; i < 4; i++ {
		_, _ = c.GetBalanceEx(context.Background())
	}
	for i := 1; i < len(nonces); i++ {
		if nonces[i] <= nonces[i-1] {
			t.Errorf("nonce not strictly increasing: %d <= %d", nonces[i], nonces[i-1])
		}
	}
}
