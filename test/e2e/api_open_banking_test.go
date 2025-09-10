package test_suite

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/client/models/operations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Open Banking", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		v3CreateRequest *components.V3CreatePaymentServiceUserRequest

		app *deferred.Deferred[*Server]

		psuID string
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

	v3CreateRequest = &components.V3CreatePaymentServiceUserRequest{
		Name: "test",
		ContactDetails: &components.V3ContactDetailsRequest{
			Email:       pointer.For("dev@formance.com"),
			PhoneNumber: pointer.For("+33612131415"),
		},
		Address: &components.V3AddressRequest{
			StreetNumber: pointer.For("1"),
			StreetName:   pointer.For("test"),
			City:         pointer.For("test"),
			Region:       pointer.For("test"),
			PostalCode:   pointer.For("test"),
			Country:      pointer.For("FR"),
		},
		BankAccountIDs: []string{},
		Metadata:       map[string]string{},
	}

	BeforeEach(func() {
		createResponse, err := app.GetValue().SDK().Payments.V3.CreatePaymentServiceUser(ctx, v3CreateRequest)
		Expect(err).To(BeNil())
		psuID = createResponse.GetV3CreatePaymentServiceUserResponse().Data
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("forwarding a psu to a connector", func() {
		var (
			connectorID string
		)

		BeforeEach(func() {
			var err error

			id := uuid.New()
			conf := newV3ConnectorConfigFn()(id)
			conf.LinkFlowError = pointer.For(false)
			conf.UpdateLinkFlowError = pointer.For(false)
			connectorID, err = installV3Connector(ctx, app.GetValue(), conf, id)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be ok", func() {
			forwardResponse, err := app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserToProvider(
				ctx,
				psuID,
				connectorID,
			)
			Expect(err).To(BeNil())
			Expect(forwardResponse.GetHTTPMeta().Response.StatusCode).To(Equal(http.StatusNoContent))
		})

		It("should fail if psu is already forwarded to the connector", func() {
			forwardResponse, err := app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserToProvider(
				ctx,
				psuID,
				connectorID,
			)
			Expect(err).To(BeNil())
			Expect(forwardResponse.GetHTTPMeta().Response.StatusCode).To(Equal(http.StatusNoContent))

			_, err = app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserToProvider(
				ctx,
				psuID,
				connectorID,
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("user already exists on this connector"))
		})

		It("should fail if connector id does not exists", func() {
			fakeConnectorID := models.ConnectorID{
				Reference: uuid.New(),
				Provider:  "fake",
			}
			_, err := app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserToProvider(
				ctx,
				psuID,
				fakeConnectorID.String(),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("connector not found"))
		})
	})

	When("creating a link and call it - success", func() {
		var (
			connectorID string
			httpClient  httpwrapper.Client
		)

		BeforeEach(func() {
			httpClient = httpwrapper.NewClient(&httpwrapper.Config{})

			var err error

			id := uuid.New()
			conf := newV3ConnectorConfigFn()(id)
			conf.LinkFlowError = pointer.For(false)
			conf.UpdateLinkFlowError = pointer.For(false)
			connectorID, err = installV3Connector(ctx, app.GetValue(), conf, id)
			Expect(err).To(BeNil())

			forwardResponse, err := app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserToProvider(
				ctx,
				psuID,
				connectorID,
			)
			Expect(err).To(BeNil())
			Expect(forwardResponse.GetHTTPMeta().Response.StatusCode).To(Equal(http.StatusNoContent))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("should be ok", func() {
			resp, err := app.GetValue().SDK().Payments.V3.CreateLinkForPaymentServiceUser(
				ctx,
				psuID,
				connectorID,
				&components.V3PaymentServiceUserCreateLinkRequest{
					ApplicationName:   "test",
					ClientRedirectURL: "https://www.google.com",
				},
			)
			Expect(err).To(BeNil())
			Expect(resp.GetV3PaymentServiceUserCreateLinkResponse().GetAttemptID()).To(Not(BeEmpty()))
			Expect(resp.GetV3PaymentServiceUserCreateLinkResponse().GetLink()).To(Not(BeEmpty()))

			link, err := url.Parse(resp.GetV3PaymentServiceUserCreateLinkResponse().GetLink())
			Expect(err).To(BeNil())

			appUrl, err := url.Parse(app.GetValue().URL())
			Expect(err).To(BeNil())

			// Here, we don't care about the link sent back from dummypay, we
			// just want to validate that the redirect endpoint is doing its job
			// when called, so we can actually replace the host and path to
			// avoid creating a gateway inside the test suite.
			link.Scheme = appUrl.Scheme
			link.Host = appUrl.Host
			link.Path = strings.TrimPrefix(link.Path, "api/payments/")

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, link.String(), nil)
			Expect(err).To(BeNil())

			statusCode, err := httpClient.Do(ctx, req, nil, nil)
			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(http.StatusNoContent))

			attemptPoller := pollAttempts(ctx, app, psuID, connectorID, GinkgoT())
			Eventually(attemptPoller()).WithTimeout(10 * time.Second).Should(HaveLinkAttemptsLengthMatcher(1, []PayloadMatcher{HaveLinkAttemptStatus(components.V3PSUOpenBankingConnectionAttemptStatusEnumCompleted)}...))

			connectionPoller := pollConnection(ctx, app, psuID, GinkgoT())
			Eventually(connectionPoller()).WithTimeout(10 * time.Second).Should(HaveUserConnectionsLengthMatcher(1, []PayloadMatcher{HaveUserConnectionStatus(components.V3ConnectionStatusEnumActive)}...))

			// We should be able to delete the connection
			connectionID := connectionPoller()()[0].ConnectionID
			_, err = app.GetValue().SDK().Payments.V3.DeletePaymentServiceUserConnectionFromConnectorID(ctx, psuID, connectorID, connectionID)
			Expect(err).To(BeNil())
			Eventually(connectionPoller()).WithTimeout(10 * time.Second).Should(HaveUserConnectionsLengthMatcher(0))
		})
	})

	When("creating a link and call it - error", func() {
		var (
			connectorID string
			httpClient  httpwrapper.Client
		)

		BeforeEach(func() {
			httpClient = httpwrapper.NewClient(&httpwrapper.Config{})

			var err error

			id := uuid.New()
			conf := newV3ConnectorConfigFn()(id)
			conf.LinkFlowError = pointer.For(true)
			conf.UpdateLinkFlowError = pointer.For(false)
			connectorID, err = installV3Connector(ctx, app.GetValue(), conf, id)
			Expect(err).To(BeNil())

			forwardResponse, err := app.GetValue().SDK().Payments.V3.ForwardPaymentServiceUserToProvider(
				ctx,
				psuID,
				connectorID,
			)
			Expect(err).To(BeNil())
			Expect(forwardResponse.GetHTTPMeta().Response.StatusCode).To(Equal(http.StatusNoContent))
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		It("the link flow should be in error and the attempt should be updated to exited, no connection should be created", func() {
			resp, err := app.GetValue().SDK().Payments.V3.CreateLinkForPaymentServiceUser(
				ctx,
				psuID,
				connectorID,
				&components.V3PaymentServiceUserCreateLinkRequest{
					ApplicationName:   "test",
					ClientRedirectURL: "https://www.google.com",
				},
			)
			Expect(err).To(BeNil())
			Expect(resp.GetV3PaymentServiceUserCreateLinkResponse().GetAttemptID()).To(Not(BeEmpty()))
			Expect(resp.GetV3PaymentServiceUserCreateLinkResponse().GetLink()).To(Not(BeEmpty()))

			link, err := url.Parse(resp.GetV3PaymentServiceUserCreateLinkResponse().GetLink())
			Expect(err).To(BeNil())

			appUrl, err := url.Parse(app.GetValue().URL())
			Expect(err).To(BeNil())

			// Here, we don't care about the link sent back from dummypay, we
			// just want to validate that the redirect endpoint is doing its job
			// when called, so we can actually replace the host and path to
			// avoid creating a gateway inside the test suite.
			link.Scheme = appUrl.Scheme
			link.Host = appUrl.Host
			link.Path = strings.TrimPrefix(link.Path, "api/payments/")

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, link.String(), nil)
			Expect(err).To(BeNil())

			statusCode, err := httpClient.Do(ctx, req, nil, nil)
			Expect(err).To(BeNil())
			Expect(statusCode).To(Equal(http.StatusNoContent))

			attemptPoller := pollAttempts(ctx, app, psuID, connectorID, GinkgoT())
			Eventually(attemptPoller()).WithTimeout(10 * time.Second).Should(HaveLinkAttemptsLengthMatcher(1, []PayloadMatcher{HaveLinkAttemptStatus(components.V3PSUOpenBankingConnectionAttemptStatusEnumExited)}...))

			connectionPoller := pollConnection(ctx, app, psuID, GinkgoT())
			Eventually(connectionPoller()).WithTimeout(10 * time.Second).Should(HaveUserConnectionsLengthMatcher(0))
		})
	})
})

func pollAttempts(ctx context.Context, app *deferred.Deferred[*Server], psuID string, connectorID string, t T) func() func() []components.V3PaymentServiceUserLinkAttempt {
	return func() func() []components.V3PaymentServiceUserLinkAttempt {
		return func() []components.V3PaymentServiceUserLinkAttempt {
			attempts, err := app.GetValue().SDK().Payments.V3.ListPaymentServiceUserLinkAttemptsFromConnectorID(
				ctx,
				operations.V3ListPaymentServiceUserLinkAttemptsFromConnectorIDRequest{
					PaymentServiceUserID: psuID,
					ConnectorID:          connectorID,
				},
			)
			require.NoError(t, err)
			return attempts.V3PaymentServiceUserLinkAttemptsCursorResponse.Cursor.Data
		}
	}
}

func pollConnection(ctx context.Context, app *deferred.Deferred[*Server], psuID string, t T) func() func() []components.V3PaymentServiceUserConnection {
	return func() func() []components.V3PaymentServiceUserConnection {
		return func() []components.V3PaymentServiceUserConnection {
			connections, err := app.GetValue().SDK().Payments.V3.ListPaymentServiceUserConnections(
				ctx,
				psuID,
				nil,
				nil,
				nil,
			)
			require.NoError(t, err)
			return connections.V3PaymentServiceUserConnectionsCursorResponse.Cursor.Data
		}
	}
}
