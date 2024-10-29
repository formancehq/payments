package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Connector Install", func() {
	var (
		handlerFn http.HandlerFunc
		conn      string
		config    json.RawMessage
	)
	BeforeEach(func() {
		conn = "psp"
		config = json.RawMessage("{}")
	})

	Context("install connector", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsInstall(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			m.EXPECT().ConnectorsInstall(gomock.Any(), conn, config).Return(
				models.ConnectorID{},
				fmt.Errorf("connector install err"),
			)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "connector", conn, &config))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status accepted on success", func(ctx SpecContext) {
			m.EXPECT().ConnectorsInstall(gomock.Any(), conn, config).Return(
				models.ConnectorID{},
				nil,
			)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "connector", conn, &config))
			assertExpectedResponse(w.Result(), http.StatusAccepted, "data")
		})
	})
})
