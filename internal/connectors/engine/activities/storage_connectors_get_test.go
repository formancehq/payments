package activities_test

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Activity StorageConnectorsGet", func() {
	var (
		act       activities.Activities
		p         *connectors.MockManager
		s         *storage.MockStorage
		evts      *events.Events
		publisher *TestPublisher
		logger    = logging.NewDefaultLogger(GinkgoWriter, true, false, false)

		connectorID models.ConnectorID
		now         time.Time
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		p = connectors.NewMockManager(ctrl)
		s = storage.NewMockStorage(ctrl)
		publisher = newTestPublisher()
		evts = events.New(publisher, "")

		act = activities.New(logger, nil, s, evts, p, 0)

		connectorID = models.ConnectorID{Provider: "test", Reference: uuid.New()}
		now = time.Now().UTC()
	})

	AfterEach(func() {
		publisher.Close()
	})

	It("returns error when storage.ConnectorsGet fails", func(ctx SpecContext) {
		s.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(nil, errors.New("boom"))

		_, err := act.StorageConnectorsGet(ctx, connectorID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("boom"))
	})

	It("returns error when storage.EncryptRaw fails", func(ctx SpecContext) {
		connector := &models.Connector{
			ConnectorBase: models.ConnectorBase{
				ID:        connectorID,
				Name:      "name",
				CreatedAt: now,
				Provider:  "test",
			},
			Config: json.RawMessage(`{"a":1}`),
		}

		s.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
		s.EXPECT().EncryptRaw(gomock.Any(), connector.Config).Return(nil, errors.New("enc-fail"))

		_, err := act.StorageConnectorsGet(ctx, connectorID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("enc-fail"))
	})

	It("returns connector with encrypted config on success", func(ctx SpecContext) {
		plain := json.RawMessage(`{"a":1}`)
		encrypted := json.RawMessage(`encrypted-bytes`)

		connector := &models.Connector{
			ConnectorBase: models.ConnectorBase{
				ID:        connectorID,
				Name:      "name",
				CreatedAt: now,
				Provider:  "test",
			},
			Config: plain,
		}

		s.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
		s.EXPECT().EncryptRaw(gomock.Any(), connector.Config).Return(encrypted, nil)

		got, err := act.StorageConnectorsGet(ctx, connectorID)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).NotTo(BeNil())
		Expect(got.ID).To(Equal(connectorID))
		Expect(string(got.Config)).To(Equal(string(encrypted)))
	})
})
