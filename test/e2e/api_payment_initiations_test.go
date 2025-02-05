package test_suite

import (
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
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
			approveRes struct {
				Data v3.PaymentInitiationsApproveResponse
			}
		)

		JustBeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorConf := newConnectorConfigurationFn()(uuid.New())
			err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			debtorID, creditorID = setupDebtorAndCreditorAccounts(ctx, app.GetValue(), e, ver, connectorRes.Data, createdAt)
			payReq = v3.PaymentInitiationsCreateRequest{
				Reference:            uuid.New().String(),
				ConnectorID:          connectorRes.Data,
				Description:          "some description",
				Type:                 models.PAYMENT_INITIATION_TYPE_TRANSFER.String(),
				Amount:               big.NewInt(3200),
				Asset:                "EUR/2",
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
			taskID, err := models.TaskIDFromString(approveRes.Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("create-transfer"))

			var msg = struct {
				ConnectorID          string `json:"connectorId"`
				SourceAccountID      string `json:"sourceAccountId,omitempty"`
				DestinationAccountID string `json:"destinationAccountId,omitempty"`
			}{
				ConnectorID:          connectorRes.Data,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}
			Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(msg))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveRes.Data.TaskID)).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

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

		It("cannot be reversed if the payment is unprocessed", func() {
			paymentID, err := models.PaymentInitiationIDFromString(initRes.Data.PaymentInitiationID)
			Expect(err).To(BeNil())

			req := v3.PaymentInitiationsReverseRequest{
				Reference:   uuid.New().String(),
				Description: payReq.Description,
				Amount:      payReq.Amount,
				Asset:       payReq.Asset,
				Metadata:    map[string]string{"reversal": "data"},
			}

			var res struct {
				Data v3.PaymentInitiationsReverseResponse
			}
			err = ReversePaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), req, &res)
			Expect(err).To(BeNil())
			Expect(res.Data.TaskID).NotTo(BeNil())
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(res.Data.TaskID)).Should(HaveTaskStatus(models.TASK_STATUS_FAILED, WithError(workflow.ErrPaymentInitiationNotProcessed)))
		})

		It("can be reversed", func() {
			paymentID, err := models.PaymentInitiationIDFromString(initRes.Data.PaymentInitiationID)
			Expect(err).To(BeNil())

			err = ApprovePaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), &approveRes)
			Expect(err).To(BeNil())

			var msg = struct {
				ConnectorID          string `json:"connectorId"`
				SourceAccountID      string `json:"sourceAccountId,omitempty"`
				DestinationAccountID string `json:"destinationAccountId,omitempty"`
			}{
				ConnectorID:          connectorRes.Data,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}
			Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(msg))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveRes.Data.TaskID)).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			req := v3.PaymentInitiationsReverseRequest{
				Reference:   uuid.New().String(),
				Description: payReq.Description,
				Amount:      payReq.Amount,
				Asset:       payReq.Asset,
				Metadata:    map[string]string{"reversal": "data"},
			}

			var res struct {
				Data v3.PaymentInitiationsReverseResponse
			}
			err = ReversePaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), req, &res)
			Expect(err).To(BeNil())
			Expect(res.Data.TaskID).NotTo(BeNil())
			blockTillWorkflowComplete(ctx, connectorRes.Data, "reverse-transfer")
			Eventually(taskPoller(res.Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			var paymentRes struct {
				Data models.PaymentInitiationExpanded
			}
			err = GetPaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), &paymentRes)
			Expect(err).To(BeNil())
			Expect(paymentRes.Data.Status).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED))
		})
	})

	When("initiating a new payout with v3", func() {
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
			approveRes struct {
				Data v3.PaymentInitiationsApproveResponse
			}
		)

		JustBeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorConf := newConnectorConfigurationFn()(uuid.New())
			err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			debtorID, creditorID = setupDebtorAndCreditorAccounts(ctx, app.GetValue(), e, ver, connectorRes.Data, createdAt)
			payReq = v3.PaymentInitiationsCreateRequest{
				Reference:            uuid.New().String(),
				ConnectorID:          connectorRes.Data,
				Description:          "payout description",
				Type:                 models.PAYMENT_INITIATION_TYPE_PAYOUT.String(),
				Amount:               big.NewInt(2233),
				Asset:                "EUR/2",
				SourceAccountID:      &debtorID,
				DestinationAccountID: &creditorID,
				Metadata:             map[string]string{"pay": "out"},
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
			taskID, err := models.TaskIDFromString(approveRes.Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("create-payout"))

			var msg = struct {
				ConnectorID          string `json:"connectorId"`
				SourceAccountID      string `json:"sourceAccountId,omitempty"`
				DestinationAccountID string `json:"destinationAccountId,omitempty"`
			}{
				ConnectorID:          connectorRes.Data,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}
			Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(msg))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveRes.Data.TaskID)).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

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

		It("cannot be reversed if the payment is unprocessed", func() {
			paymentID, err := models.PaymentInitiationIDFromString(initRes.Data.PaymentInitiationID)
			Expect(err).To(BeNil())

			req := v3.PaymentInitiationsReverseRequest{
				Reference:   uuid.New().String(),
				Description: payReq.Description,
				Amount:      payReq.Amount,
				Asset:       payReq.Asset,
				Metadata:    map[string]string{"reversal": "data"},
			}

			var res struct {
				Data v3.PaymentInitiationsReverseResponse
			}
			err = ReversePaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), req, &res)
			Expect(err).To(BeNil())
			Expect(res.Data.TaskID).NotTo(BeNil())
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(res.Data.TaskID)).Should(HaveTaskStatus(models.TASK_STATUS_FAILED, WithError(workflow.ErrPaymentInitiationNotProcessed)))
		})

		It("can be reversed", func() {
			paymentID, err := models.PaymentInitiationIDFromString(initRes.Data.PaymentInitiationID)
			Expect(err).To(BeNil())

			err = ApprovePaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), &approveRes)
			Expect(err).To(BeNil())

			var msg = struct {
				ConnectorID          string `json:"connectorId"`
				SourceAccountID      string `json:"sourceAccountId,omitempty"`
				DestinationAccountID string `json:"destinationAccountId,omitempty"`
			}{
				ConnectorID:          connectorRes.Data,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}
			Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(msg))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveRes.Data.TaskID)).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			req := v3.PaymentInitiationsReverseRequest{
				Reference:   uuid.New().String(),
				Description: payReq.Description,
				Amount:      payReq.Amount,
				Asset:       payReq.Asset,
				Metadata:    map[string]string{"reversal": "data"},
			}

			var res struct {
				Data v3.PaymentInitiationsReverseResponse
			}
			err = ReversePaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), req, &res)
			Expect(err).To(BeNil())
			Expect(res.Data.TaskID).NotTo(BeNil())
			blockTillWorkflowComplete(ctx, connectorRes.Data, "reverse-payout")
			Eventually(taskPoller(res.Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			var paymentRes struct {
				Data models.PaymentInitiationExpanded
			}
			err = GetPaymentInitiation(ctx, app.GetValue(), ver, paymentID.String(), &paymentRes)
			Expect(err).To(BeNil())
			Expect(paymentRes.Data.Status).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED))
		})
	})
})
