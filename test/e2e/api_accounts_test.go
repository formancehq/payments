//go:build it

package test_suite

import (
	"time"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Accounts", func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		createRequest v3.CreateAccountRequest

		app *utils.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                 stack,
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	createdAt, _ := time.Parse("2006-Jan-02", "2024-Nov-29")
	createRequest = v3.CreateAccountRequest{
		Reference:    "ref",
		AccountName:  "foo",
		CreatedAt:    createdAt,
		DefaultAsset: "USD",
		Type:         string(models.ACCOUNT_TYPE_INTERNAL),
		Metadata:     map[string]string{"key": "val"},
	}

	When("creating a new account", func() {
		var (
			connectorRes   struct{ Data string }
			createResponse struct{ Data models.Account }
			getResponse    struct{ Data models.Account }
			err            error
		)

		DescribeTable("should be successful",
			func(ver int) {
				connectorConf := newConnectorConfigurationFn()(uuid.New())
				err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
				Expect(err).To(BeNil())

				createRequest.ConnectorID = connectorRes.Data
				err = CreateAccount(ctx, app.GetValue(), ver, createRequest, &createResponse)
				Expect(err).To(BeNil())

				err = GetAccount(ctx, app.GetValue(), ver, createResponse.Data.ID.String(), &getResponse)
				Expect(err).To(BeNil())
				Expect(getResponse.Data).To(Equal(createResponse.Data))
			},
			Entry("with v2", 2),
			Entry("with v3", 3),
		)
	})
})
