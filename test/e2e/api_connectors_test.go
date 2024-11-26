//go:build it

package test_suite

import (
	"fmt"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Connectors", func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()
	)

	app := NewTestServer(func() Configuration {
		return Configuration{
			Stack:                 stack,
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	When("installing a connector", func() {
		var (
			connectorRes struct{ Data string }
			id           uuid.UUID
		)
		JustBeforeEach(func() {
			id = uuid.New()
		})

		It("should be ok with v3", func() {
			ver := 3
			connectorConf := ConnectorConf{
				Name:          fmt.Sprintf("connector-%s", id.String()),
				PollingPeriod: "2m",
				PageSize:      30,
				APIKey:        "key",
				Endpoint:      "http://example.com",
			}
			err := InstallConnector(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			getRes := struct{ Data ConnectorConf }{}
			err = ConnectorConfig(ctx, app.GetValue(), ver, connectorRes.Data, &getRes)
			Expect(err).To(BeNil())
			Expect(getRes.Data).To(Equal(connectorConf))
		})

		It("should be ok with v2", func() {
			ver := 2
			connectorConf := ConnectorConf{
				Name:          fmt.Sprintf("connector-%s", id.String()),
				PollingPeriod: "2m",
				PageSize:      30,
				APIKey:        "key",
				Endpoint:      "http://example.com",
			}
			err := InstallConnector(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			getRes := struct{ Data ConnectorConf }{}
			err = ConnectorConfig(ctx, app.GetValue(), ver, connectorRes.Data, &getRes)
			Expect(err).To(BeNil())
			Expect(getRes.Data).To(Equal(connectorConf))
		})
	})
})
