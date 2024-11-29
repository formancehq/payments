//go:build it

package test_suite

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/google/uuid"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	"github.com/formancehq/payments/internal/models"
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
			workflowID   string
		)
		JustBeforeEach(func() {
			id = uuid.New()
		})

		It("should be ok with v3", func() {
			ver := 3
			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			cl := temporalServer.GetValue().DefaultClient()
			req := &workflowservice.ListOpenWorkflowExecutionsRequest{Namespace: temporalServer.GetValue().DefaultNamespace()}
			workflowRes, err := cl.ListOpenWorkflow(ctx, req)
			for _, info := range workflowRes.Executions {
				if strings.HasPrefix(info.Execution.WorkflowId, "run-tasks-") {
					workflowID = info.Execution.WorkflowId
					break
				}
			}
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorRes.Data)))

			getRes := struct{ Data ConnectorConf }{}
			err = ConnectorConfig(ctx, app.GetValue(), ver, connectorRes.Data, &getRes)
			Expect(err).To(BeNil())
			Expect(getRes.Data).To(Equal(connectorConf))
		})

		It("should be ok with v2", func() {
			ver := 2
			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			getRes := struct{ Data ConnectorConf }{}
			err = ConnectorConfig(ctx, app.GetValue(), ver, connectorRes.Data, &getRes)
			Expect(err).To(BeNil())
			Expect(getRes.Data).To(Equal(connectorConf))
		})
	})

	When("uninstalling a connector", func() {
		var (
			connectorRes struct{ Data string }
			id           uuid.UUID
		)
		JustBeforeEach(func() {
			id = uuid.New()
		})

		It("should be ok with v3", func() {
			ver := 3
			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			delRes := struct{ Data string }{}
			err = ConnectorUninstall(ctx, app.GetValue(), ver, connectorRes.Data, &delRes)
			Expect(err).To(BeNil())
			Expect(delRes.Data).To(Equal(connectorRes.Data))
			blockTillWorkflowComplete(ctx, "uninstall")
		})

		It("should be ok with v2", func() {
			ver := 2
			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			err = ConnectorUninstall(ctx, app.GetValue(), ver, connectorRes.Data, nil)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, "uninstall")
		})
	})

	When("searching for schedules for a connector", func() {
		var (
			connectorRes  struct{ Data string }
			id            uuid.UUID
			ver           int
			expectedTypes = map[string]struct{}{
				"FetchAccounts":         {},
				"FetchExternalAccounts": {},
				"FetchPayments":         {},
			}
		)
		JustBeforeEach(func() {
			ver = 3
			id = uuid.New()

			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			workflowID := blockTillWorkflowComplete(ctx, "run-tasks-")
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorRes.Data)))
		})

		It("should be ok with v3", func(ctx SpecContext) {
			schCl := temporalServer.GetValue().DefaultClient().ScheduleClient()
			list, err := schCl.List(ctx, client.ScheduleListOptions{PageSize: 1})
			Expect(err).To(BeNil())
			Expect(list.HasNext()).To(BeTrue())

			for list.HasNext() {
				schedule, err := list.Next()
				Expect(err).To(BeNil())
				_, ok := expectedTypes[schedule.WorkflowType.Name]
				Expect(ok).To(BeTrue())
			}

			res := struct {
				Cursor bunpaginate.Cursor[models.Schedule]
			}{}
			err = ConnectorSchedules(ctx, app.GetValue(), ver, connectorRes.Data, &res)
			Expect(err).To(BeNil())
			Expect(len(res.Cursor.Data) > 0).To(BeTrue())
			for _, schedule := range res.Cursor.Data {
				Expect(schedule.ConnectorID.String()).To(Equal(connectorRes.Data))
				Expect(schedule.ConnectorID.Provider).To(Equal("generic"))
			}
		})
	})
})

func blockTillWorkflowComplete(ctx context.Context, searchKeyword string) string {
	var (
		workflowID string
		runID      string
	)

	cl := temporalServer.GetValue().DefaultClient()
	req := &workflowservice.ListOpenWorkflowExecutionsRequest{Namespace: temporalServer.GetValue().DefaultNamespace()}
	workflowRes, err := cl.ListOpenWorkflow(ctx, req)
	for _, info := range workflowRes.Executions {
		if strings.HasPrefix(info.Execution.WorkflowId, searchKeyword) {
			workflowID = info.Execution.WorkflowId
			runID = info.Execution.RunId
			break
		}
	}

	// if we couldn't find it either it's already done or it wasn't scheduled
	if workflowID == "" {
		return ""
	}
	workflowRun := cl.GetWorkflow(ctx, workflowID, runID)
	err = workflowRun.Get(ctx, nil) // blocks to ensure workflow is finished
	Expect(err).To(BeNil())
	return workflowID
}

func newConnectorConfigurationFn() func(id uuid.UUID) ConnectorConf {
	return func(id uuid.UUID) ConnectorConf {
		pspServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `[]`)
		}))
		GinkgoT().Cleanup(func() {
			pspServer.Close()
		})
		return ConnectorConf{
			Name:          fmt.Sprintf("connector-%s", id.String()),
			PollingPeriod: "2m",
			PageSize:      30,
			APIKey:        "key",
			Endpoint:      pspServer.URL,
		}
	}
}
