//go:build it

package test_suite

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/formancehq/go-libs/v3/testing/docker"
	"github.com/formancehq/go-libs/v3/testing/platform/natstesting"
	"github.com/formancehq/go-libs/v3/testing/platform/pgtesting"
	"github.com/formancehq/go-libs/v3/testing/platform/temporaltesting"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/api/serviceerror"
	v17 "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

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
	//bunDB             *bun.DB
	//currentDBDeferred *deferred.Deferred[*pgtesting.Database]

	DBTemplate = "dbtemplate"
)

type GenericEventPayload struct {
	ID string `json:"id"`
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

		migrateDB, err := bunconnect.OpenSQLDB(context.Background(), templateDatabase.ConnectionOptions())
		Expect(err).To(BeNil())

		err = storage.Migrate(context.Background(), logging.Testing(), migrateDB, "test")
		Expect(err).To(BeNil())
		Expect(migrateDB.Close()).To(BeNil())

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

func flushRemainingWorkflows(ctx context.Context) {
	cl := temporalServer.GetValue().DefaultClient()
	maxPageSize := 25

	mu := &sync.Mutex{}
	errs := make([]error, 0)

	iterateThroughTemporalWorkflowExecutions(ctx, cl, int32(maxPageSize), func(info *v17.WorkflowExecutionInfo) bool {
		err := cl.TerminateWorkflow(ctx, info.Execution.WorkflowId, info.Execution.RunId, "system flush")
		if err != nil {
			mu.Lock()
			errs = append(errs, err)
			mu.Unlock()
		}
		return false
	})

	for _, err := range errs {
		// might already be completed
		var notFoundErr *serviceerror.NotFound
		if errors.As(err, &notFoundErr) {
			continue
		}
		Expect(err).To(BeNil())
	}
}

// pages through all workflow executions until the callback function returns true
func iterateThroughTemporalWorkflowExecutions(
	ctx context.Context,
	cl client.Client,
	maxPageSize int32,
	callbackFn func(info *v17.WorkflowExecutionInfo) bool,
) {
	namespace := temporalServer.GetValue().DefaultNamespace()
	var nextPageToken []byte

PAGES:
	for {
		req := &workflowservice.ListOpenWorkflowExecutionsRequest{
			Namespace:       namespace,
			NextPageToken:   nextPageToken,
			MaximumPageSize: maxPageSize,
		}
		workflowRes, err := cl.ListOpenWorkflow(ctx, req)
		Expect(err).To(BeNil())

		ch := make(chan bool, int(maxPageSize))
		wg := &sync.WaitGroup{}
		for _, info := range workflowRes.Executions {
			wg.Add(1)
			go func(in *v17.WorkflowExecutionInfo) {
				defer wg.Done()
				ch <- callbackFn(in)
			}(info)
		}

		// wait for this batch of goroutines to finish before allowing the loop to continue
		wg.Wait()
		close(ch)

		for shouldStop := range ch {
			if shouldStop {
				break PAGES
			}
		}

		if len(workflowRes.NextPageToken) == 0 {
			break
		}
		nextPageToken = workflowRes.NextPageToken
	}
}
