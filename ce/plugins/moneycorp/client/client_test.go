package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/formancehq/payments/pkg/domain/httpwrapper"
	"github.com/stretchr/testify/require"
)

// newTestClient builds a client wired to a test HTTP server, using the real
// httpErrorCheckerFn so the 404 -> nil mapping is exercised exactly as in
// production. The OAuth apiTransport is intentionally bypassed.
func newTestClient(handler http.HandlerFunc) (*client, *httptest.Server) {
	server := httptest.NewServer(handler)
	httpClient := httpwrapper.NewClient(&httpwrapper.Config{
		HttpErrorCheckerFn: httpErrorCheckerFn,
		Timeout:            10 * time.Second,
	})
	return &client{httpClient: httpClient, endpoint: server.URL}, server
}

func TestHttpErrorCheckerFn(t *testing.T) {
	t.Parallel()

	// 404 is intentionally swallowed so read paths can treat it as an empty
	// state; write paths guard against the resulting nil body themselves.
	require.NoError(t, httpErrorCheckerFn(http.StatusNotFound))

	require.ErrorIs(t, httpErrorCheckerFn(http.StatusBadRequest), httpwrapper.ErrStatusCodeClientError)
	require.ErrorIs(t, httpErrorCheckerFn(http.StatusForbidden), httpwrapper.ErrStatusCodeClientError)
	require.ErrorIs(t, httpErrorCheckerFn(http.StatusInternalServerError), httpwrapper.ErrStatusCodeServerError)
	require.ErrorIs(t, httpErrorCheckerFn(http.StatusTooManyRequests), httpwrapper.ErrStatusCodeTooManyRequests)
	require.NoError(t, httpErrorCheckerFn(http.StatusOK))
}
