//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"os"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/testing/docker"
	"github.com/formancehq/go-libs/v3/testing/platform/natstesting"
	"github.com/formancehq/go-libs/v3/testing/platform/pgtesting"
	"github.com/formancehq/go-libs/v3/testing/platform/temporaltesting"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	os.Setenv("PLUGIN_MAGIC_COOKIE", magicCookieVal)
	RunSpecs(t, "Test Suite")
}

var (
	dockerPool     = deferred.New[*docker.Pool]()
	pgServer       = deferred.New[*pgtesting.PostgresServer]()
	temporalServer = deferred.New[*temporaltesting.TemporalServer]()
	natsServer     = deferred.New[*natstesting.NatsServer]()
	debug          = os.Getenv("DEBUG") == "true"
	logger         = logging.NewDefaultLogger(GinkgoWriter, debug, false, false)
	stack          = "somestackval-abcd"
	magicCookieVal = "needed-for-plugin-to-work"

	DBTemplate = "dbtemplate"
)

type GenericEventPayload struct {
	ID string `json:"id"`
}

type ConnectorConf struct {
	Name          string `json:"name"`
	PollingPeriod string `json:"pollingPeriod"`
	PageSize      int    `json:"pageSize"`
	Directory     string `json:"directory"`
}

type ParallelExecutionContext struct {
	PostgresServer *pgtesting.PostgresServer
	NatsServer     *natstesting.NatsServer
	TemporalServer *temporaltesting.TemporalServer
}

var _ = SynchronizedBeforeSuite(func() []byte {
	deferred.RegisterRecoverHandler(GinkgoRecover)

	By("Initializing docker pool")
	dockerPool.SetValue(docker.NewPool(GinkgoT(), logger))

	pgServer.LoadAsync(func() (*pgtesting.PostgresServer, error) {
		By("Initializing postgres server")
		ret := pgtesting.CreatePostgresServer(
			GinkgoT(),
			dockerPool.GetValue(),
			pgtesting.WithPGStatsExtension(),
			pgtesting.WithPGCrypto(),
		)
		By("Postgres address: " + ret.GetDSN())

		templateDatabase := ret.NewDatabase(GinkgoT(), pgtesting.WithName(DBTemplate))

		bunDB, err := bunconnect.OpenSQLDB(context.Background(), templateDatabase.ConnectionOptions())
		Expect(err).To(BeNil())

		err = storage.Migrate(context.Background(), logging.Testing(), bunDB, "test")
		Expect(err).To(BeNil())
		Expect(bunDB.Close()).To(BeNil())

		return ret, nil
	})
	natsServer.LoadAsync(func() (*natstesting.NatsServer, error) {
		By("Initializing nats server")
		ret := natstesting.CreateServer(GinkgoT(), debug, logger)
		By("Nats address: " + ret.ClientURL())
		return ret, nil
	})

	temporalServer.LoadAsync(func() (*temporaltesting.TemporalServer, error) {
		By("Initializing temporal server")
		ret := temporaltesting.CreateTemporalServer(GinkgoT(), GinkgoWriter)
		return ret, nil
	})

	By("Waiting services alive")
	deferred.Wait(pgServer, natsServer, temporalServer)
	By("All services ready.")

	data, err := json.Marshal(ParallelExecutionContext{
		PostgresServer: pgServer.GetValue(),
		NatsServer:     natsServer.GetValue(),
		TemporalServer: temporalServer.GetValue(),
	})
	Expect(err).To(BeNil())

	return data
}, func(data []byte) {
	select {
	case <-pgServer.Done():
		// Process #1, setup is terminated
		return
	default:
	}
	pec := ParallelExecutionContext{}
	err := json.Unmarshal(data, &pec)
	Expect(err).To(BeNil())

	pgServer.SetValue(pec.PostgresServer)
	natsServer.SetValue(pec.NatsServer)
	temporalServer.SetValue(pec.TemporalServer)
})

func UseTemplatedDatabase() *deferred.Deferred[*pgtesting.Database] {
	return pgtesting.UsePostgresDatabase(pgServer, pgtesting.CreateWithTemplate(DBTemplate))
}
