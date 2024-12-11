package test_suite

import (
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Payment Initiation", func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		creditorRequest v3.CreateAccountRequest
		creditorRes     struct{ Data models.Account }
		debtorRes       struct{ Data models.Account }

		app *utils.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                 stack,
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			NatsURL:               natsServer.GetValue().ClientURL(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	createdAt, _ := time.Parse("2006-Jan-02", "2024-Nov-29")
	creditorRequest = v3.CreateAccountRequest{
		Reference:    "creditor",
		AccountName:  "creditor",
		CreatedAt:    createdAt,
		DefaultAsset: "EUR",
		Type:         string(models.ACCOUNT_TYPE_INTERNAL),
		Metadata:     map[string]string{"key": "val"},
	}

	When("initiating a new transfer with v3", func() {
		var (
			ver = 3
			e   chan *nats.Msg
			err error

			debtorID   string
			creditorID string
			payReq     v3.PaymentInitiationsCreateRequest

			connectorRes struct{ Data string }
			initRes      struct {
				Data v3.PaymentInitiationsCreateResponse
			}
			approveRes struct{ Data models.Task }
		)

		JustBeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorConf := newConnectorConfigurationFn()(uuid.New())
			err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			creditorRequest.ConnectorID = connectorRes.Data
			err = CreateAccount(ctx, app.GetValue(), ver, creditorRequest, &creditorRes)
			Expect(err).To(BeNil())

			debtorRequest := v3.CreateAccountRequest{
				Reference:    "debtor",
				AccountName:  "debtor",
				ConnectorID:  connectorRes.Data,
				CreatedAt:    createdAt,
				DefaultAsset: "EUR",
				Type:         string(models.ACCOUNT_TYPE_EXTERNAL),
				Metadata:     map[string]string{"ping": "pong"},
			}
			err = CreateAccount(ctx, app.GetValue(), ver, debtorRequest, &debtorRes)
			Expect(err).To(BeNil())
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))

			debtorID = debtorRes.Data.ID.String()
			creditorID = creditorRes.Data.ID.String()
			payReq = v3.PaymentInitiationsCreateRequest{
				Reference:            uuid.New().String(),
				ScheduledAt:          time.Now(),
				ConnectorID:          connectorRes.Data,
				Description:          "some description",
				Type:                 models.PAYMENT_INITIATION_TYPE_TRANSFER.String(),
				Amount:               big.NewInt(3200),
				Asset:                "EUR",
				SourceAccountID:      &debtorID,
				DestinationAccountID: &creditorID,
				Metadata:             map[string]string{"key": "val"},
			}

			err := CreatePaymentInitiation(ctx, app.GetValue(), ver, payReq, &initRes)
			Expect(err).To(BeNil())
			Expect(initRes.Data.TaskID).To(Equal("")) // task nil when not sending to PSP
		})

		It("can be processed", func() {
			paymentID, err := models.PaymentInitiationIDFromString(initRes.Data.PaymentInitiationID)
			Expect(err).To(BeNil())

			err = ApprovePaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), &approveRes)
			Expect(err).To(BeNil())
			Expect(approveRes.Data).NotTo(BeNil())
			Expect(approveRes.Data.ID.Reference).To(ContainSubstring("create-transfer"))

			var msg = struct {
				ConnectorID          string `json:"connectorId"`
				SourceAccountID      string `json:"sourceAccountId,omitempty"`
				DestinationAccountID string `json:"destinationAccountId,omitempty"`
			}{
				ConnectorID:          connectorRes.Data,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(msg))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveRes.Data.ID.String())).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			var paymentRes struct {
				Data models.PaymentInitiationExpanded
			}
			err = GetPaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), &paymentRes)
			Expect(err).To(BeNil())
			Expect(paymentRes.Data.Status).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED))
		})

		It("can be rejected", func() {
			paymentID, err := models.PaymentInitiationIDFromString(initRes.Data.PaymentInitiationID)
			Expect(err).To(BeNil())

			err = RejectPaymentInitiation(ctx, app.GetValue(), ver, paymentID.String())
			Expect(err).To(BeNil())

			var paymentRes struct {
				Data models.PaymentInitiationExpanded
			}
			err = GetPaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), &paymentRes)
			Expect(err).To(BeNil())
			Expect(paymentRes.Data.Status).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED))
		})
	})
})
