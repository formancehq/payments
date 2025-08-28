package test_suite

import (
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Payment Initiation", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		app *deferred.Deferred[*Server]
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

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("initiating a new transfer with v3", func() {
		var (
			e   chan *nats.Msg
			err error

			debtorID            string
			creditorID          string
			payReq              *components.V3InitiatePaymentRequest
			paymentInitiationID string

			connectorID string
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())

			debtorID, creditorID = setupDebtorAndCreditorV3Accounts(ctx, app.GetValue(), e, connectorID, createdAt)
			payReq = &components.V3InitiatePaymentRequest{
				Reference:            uuid.New().String(),
				ConnectorID:          connectorID,
				Description:          "some description",
				Type:                 "TRANSFER",
				Amount:               big.NewInt(3200),
				Asset:                "EUR/2",
				SourceAccountID:      &debtorID,
				DestinationAccountID: &creditorID,
				Metadata:             map[string]string{"key": "val"},
			}

			createResponse, err := app.GetValue().SDK().Payments.V3.InitiatePayment(ctx, pointer.For(false), payReq)
			Expect(err).To(BeNil())
			Expect(createResponse.GetV3InitiatePaymentResponse().Data.TaskID).ToNot(BeNil())
			Expect(*createResponse.GetV3InitiatePaymentResponse().Data.TaskID).To(Equal(""))              // task empty when not sending to PSP
			Expect(createResponse.GetV3InitiatePaymentResponse().Data.PaymentInitiationID).ToNot(BeNil()) // task nil when not sending to PSP
			paymentInitiationID = *createResponse.GetV3InitiatePaymentResponse().Data.PaymentInitiationID
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPaymentInitiation)))
			var msg = struct {
				Status string `json:"status"`
			}{
				Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION.String(),
			}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPaymentInitiationAdjustment, WithPayloadSubset(msg))))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("can be processed", func() {
			approveResponse, err := app.GetValue().SDK().Payments.V3.ApprovePaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(approveResponse.GetV3ApprovePaymentInitiationResponse().Data).NotTo(BeNil())
			taskID, err := models.TaskIDFromString(approveResponse.GetV3ApprovePaymentInitiationResponse().Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("create-transfer"))

			var paymentMsg = struct {
				ConnectorID          string `json:"connectorID"`
				SourceAccountID      string `json:"sourceAccountID,omitempty"`
				DestinationAccountID string `json:"destinationAccountID,omitempty"`
			}{
				ConnectorID:          connectorID,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}

			type PIAdjMsg struct {
				Status string `json:"status"`
			}

			processingPI := PIAdjMsg{
				Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING.String(),
			}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPaymentInitiationAdjustment, WithPayloadSubset(processingPI))))
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(paymentMsg))))
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPaymentInitiationRelatedPayment)))
			processedPI := PIAdjMsg{
				Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED.String(),
			}
			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPaymentInitiationAdjustment, WithPayloadSubset(processedPI))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveResponse.GetV3ApprovePaymentInitiationResponse().Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(string(getResponse.GetV3GetPaymentInitiationResponse().Data.Status)).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED.String()))
		})

		It("can be rejected", func() {
			_, err := app.GetValue().SDK().Payments.V3.RejectPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(err).To(BeNil())
			Expect(string(getResponse.GetV3GetPaymentInitiationResponse().Data.Status)).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED.String()))
		})

		It("cannot be reversed if the payment is unprocessed", func() {
			reverseResponse, err := app.GetValue().SDK().Payments.V3.ReversePaymentInitiation(ctx, paymentInitiationID, &components.V3ReversePaymentInitiationRequest{
				Reference:   uuid.New().String(),
				Description: payReq.Description,
				Amount:      payReq.Amount,
				Asset:       payReq.Asset,
				Metadata:    map[string]string{"reversal": "data"},
			})
			Expect(err).To(BeNil())
			Expect(reverseResponse.GetV3ReversePaymentInitiationResponse().Data.TaskID).NotTo(BeNil())
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(*reverseResponse.GetV3ReversePaymentInitiationResponse().Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_FAILED, WithError(workflow.ErrPaymentInitiationNotProcessed)))
		})

		It("can be reversed", func() {
			approveResponse, err := app.GetValue().SDK().Payments.V3.ApprovePaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())

			var msg = struct {
				ConnectorID          string `json:"connectorID"`
				SourceAccountID      string `json:"sourceAccountID,omitempty"`
				DestinationAccountID string `json:"destinationAccountID,omitempty"`
			}{
				ConnectorID:          connectorID,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}
			Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(msg))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveResponse.GetV3ApprovePaymentInitiationResponse().Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			reverseResponse, err := app.GetValue().SDK().Payments.V3.ReversePaymentInitiation(ctx, paymentInitiationID, &components.V3ReversePaymentInitiationRequest{
				Reference:   uuid.New().String(),
				Description: payReq.Description,
				Amount:      payReq.Amount,
				Asset:       payReq.Asset,
				Metadata:    map[string]string{"reversal": "data"},
			})
			Expect(err).To(BeNil())
			Expect(reverseResponse.GetV3ReversePaymentInitiationResponse().Data.TaskID).NotTo(BeNil())
			blockTillWorkflowComplete(ctx, connectorID, "reverse-transfer")
			Eventually(taskPoller(*reverseResponse.GetV3ReversePaymentInitiationResponse().Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(string(getResponse.GetV3GetPaymentInitiationResponse().Data.Status)).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED.String()))
		})
	})

	When("initiating a new payout with v3", func() {
		var (
			e   chan *nats.Msg
			err error

			debtorID            string
			creditorID          string
			payReq              *components.V3InitiatePaymentRequest
			paymentInitiationID string

			connectorID string
		)

		BeforeEach(func() {
			e = Subscribe(GinkgoT(), app.GetValue())
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())

			debtorID, creditorID = setupDebtorAndCreditorV3Accounts(ctx, app.GetValue(), e, connectorID, createdAt)
			payReq = &components.V3InitiatePaymentRequest{
				Reference:            uuid.New().String(),
				ConnectorID:          connectorID,
				Description:          "payout description",
				Type:                 "PAYOUT",
				Amount:               big.NewInt(2233),
				Asset:                "EUR/2",
				SourceAccountID:      &debtorID,
				DestinationAccountID: &creditorID,
				Metadata:             map[string]string{"pay": "out"},
			}

			createResponse, err := app.GetValue().SDK().Payments.V3.InitiatePayment(ctx, pointer.For(false), payReq)
			Expect(err).To(BeNil())
			Expect(createResponse.GetV3InitiatePaymentResponse().Data.TaskID).ToNot(BeNil())              // task nil when not sending to PSP
			Expect(*createResponse.GetV3InitiatePaymentResponse().Data.TaskID).To(Equal(""))              // task nil when not sending to PSP
			Expect(createResponse.GetV3InitiatePaymentResponse().Data.PaymentInitiationID).ToNot(BeNil()) // task nil when not sending to PSP
			paymentInitiationID = *createResponse.GetV3InitiatePaymentResponse().Data.PaymentInitiationID
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("can be processed", func() {
			approveResponse, err := app.GetValue().SDK().Payments.V3.ApprovePaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(approveResponse.GetV3ApprovePaymentInitiationResponse().Data).NotTo(BeNil())
			taskID, err := models.TaskIDFromString(approveResponse.GetV3ApprovePaymentInitiationResponse().Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("create-payout"))

			var msg = struct {
				ConnectorID          string `json:"connectorID"`
				SourceAccountID      string `json:"sourceAccountID,omitempty"`
				DestinationAccountID string `json:"destinationAccountID,omitempty"`
			}{
				ConnectorID:          connectorID,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}
			Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(msg))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveResponse.GetV3ApprovePaymentInitiationResponse().Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(string(getResponse.GetV3GetPaymentInitiationResponse().Data.Status)).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED.String()))
		})

		It("can be rejected", func() {
			_, err := app.GetValue().SDK().Payments.V3.RejectPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(string(getResponse.GetV3GetPaymentInitiationResponse().Data.Status)).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED.String()))
		})

		It("cannot be reversed if the payment is unprocessed", func() {
			reverseResponse, err := app.GetValue().SDK().Payments.V3.ReversePaymentInitiation(ctx, paymentInitiationID, &components.V3ReversePaymentInitiationRequest{
				Reference:   uuid.New().String(),
				Description: payReq.Description,
				Amount:      payReq.Amount,
				Asset:       payReq.Asset,
				Metadata:    map[string]string{"reversal": "data"},
			})
			Expect(err).To(BeNil())
			Expect(reverseResponse.GetV3ReversePaymentInitiationResponse().Data.TaskID).NotTo(BeNil())
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(*reverseResponse.GetV3ReversePaymentInitiationResponse().Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_FAILED, WithError(workflow.ErrPaymentInitiationNotProcessed)))
		})

		It("can be reversed", func() {
			approveResponse, err := app.GetValue().SDK().Payments.V3.ApprovePaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())

			var msg = struct {
				ConnectorID          string `json:"connectorID"`
				SourceAccountID      string `json:"sourceAccountID,omitempty"`
				DestinationAccountID string `json:"destinationAccountID,omitempty"`
			}{
				ConnectorID:          connectorID,
				SourceAccountID:      debtorID,
				DestinationAccountID: creditorID,
			}
			Eventually(e).WithTimeout(2 * time.Second).Should(Receive(Event(evts.EventTypeSavedPayments, WithPayloadSubset(msg))))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			Eventually(taskPoller(approveResponse.GetV3ApprovePaymentInitiationResponse().Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			reverseResponse, err := app.GetValue().SDK().Payments.V3.ReversePaymentInitiation(ctx, paymentInitiationID, &components.V3ReversePaymentInitiationRequest{
				Reference:   uuid.New().String(),
				Description: payReq.Description,
				Amount:      payReq.Amount,
				Asset:       payReq.Asset,
				Metadata:    map[string]string{"reversal": "data"},
			})
			Expect(err).To(BeNil())
			Expect(reverseResponse.GetV3ReversePaymentInitiationResponse().Data.TaskID).NotTo(BeNil())
			blockTillWorkflowComplete(ctx, connectorID, "reverse-payout")
			Eventually(taskPoller(*reverseResponse.GetV3ReversePaymentInitiationResponse().Data.TaskID)).WithTimeout(2 * time.Second).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPaymentInitiation(ctx, paymentInitiationID)
			Expect(err).To(BeNil())
			Expect(string(getResponse.GetV3GetPaymentInitiationResponse().Data.Status)).To(Equal(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED.String()))
		})
	})
})
