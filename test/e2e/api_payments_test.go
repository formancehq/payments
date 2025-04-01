package test_suite

import (
	"context"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/formancehq/payments/pkg/testserver"
	. "github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Payments", func() {
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

	When("creating a new payment with v3", func() {
		var (
			connectorRes   struct{ Data string }
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

			connectorConf := newConnectorConfigurationFn()(uuid.New())
			err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
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

			debtorID, creditorID := setupDebtorAndCreditorAccounts(ctx, app.GetValue(), e, ver, connectorRes.Data, createdAt)
			createRequest := v3.CreatePaymentRequest{
				Reference:            "ref",
				ConnectorID:          connectorRes.Data,
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

			Eventually(e).Should(Receive(Event(evts.V2EventTypeSavedPayments)))

			err = GetPayment(ctx, app.GetValue(), ver, createResponse.Data.ID.String(), &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.Amount).To(Equal(big.NewInt(0).Sub(createRequest.Amount, adj[0].Amount)))
			Expect(getResponse.Data.Status.String()).To(Equal(adj[0].Status))
			Expect(getResponse.Data.Adjustments).To(HaveLen(1))
		})
	})

	When("creating a new payment with v2", func() {
		var (
			connectorRes   struct{ Data v2.ConnectorInstallResponse }
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

			connectorConf := newConnectorConfigurationFn()(uuid.New())
			err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())
		})

		It("should be ok", func() {
			debtorID, creditorID := setupDebtorAndCreditorAccounts(ctx, app.GetValue(), e, ver, connectorRes.Data.ConnectorID, createdAt)
			createRequest := v2.CreatePaymentRequest{
				Reference:            "ref",
				ConnectorID:          connectorRes.Data.ConnectorID,
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

			Eventually(e).Should(Receive(Event(evts.V2EventTypeSavedPayments)))

			err = GetPayment(ctx, app.GetValue(), ver, createResponse.Data.ID, &getResponse)
			Expect(err).To(BeNil())
			Expect(getResponse.Data.Amount).To(Equal(createRequest.Amount))
			Expect(getResponse.Data.Status).To(Equal(createRequest.Status))
		})
	})
})

func setupDebtorAndCreditorAccounts(
	ctx context.Context,
	app *testserver.Server,
	e chan *nats.Msg,
	ver int,
	connectorID string,
	createdAt time.Time,
) (debtorID, creditorID string) {
	var (
		creditorRes struct{ Data models.Account }
		debtorRes   struct{ Data models.Account }
	)

	creditorRequest := v3.CreateAccountRequest{
		Reference:    "creditor",
		Name:         "creditor",
		ConnectorID:  connectorID,
		CreatedAt:    createdAt.Add(-time.Hour),
		DefaultAsset: "USD/2",
		Type:         string(models.ACCOUNT_TYPE_INTERNAL),
		Metadata:     map[string]string{"key": "val"},
	}
	err := CreateAccount(ctx, app, ver, creditorRequest, &creditorRes)
	Expect(err).To(BeNil())
	Eventually(e).Should(Receive(Event(evts.V2EventTypeSavedAccounts)))

	debtorRequest := v3.CreateAccountRequest{
		Reference:    "debtor",
		Name:         "debtor",
		ConnectorID:  connectorID,
		CreatedAt:    createdAt,
		DefaultAsset: "USD/2",
		Type:         string(models.ACCOUNT_TYPE_EXTERNAL),
		Metadata:     map[string]string{"ping": "pong"},
	}
	err = CreateAccount(ctx, app, ver, debtorRequest, &debtorRes)
	Expect(err).To(BeNil())
	Eventually(e).Should(Receive(Event(evts.V2EventTypeSavedAccounts)))

	return debtorRes.Data.ID.String(), creditorRes.Data.ID.String()
}
