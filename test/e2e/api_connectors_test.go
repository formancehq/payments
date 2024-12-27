//go:build it

package test_suite

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	. "github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
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
			id         uuid.UUID
			workflowID string
		)
		JustBeforeEach(func() {
			id = uuid.New()
		})

		It("should be ok with v3", func() {
			ver := 3
			var connectorRes struct{ Data string }
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
			var connectorRes struct{ Data v2.ConnectorInstallResponse }
			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			getRes := struct{ Data ConnectorConf }{}
			err = ConnectorConfig(ctx, app.GetValue(), ver, connectorRes.Data.ConnectorID, &getRes)
			Expect(err).To(BeNil())
			Expect(getRes.Data).To(Equal(connectorConf))
		})
	})

	When("uninstalling a connector", func() {
		var (
			id uuid.UUID
		)
		JustBeforeEach(func() {
			id = uuid.New()
		})

		It("should be ok with v3", func() {
			ver := 3
			var connectorRes struct{ Data string }
			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			delRes := struct {
				Data v3.ConnectorUninstallResponse `json:"data"`
			}{}
			err = ConnectorUninstall(ctx, app.GetValue(), ver, connectorRes.Data, &delRes)
			Expect(err).To(BeNil())
			Expect(delRes.Data).NotTo(BeNil())
			taskID, err := models.TaskIDFromString(delRes.Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("uninstall"))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			blockTillWorkflowComplete(ctx, connectorRes.Data, "uninstall")
			Eventually(taskPoller(delRes.Data.TaskID)).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))
		})

		It("should be ok with v2", func() {
			ver := 2
			var connectorRes struct{ Data v2.ConnectorInstallResponse }
			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			err = ConnectorUninstall(ctx, app.GetValue(), ver, connectorRes.Data.ConnectorID, nil)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, connectorRes.Data.ConnectorID, "uninstall")
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
				"FetchBalances":         {},
			}
		)
		JustBeforeEach(func() {
			ver = 3
			id = uuid.New()

			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			workflowID := blockTillWorkflowComplete(ctx, connectorRes.Data, "run-tasks-")
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorRes.Data)))
		})

		It("should be ok with v3", func(ctx SpecContext) {
			schCl := temporalServer.GetValue().DefaultClient().ScheduleClient()
			list, err := schCl.List(ctx, client.ScheduleListOptions{PageSize: 1})
			Expect(err).To(BeNil())
			Expect(list.HasNext()).To(BeTrue())

			for list.HasNext() {
				schedule, err := list.Next()
				if !strings.Contains(schedule.ID, connectorRes.Data) {
					continue
				}
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
				Expect(schedule.ConnectorID.Provider).To(Equal("dummypay"))
			}
		})
	})
})

func blockTillWorkflowComplete(ctx context.Context, connectorID string, searchKeyword string) string {
	var (
		workflowID string
		runID      string
	)

	cl := temporalServer.GetValue().DefaultClient()
	req := &workflowservice.ListOpenWorkflowExecutionsRequest{Namespace: temporalServer.GetValue().DefaultNamespace()}
	workflowRes, err := cl.ListOpenWorkflow(ctx, req)
	for _, info := range workflowRes.Executions {
		if strings.Contains(info.Execution.WorkflowId, connectorID) && strings.HasPrefix(info.Execution.WorkflowId, searchKeyword) {
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
		dir, err := os.MkdirTemp("", "dummypay")
		Expect(err).To(BeNil())
		GinkgoT().Cleanup(func() {
			os.RemoveAll(dir)
		})

		return ConnectorConf{
			Name:          fmt.Sprintf("connector-%s", id.String()),
			PollingPeriod: "30s",
			PageSize:      30,
			Directory:     dir,
		}
	}
}
