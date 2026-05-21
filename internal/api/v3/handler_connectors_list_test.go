package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Connectors List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list connectors", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&paginate.Cursor[models.Connector]{}, fmt.Errorf("connectors list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&paginate.Cursor[models.Connector]{}, nil,
			)
			m.EXPECT().ConnectorsCapabilities().Return(map[string][]models.Capability{})
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})

		It("should inline capabilities for every row", func(ctx SpecContext) {
			connectorID := models.ConnectorID{Reference: uuid.New(), Provider: "stripe"}
			caps := []models.Capability{
				models.CAPABILITY_FETCH_ACCOUNTS,
				models.CAPABILITY_CREATE_TRANSFER,
			}
			m.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&paginate.Cursor[models.Connector]{Data: []models.Connector{{
					ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: "stripe"},
					Config:        json.RawMessage(`{}`),
				}}},
				nil,
			)
			m.EXPECT().ConnectorsCapabilities().Return(map[string][]models.Capability{"stripe": caps})

			handlerFn(w, httptest.NewRequest(http.MethodGet, "/", nil))

			res := w.Result()
			defer res.Body.Close()
			Expect(res.StatusCode).To(Equal(http.StatusOK))

			var body struct {
				Cursor struct {
					Data []struct {
						Provider     string              `json:"provider"`
						Capabilities []models.Capability `json:"capabilities"`
					} `json:"data"`
				} `json:"cursor"`
			}
			Expect(json.NewDecoder(res.Body).Decode(&body)).To(Succeed())
			Expect(body.Cursor.Data).To(HaveLen(1))
			Expect(body.Cursor.Data[0].Provider).To(Equal("stripe"))
			Expect(body.Cursor.Data[0].Capabilities).To(Equal(caps))
		})

		It("should emit an empty capabilities array for unregistered providers", func(ctx SpecContext) {
			connectorID := models.ConnectorID{Reference: uuid.New(), Provider: "ghost"}
			m.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&paginate.Cursor[models.Connector]{Data: []models.Connector{{
					ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: "ghost"},
					Config:        json.RawMessage(`{}`),
				}}},
				nil,
			)
			m.EXPECT().ConnectorsCapabilities().Return(map[string][]models.Capability{})

			handlerFn(w, httptest.NewRequest(http.MethodGet, "/", nil))

			res := w.Result()
			defer res.Body.Close()

			var body struct {
				Cursor struct {
					Data []struct {
						Capabilities []models.Capability `json:"capabilities"`
					} `json:"data"`
				} `json:"cursor"`
			}
			Expect(json.NewDecoder(res.Body).Decode(&body)).To(Succeed())
			Expect(body.Cursor.Data[0].Capabilities).To(BeEmpty())
		})
	})
})
