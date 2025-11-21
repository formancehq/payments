//go:build it

package test_suite

import (
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/pkg/client/models/components"
	evts "github.com/formancehq/payments/pkg/events"
	. "github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This e2e test ensures that creating an account publishes the corresponding outbox event.
var _ = Context("Publishes events", Ordered, Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		app *deferred.Deferred[*Server]

		e chan *nats.Msg

		defaultConnectorID string
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                      stack,
			NatsURL:                    natsServer.GetValue().ClientURL(),
			PostgresConfiguration:      db.GetValue().ConnectionOptions(),
			TemporalNamespace:          temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:            temporalServer.GetValue().Address(),
			Output:                     GinkgoWriter,
			SkipOutboxScheduleCreation: false,
		}
	})

	BeforeEach(func() {
		// Subscribe to published events on NATS for this test
		e = Subscribe(GinkgoT(), app.GetValue())
		var err error
		defaultConnectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		uninstallConnector(ctx, app.GetValue(), defaultConnectorID)
		flushRemainingWorkflows(ctx)
	})

	It("creates an account on the default connector and publishes the account saved event", func() {
		// Create a simple v3 account on the default connector
		createResponse, err := app.GetValue().SDK().Payments.V3.CreateAccount(ctx, &components.V3CreateAccountRequest{
			Reference:   "ref",
			ConnectorID: defaultConnectorID,
			AccountName: "foo",
			Type:        "INTERNAL",
			CreatedAt:   time.Now().UTC(),
		})
		Expect(err).To(BeNil())
		Expect(createResponse.GetV3CreateAccountResponse().Data.ID).NotTo(BeEmpty())

		// Verify the published event for account saved is eventually received on NATS
		var expectedEventPayload = struct {
			ConnectorID string `json:"connectorID"`
			AccountID   string `json:"id"`
			Reference   string `json:"reference"`
			Type        string `json:"type"`
			Name        string `json:"name"`
			Provider    string `json:"provider"`
		}{
			ConnectorID: defaultConnectorID,
			AccountID:   createResponse.GetV3CreateAccountResponse().Data.ID,
			Reference:   "ref",
			Type:        "INTERNAL",
			Name:        "foo",
			Provider:    "dummypay",
		}
		Eventually(e, engine.OUTBOX_POLLING_PERIOD*3, engine.OUTBOX_POLLING_PERIOD).
			Should(Receive(Event(evts.EventTypeSavedAccounts, WithPayloadSubset(expectedEventPayload))))
	})
})
