package activities_test

import (
	"context"
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

var _ = Describe("Activity StorageConnectorsStore", func() {
	var (
		act       activities.Activities
		p         *connectors.MockManager
		s         *storage.MockStorage
		evts      *events.Events
		publisher *TestPublisher
		logger    = logging.NewDefaultLogger(GinkgoWriter, true, false, false)

		connector   models.Connector
		oldConnID   *models.ConnectorID
		now         time.Time
		connectorID models.ConnectorID
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
		connector = models.Connector{
			ConnectorBase: models.ConnectorBase{
				ID:        connectorID,
				Name:      "name",
				CreatedAt: now,
				Provider:  "test",
			},
			Config: json.RawMessage(`{"a":1}`),
		}
		oc := models.ConnectorID{Provider: "test", Reference: uuid.New()}
		oldConnID = &oc
	})

	AfterEach(func() {
		publisher.Close()
	})

	It("returns error when storage.DecryptRaw fails", func(ctx SpecContext) {
		s.EXPECT().DecryptRaw(gomock.Any(), connector.Config).Return(nil, errors.New("dec-fail"))

		err := act.StorageConnectorsStore(ctx, connector, oldConnID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("dec-fail"))
	})

	It("returns error when storage.ConnectorsInstall fails after decrypt", func(ctx SpecContext) {
		decrypted := json.RawMessage(`{"b":2}`)

		s.EXPECT().DecryptRaw(gomock.Any(), connector.Config).Return(decrypted, nil)
		s.EXPECT().ConnectorsInstall(gomock.Any(), gomock.Any(), oldConnID).
			DoAndReturn(func(_ context.Context, c models.Connector, old *models.ConnectorID) error {
				Expect(string(c.Config)).To(Equal(string(decrypted)))
				Expect(old).NotTo(BeNil())
				Expect(*old).To(Equal(*oldConnID))
				return errors.New("install-fail")
			})

		err := act.StorageConnectorsStore(ctx, connector, oldConnID)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("install-fail"))
	})

	It("passes through plain-text config when DecryptRaw signals not encrypted", func(ctx SpecContext) {
		// DecryptRaw indicates the config is not encrypted
		s.EXPECT().DecryptRaw(gomock.Any(), connector.Config).Return(nil, storage.ErrNotEncrypted)

		// ConnectorsInstall should be called with the original config
		s.EXPECT().ConnectorsInstall(gomock.Any(), gomock.Any(), oldConnID).
			DoAndReturn(func(_ context.Context, c models.Connector, old *models.ConnectorID) error {
				Expect(string(c.Config)).To(Equal(string(connector.Config)))
				Expect(old).NotTo(BeNil())
				Expect(*old).To(Equal(*oldConnID))
				return nil
			})

		err := act.StorageConnectorsStore(ctx, connector, oldConnID)
		Expect(err).NotTo(HaveOccurred())
	})

	It("succeeds and calls ConnectorsInstall with decrypted config and oldConnectorID", func(ctx SpecContext) {
		decrypted := json.RawMessage(`{"c":3}`)

		s.EXPECT().DecryptRaw(gomock.Any(), connector.Config).Return(decrypted, nil)
		s.EXPECT().ConnectorsInstall(gomock.Any(), gomock.Any(), oldConnID).
			DoAndReturn(func(_ context.Context, c models.Connector, old *models.ConnectorID) error {
				Expect(string(c.Config)).To(Equal(string(decrypted)))
				Expect(old).NotTo(BeNil())
				Expect(*old).To(Equal(*oldConnID))
				return nil
			})

		err := act.StorageConnectorsStore(ctx, connector, oldConnID)
		Expect(err).NotTo(HaveOccurred())
	})
})
