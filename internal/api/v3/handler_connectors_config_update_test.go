package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Connector Config Update", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("update a connector config", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend

			connectorID models.ConnectorID
			connector   models.Connector
			config      models.Config
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsConfigUpdate(m)
			connectorID = models.ConnectorID{
				Reference: uuid.New(),
				Provider:  "dummypay",
			}
			connectorName := "some-name"
			connector = models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID,
					Name:     connectorName,
					Provider: connectorID.Provider,
				},
			}
			config = models.Config{PollingPeriod: 20 * time.Minute}
			config.Name = connectorName
			conf, err := config.MarshalJSON()
			require.Nil(GinkgoT(), err)
			connector.Config = conf
		})

		It("should return a validation error when connector ID is invalid", func(ctx SpecContext) {
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "connectorID", "invalidID", &config))

			assertExpectedResponse(w.Result(), http.StatusBadRequest, "INVALID_ID")
		})

		It("should return a validation error when request body is too big", func(ctx SpecContext) {
			data := oversizeRequestBody()
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "connectorID", connectorID.String(), &data))

			assertExpectedResponse(w.Result(), http.StatusRequestEntityTooLarge, "MISSING_OR_INVALID_BODY")
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			m.EXPECT().ConnectorsConfigUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				fmt.Errorf("connector update err"),
			)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "connectorID", connectorID.String(), &config))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			m.EXPECT().ConnectorsConfigUpdate(gomock.Any(), connector.ID, gomock.Any()).Return(nil)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "connectorID", connectorID.String(), &config))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
