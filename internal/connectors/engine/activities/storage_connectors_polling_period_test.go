package activities_test

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Activity StorageConnectorsGetPollingPeriod", func() {
	var (
		act       activities.Activities
		p         *connectors.MockManager
		s         *storage.MockStorage
		evts      *events.Events
		publisher *TestPublisher
		logger    = logging.NewDefaultLogger(GinkgoWriter, true, false, false)

		connectorID models.ConnectorID
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		p = connectors.NewMockManager(ctrl)
		s = storage.NewMockStorage(ctrl)
		publisher = newTestPublisher()
		evts = events.New(publisher, "")

		act = activities.New(logger, nil, s, evts, p, 0, 0)

		connectorID = models.ConnectorID{Provider: "test", Reference: uuid.New()}
	})

	AfterEach(func() {
		publisher.Close()
	})

	It("returns error when storage.ConnectorsGet fails", func(ctx SpecContext) {
		s.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(nil, errors.New("boom"))

		_, err := act.StorageConnectorsGetPollingPeriod(ctx, connectorID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("boom"))
	})

	It("returns the polling period from the config (and never the secret config)", func(ctx SpecContext) {
		connector := &models.Connector{
			ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: "test"},
			Config:        json.RawMessage(`{"name":"test","pollingPeriod":"2m","apiKey":"super-secret"}`),
		}
		s.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
		p.EXPECT().DefaultConfig().Return(models.Config{PollingPeriod: time.Minute})

		got, err := act.StorageConnectorsGetPollingPeriod(ctx, connectorID)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(2 * time.Minute))
	})

	It("falls back to the default polling period when the config omits it", func(ctx SpecContext) {
		connector := &models.Connector{
			ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: "test"},
			Config:        json.RawMessage(`{"name":"test"}`),
		}
		s.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
		p.EXPECT().DefaultConfig().Return(models.Config{PollingPeriod: 90 * time.Second})

		got, err := act.StorageConnectorsGetPollingPeriod(ctx, connectorID)
		Expect(err).NotTo(HaveOccurred())
		// EN-1093/H12: an absent pollingPeriod must resolve to the configurer default,
		// not zero — otherwise a zero-interval schedule would be created.
		Expect(got).To(Equal(90 * time.Second))
	})

	It("returns a non-retryable error on a corrupt config", func(ctx SpecContext) {
		connector := &models.Connector{
			ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: "test"},
			Config:        json.RawMessage(`{not-json`),
		}
		s.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
		p.EXPECT().DefaultConfig().Return(models.Config{PollingPeriod: time.Minute})

		_, err := act.StorageConnectorsGetPollingPeriod(ctx, connectorID)
		Expect(err).To(HaveOccurred())
		var appErr *temporal.ApplicationError
		Expect(errors.As(err, &appErr)).To(BeTrue())
		Expect(appErr.NonRetryable()).To(BeTrue())
	})
})
