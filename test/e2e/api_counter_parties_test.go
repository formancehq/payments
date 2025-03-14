//go:build it

package test_suite

import (
	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	evts "github.com/formancehq/payments/pkg/events"
	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Counter Parties", func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		accountNumber = "123456789"
		iban          = "DE89370400440532013000"
		createRequest v3.CounterPartiesCreateRequest

		app *utils.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                 stack,
			NatsURL:               natsServer.GetValue().ClientURL(),
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	createRequest = v3.CounterPartiesCreateRequest{
		Name: "test",
		BankAccountInformation: &v3.BankAccountInformationRequest{
			AccountNumber: &accountNumber,
			IBAN:          &iban,
		},
		ContactDetails: &v3.ContactDetailsRequest{
			Email: pointer.For("test"),
			Phone: pointer.For("test"),
		},
		Address: &v3.AddressRequest{
			StreetName:   pointer.For("test"),
			StreetNumber: pointer.For("test"),
			City:         pointer.For("test"),
			PostalCode:   pointer.For("test"),
			Country:      pointer.For("FR"),
		},
		Metadata: map[string]string{
			"foo": "bar",
		},
	}

	When("creating a new counter parties with v3", func() {
		var (
			createResponse struct{ Data string }
			getResponse    struct{ Data models.CounterParty }
			err            error
		)
		JustBeforeEach(func() {
			err = CreateCounterParty(ctx, app.GetValue(), createRequest, &createResponse)
		})
		It("should be ok", func() {
			Expect(err).To(BeNil())
			id, err := uuid.Parse(createResponse.Data)
			Expect(err).To(BeNil())
			err = GetCounterParty(ctx, app.GetValue(), id.String(), &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.ID.String()).To(Equal(id.String()))

		})
	})

	When("forwarding a counter party to a connector with v3", func() {
		var (
			createRes    struct{ Data string }
			forwardReq   v3.CounterPartiesForwardToConnectorRequest
			connectorRes struct{ Data string }
			res          struct {
				Data v3.CounterPartiesForwardToConnectorResponse
			}
			err error
			e   chan *nats.Msg
			id  uuid.UUID
		)
		JustBeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			err = CreateCounterParty(ctx, app.GetValue(), createRequest, &createRes)
			Expect(err).To(BeNil())
			id, err = uuid.Parse(createRes.Data)
			Expect(err).To(BeNil())

			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), 3, connectorConf, &connectorRes)
			Expect(err).To(BeNil())
		})

		It("should fail when connector ID is invalid", func() {
			forwardReq = v3.CounterPartiesForwardToConnectorRequest{ConnectorID: "invalid"}
			err = ForwardCounterParty(ctx, app.GetValue(), id.String(), &forwardReq, &res)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("400"))
		})
		It("should be ok when connector is installed", func() {
			forwardReq = v3.CounterPartiesForwardToConnectorRequest{ConnectorID: connectorRes.Data}
			err = ForwardCounterParty(ctx, app.GetValue(), id.String(), &forwardReq, &res)
			Expect(err).To(BeNil())
			taskID, err := models.TaskIDFromString(res.Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring(id.String()))
			cID := models.MustConnectorIDFromString(connectorRes.Data)
			Expect(taskID.Reference).To(ContainSubstring(cID.Reference.String()))

			_, err = models.ConnectorIDFromString(connectorRes.Data)
			Expect(err).To(BeNil())

			var getResponse struct{ Data models.BankAccount }
			err = GetCounterParty(ctx, app.GetValue(), id.String(), &getResponse)
			Expect(err).To(BeNil())

			Eventually(e).Should(Receive(Event(evts.EventTypeSavedBankAccount)))
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedCounterParty)))
		})
	})
})
