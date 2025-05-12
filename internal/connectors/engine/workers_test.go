package engine_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/temporal"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Worker Tests", func() {
	Context("on start", func() {
		var (
			pool       *engine.WorkerPool
			store      *storage.MockStorage
			plgs       *plugins.MockPlugins
			connectors []models.Connector
		)
		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			logger := logging.NewDefaultLogger(GinkgoWriter, false, false, false)
			cl, err := client.NewLazyClient(client.Options{})
			Expect(err).To(BeNil())
			store = storage.NewMockStorage(ctrl)
			plgs = plugins.NewMockPlugins(ctrl)
			pool = engine.NewWorkerPool(logger, "stackname", cl, []temporal.DefinitionSet{}, []temporal.DefinitionSet{}, store, plgs, worker.Options{})

			connID1 := models.ConnectorID{Reference: uuid.New(), Provider: "provider1"}
			connID2 := models.ConnectorID{Reference: uuid.New(), Provider: "provider2"}

			connectors = []models.Connector{
				{ID: connID1, Name: "abc-connector", Provider: connID1.Provider, CreatedAt: time.Now().Add(-time.Minute), Config: json.RawMessage(`{}`)},
				{ID: connID2, Name: "efg-connector", Provider: connID2.Provider, CreatedAt: time.Now(), Config: json.RawMessage(`{}`)},
			}

		})

		It("should fail when listener fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("listener err")
			store.EXPECT().ListenConnectorsChanges(gomock.Any(), gomock.Any()).Return(expectedErr)
			err := pool.OnStart(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should fail when unable to fetch connectors from storage", func(ctx SpecContext) {
			store.EXPECT().ListenConnectorsChanges(gomock.Any(), gomock.Any()).Return(nil)

			expectedErr := fmt.Errorf("storage err")
			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(nil, expectedErr)
			err := pool.OnStart(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should call RegisterPlugin on all connectors found", func(ctx SpecContext) {
			store.EXPECT().ListenConnectorsChanges(gomock.Any(), gomock.Any()).Return(nil)

			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(&bunpaginate.Cursor[models.Connector]{
				Data: connectors,
			}, nil)
			plgs.EXPECT().LoadPlugin(connectors[0].ID, connectors[0].Provider, connectors[0].Name, gomock.Any(), connectors[0].Config, false).Return(nil)
			plgs.EXPECT().LoadPlugin(connectors[1].ID, connectors[1].Provider, connectors[1].Name, gomock.Any(), connectors[1].Config, false).Return(nil)
			err := pool.OnStart(ctx)
			Expect(err).To(BeNil())
		})
	})
})
