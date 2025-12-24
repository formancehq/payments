package activities_test

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/mock/gomock"
)

var _ = Describe("StorageConnectorsGetMetadata", func() {
	var (
		act    activities.Activities
		s      *storage.MockStorage
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		ctx    context.Context
		id     models.ConnectorID
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		s = storage.NewMockStorage(ctrl)
		act = activities.New(logger, nil, s, nil, nil, 0)
		ctx = context.Background()
		id = models.ConnectorID{Reference: uuid.New(), Provider: "test"}
	})

	It("returns default polling period when config is empty", func() {
		c := &models.Connector{ConnectorBase: models.ConnectorBase{ID: id, Provider: "test"}}
		s.EXPECT().ConnectorsGet(ctx, id).Return(c, nil)

		meta, err := act.StorageConnectorsGetMetadata(ctx, id)
		Expect(err).To(BeNil())
		Expect(meta).ToNot(BeNil())
		Expect(meta.ConnectorID).To(Equal(id))
		Expect(meta.PollingPeriod).To(Equal(models.DefaultConfig().PollingPeriod))
		Expect(meta.ScheduledForDeletion).To(BeFalse())
		Expect(meta.Provider).To(Equal("test"))
	})

	It("extracts polling period from config when provided", func() {
		// Build a config JSON with a custom polling period
		cfg := struct {
			Name          string `json:"name"`
			PollingPeriod string `json:"pollingPeriod"`
		}{
			Name:          "my-connector",
			PollingPeriod: (45 * time.Minute).String(),
		}
		raw, err := json.Marshal(cfg)
		Expect(err).To(BeNil())

		c := &models.Connector{
			ConnectorBase:        models.ConnectorBase{ID: id},
			Config:               json.RawMessage(raw),
			ScheduledForDeletion: true,
		}
		s.EXPECT().ConnectorsGet(ctx, id).Return(c, nil)

		meta, err := act.StorageConnectorsGetMetadata(ctx, id)
		Expect(err).To(BeNil())
		Expect(meta).ToNot(BeNil())
		Expect(meta.ConnectorID).To(Equal(id))
		Expect(meta.PollingPeriod).To(Equal(45 * time.Minute))
		Expect(meta.ScheduledForDeletion).To(BeTrue())
	})

	It("propagates storage errors as temporal errors", func() {
		s.EXPECT().ConnectorsGet(ctx, id).Return(nil, storage.ErrNotFound)

		meta, err := act.StorageConnectorsGetMetadata(ctx, id)
		Expect(meta).To(BeNil())
		Expect(err).To(MatchError(temporal.NewNonRetryableApplicationError(storage.ErrNotFound.Error(), activities.ErrTypeStorage, storage.ErrNotFound)))
	})
})
