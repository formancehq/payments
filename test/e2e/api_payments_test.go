package test_suite

import (
	"context"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Payments", func() {
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

	When("creating a new payment with v3", func() {
		var (
			connectorID    string
			createResponse struct{ Data models.Payment }
			getResponse    struct{ Data models.Payment }
			e              chan *nats.Msg
			ver            int
			createdAt      time.Time
			initialAmount  *big.Int
			asset          string
			err            error
		)
		JustBeforeEach(func() {
			ver = 3
			createdAt = time.Now()
			initialAmount = big.NewInt(1340)
			asset = "USD/2"
			e = Subscribe(GinkgoT(), app.GetValue())

			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		It("should be ok", func() {
			adj := []v3.CreatePaymentsAdjustmentsRequest{
				{
					Reference: "ref_adjustment",
					CreatedAt: createdAt,
					Amount:    big.NewInt(55),
					Asset:     &asset,
					Status:    models.PAYMENT_STATUS_REFUNDED.String(),
				},
			}

			debtorID, creditorID := setupDebtorAndCreditorV3Accounts(ctx, app.GetValue(), e, connectorID, createdAt)
			createRequest := v3.CreatePaymentRequest{
				Reference:            "ref",
				ConnectorID:          connectorID,
				CreatedAt:            createdAt,
				InitialAmount:        initialAmount,
				Amount:               initialAmount,
				Asset:                asset,
				Type:                 models.PAYMENT_TYPE_PAYIN.String(),
				SourceAccountID:      &debtorID,
				DestinationAccountID: &creditorID,
				Scheme:               models.PAYMENT_SCHEME_CARD_AMEX.String(),
				Adjustments:          adj,
				Metadata:             map[string]string{"key": "val"},
			}

			err = CreatePayment(ctx, app.GetValue(), ver, createRequest, &createResponse)
			Expect(err).To(BeNil())

			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPayments)))

			err = GetPayment(ctx, app.GetValue(), ver, createResponse.Data.ID.String(), &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.Amount).To(Equal(big.NewInt(0).Sub(createRequest.Amount, adj[0].Amount)))
			Expect(getResponse.Data.Status.String()).To(Equal(adj[0].Status))
			Expect(getResponse.Data.Adjustments).To(HaveLen(1))
		})
	})

	When("creating a new payment with v2", func() {
		var (
			connectorID    string
			createResponse struct{ Data v2.PaymentResponse }
			getResponse    struct{ Data v2.PaymentResponse }
			e              chan *nats.Msg
			ver            int
			createdAt      time.Time
			initialAmount  *big.Int
			asset          string
			err            error
		)
		JustBeforeEach(func() {
			ver = 2
			createdAt = time.Now()
			initialAmount = big.NewInt(1340)
			asset = "USD/2"
			e = Subscribe(GinkgoT(), app.GetValue())

			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 2)
			Expect(err).To(BeNil())
		})

		It("should be ok", func() {
			debtorID, creditorID := setupDebtorAndCreditorV2Accounts(ctx, app.GetValue(), e, connectorID, createdAt)
			createRequest := v2.CreatePaymentRequest{
				Reference:            "ref",
				ConnectorID:          connectorID,
				CreatedAt:            createdAt,
				Amount:               initialAmount,
				Asset:                asset,
				Type:                 models.PAYMENT_TYPE_PAYIN.String(),
				Status:               models.PAYMENT_STATUS_SUCCEEDED.String(),
				SourceAccountID:      &debtorID,
				DestinationAccountID: &creditorID,
				Scheme:               models.PAYMENT_SCHEME_CARD_AMEX.String(),
				Metadata:             map[string]string{"key": "val"},
			}

			err = CreatePayment(ctx, app.GetValue(), ver, createRequest, &createResponse)
			Expect(err).To(BeNil())
			Expect(createResponse.Data.ID).NotTo(Equal(""))

			Eventually(e).Should(Receive(Event(evts.EventTypeSavedPayments)))

			err = GetPayment(ctx, app.GetValue(), ver, createResponse.Data.ID, &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.Amount).To(Equal(createRequest.Amount))
			Expect(getResponse.Data.Status).To(Equal(createRequest.Status))
		})
	})
})

func setupDebtorAndCreditorV3Accounts(
	ctx context.Context,
	app *testserver.Server,
	e chan *nats.Msg,
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
	Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))

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
	Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))

	return debtorID, creditorID
}

func setupDebtorAndCreditorV2Accounts(
	ctx context.Context,
	app *testserver.Server,
	e chan *nats.Msg,
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
	Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))

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
	Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))

	return debtorID, creditorID
}
