//go:build it

package test_suite

import (
	"context"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Payments", Serial, func() {
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

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("creating a new payment with v3", func() {
		var (
			connectorID   string
			createdAt     time.Time
			initialAmount *big.Int
			asset         string
			err           error
		)
		BeforeEach(func() {
			createdAt = time.Now()
			initialAmount = big.NewInt(1340)
			asset = "USD/2"

			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be ok", func() {
			adj := []components.V3CreatePaymentAdjustmentRequest{
				{
					Reference: "ref_adjustment",
					CreatedAt: createdAt,
					Amount:    big.NewInt(55),
					Asset:     &asset,
					Status:    "REFUNDED",
				},
			}

			debtorID, creditorID := setupDebtorAndCreditorV3Accounts(ctx, app.GetValue(), connectorID, createdAt)
			createRequest := &components.V3CreatePaymentRequest{
				Reference:            "ref",
				ConnectorID:          connectorID,
				CreatedAt:            createdAt,
				InitialAmount:        initialAmount,
				Amount:               initialAmount,
				Asset:                asset,
				Type:                 "PAY-IN",
				SourceAccountID:      &debtorID,
				DestinationAccountID: &creditorID,
				Scheme:               models.PAYMENT_SCHEME_CARD_AMEX.String(),
				Metadata:             map[string]string{"key": "val"},
				Adjustments:          adj,
			}
			createResponse, err := app.GetValue().SDK().Payments.V3.CreatePayment(ctx, createRequest)
			Expect(err).To(BeNil())

			MustEventuallyOutbox(ctx, app.GetValue(), models.OUTBOX_EVENT_PAYMENT_SAVED)

			getResponse, err := app.GetValue().SDK().Payments.V3.GetPayment(ctx, createResponse.GetV3CreatePaymentResponse().Data.ID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetV3GetPaymentResponse().Data.Amount).To(Equal(big.NewInt(0).Sub(createRequest.Amount, adj[0].Amount)))
			Expect(getResponse.GetV3GetPaymentResponse().Data.Status).To(Equal(adj[0].Status))
			Expect(getResponse.GetV3GetPaymentResponse().Data.Adjustments).To(HaveLen(1))
		})
	})

	When("creating a new payment with v2", func() {
		var (
			connectorID   string
			createdAt     time.Time
			initialAmount *big.Int
			asset         string
			err           error
		)
		BeforeEach(func() {
			createdAt = time.Now()
			initialAmount = big.NewInt(1340)
			asset = "USD/2"

			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 2)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be ok", func() {
			debtorID, creditorID := setupDebtorAndCreditorV2Accounts(ctx, app.GetValue(), connectorID, createdAt)
			createRequest := components.PaymentRequest{
				Reference:            "ref",
				ConnectorID:          connectorID,
				CreatedAt:            createdAt,
				Amount:               initialAmount,
				Asset:                asset,
				Type:                 "PAY-IN",
				Status:               "SUCCEEDED",
				Scheme:               components.PaymentSchemeAmex,
				SourceAccountID:      &debtorID,
				DestinationAccountID: &creditorID,
			}

			createResponse, err := app.GetValue().SDK().Payments.V1.CreatePayment(ctx, createRequest)
			Expect(err).To(BeNil())
			Expect(createResponse.GetPaymentResponse().Data.ID).NotTo(Equal(""))

			MustEventuallyOutbox(ctx, app.GetValue(), models.OUTBOX_EVENT_PAYMENT_SAVED)

			getResponse, err := app.GetValue().SDK().Payments.V1.GetPayment(ctx, createResponse.GetPaymentResponse().Data.ID)
			Expect(err).To(BeNil())
			Expect(getResponse.GetPaymentResponse().Data.Amount).To(Equal(createRequest.Amount))
			Expect(getResponse.GetPaymentResponse().Data.Status).To(Equal(createRequest.Status))
		})
	})
})

func setupDebtorAndCreditorV3Accounts(
	ctx context.Context,
	app *testserver.Server,
	connectorID string,
	createdAt time.Time,
) (string, string) {
	creditorID, err := createV3Account(ctx, app, &components.V3CreateAccountRequest{
		Reference:    "creditor",
		ConnectorID:  connectorID,
		CreatedAt:    createdAt.Add(-time.Hour),
		AccountName:  "creditor",
		Type:         "INTERNAL",
		DefaultAsset: pointer.For("USD/2"),
		Metadata: map[string]string{
			"key": "val",
		},
	})
	Expect(err).To(BeNil())
	MustEventuallyOutbox(ctx, app, models.OUTBOX_EVENT_ACCOUNT_SAVED)

	debtorID, err := createV3Account(ctx, app, &components.V3CreateAccountRequest{
		Reference:    "debtor",
		ConnectorID:  connectorID,
		CreatedAt:    createdAt,
		AccountName:  "debtor",
		Type:         "EXTERNAL",
		DefaultAsset: pointer.For("USD/2"),
		Metadata: map[string]string{
			"ping": "pong",
		},
	})
	Expect(err).To(BeNil())
	MustEventuallyOutbox(ctx, app, models.OUTBOX_EVENT_ACCOUNT_SAVED)

	return debtorID, creditorID
}

func setupDebtorAndCreditorV2Accounts(
	ctx context.Context,
	app *testserver.Server,
	connectorID string,
	createdAt time.Time,
) (string, string) {
	creditorID, err := createV2Account(ctx, app, components.AccountRequest{
		Reference:    "creditor",
		ConnectorID:  connectorID,
		CreatedAt:    createdAt.Add(-time.Hour),
		AccountName:  pointer.For("creditor"),
		Type:         "INTERNAL",
		DefaultAsset: pointer.For("USD/2"),
		Metadata: map[string]string{
			"key": "val",
		},
	})
	Expect(err).To(BeNil())
	MustEventuallyOutbox(ctx, app, models.OUTBOX_EVENT_ACCOUNT_SAVED)

	debtorID, err := createV2Account(ctx, app, components.AccountRequest{
		Reference:    "debtor",
		ConnectorID:  connectorID,
		CreatedAt:    createdAt,
		AccountName:  pointer.For("debtor"),
		Type:         "EXTERNAL",
		DefaultAsset: pointer.For("USD/2"),
		Metadata: map[string]string{
			"ping": "pong",
		},
	})
	Expect(err).To(BeNil())
	MustEventuallyOutbox(ctx, app, models.OUTBOX_EVENT_ACCOUNT_SAVED)

	return debtorID, creditorID
}
