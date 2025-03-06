//go:build it

package test_suite

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
			Debug:                 true,
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

		DescribeTable("should respond with a validation error when plugin-side config invalid",
			func(ver int, dirVal string, expectedErr string) {
				connectorConf := newConnectorConfigurationFn()(id)
				connectorConf.Directory = dirVal
				err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, nil)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("400"))
				Expect(err.Error()).To(ContainSubstring(expectedErr))
			},
			Entry("empty directory with v2", 2, "", "validation for 'Directory' failed on the 'required' tag"),
			Entry("empty directory with v3", 3, "", "validation for 'Directory' failed on the 'required' tag"),
			Entry("invalid directory with v2", 2, "^&()sodj", "validation for 'Directory' failed on the 'dirpath' tag"),
			Entry("invalid directory with v3", 3, "^&()sodj", "validation for 'Directory' failed on the 'dirpath' tag"),
		)
	})

	When("updating a connector config", func() {
		var (
			id           uuid.UUID
			ver          int
			connectorRes struct{ Data string }
			connectorID  string
		)
		JustBeforeEach(func() {
			id = uuid.New()
			ver = 3

			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())
			connectorID = connectorRes.Data
			blockTillWorkflowComplete(ctx, connectorID, "run-tasks-")
		})

		It("should be ok with v3", func() {
			config := newConnectorConfigurationFn()(id)
			config.PollingPeriod = "2m"
			err := ConnectorConfigUpdate(ctx, app.GetValue(), ver, connectorID, &config)
			Expect(err).To(BeNil())

			getRes := struct{ Data ConnectorConf }{}
			err = ConnectorConfig(ctx, app.GetValue(), ver, connectorID, &getRes)
			Expect(err).To(BeNil())
			Expect(getRes.Data).To(Equal(config))
		})

		DescribeTable("should respond with a validation error when plugin-side config invalid",
			func(ver int, dirValue string, expectedErr string) {
				config := newConnectorConfigurationFn()(id)
				config.Directory = dirValue
				err := ConnectorConfigUpdate(ctx, app.GetValue(), ver, connectorID, &config)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("400"))
				Expect(err.Error()).To(ContainSubstring(expectedErr))
			},
			Entry("empty directory", 3, "", "validation for 'Directory' failed on the 'required' tag"),
			Entry("invalid directory", 3, "$#2djskajdj", "validation for 'Directory' failed on the 'dirpath' tag"),
		)
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
			Eventually(taskPoller(delRes.Data.TaskID)).WithTimeout(models.DefaultConnectorClientTimeout * 2).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))
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

	When("resetting a connector", func() {
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

			resetRes := struct {
				Data v3.ConnectorResetResponse `json:"data"`
			}{}
			err = ConnectorReset(ctx, app.GetValue(), ver, connectorRes.Data, &resetRes)
			Expect(err).To(BeNil())
			Expect(resetRes.Data).NotTo(BeNil())
			taskID, err := models.TaskIDFromString(resetRes.Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("reset"))
			taskPoller := TaskPoller(ctx, GinkgoT(), app.GetValue())
			blockTillWorkflowComplete(ctx, connectorRes.Data, "reset")
			Eventually(taskPoller(resetRes.Data.TaskID)).WithTimeout(models.DefaultConnectorClientTimeout * 2).Should(HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))
		})

		It("should be ok with v2", func() {
			ver := 2
			var connectorRes struct{ Data v2.ConnectorInstallResponse }
			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			err = ConnectorReset(ctx, app.GetValue(), ver, connectorRes.Data.ConnectorID, nil)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, connectorRes.Data.ConnectorID, "reset")
		})
	})

	When("fetching a single schedule for a connector", func() {
		var (
			connectorRes struct{ Data string }
			id           uuid.UUID
			ver          int

			expectedSchedule models.Schedule
		)
		JustBeforeEach(func() {
			ver = 3
			id = uuid.New()

			connectorConf := newConnectorConfigurationFn()(id)
			err := ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
			Expect(err).To(BeNil())

			workflowID := blockTillWorkflowComplete(ctx, connectorRes.Data, "run-tasks-")
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorRes.Data)))

			listRes := struct {
				Cursor bunpaginate.Cursor[models.Schedule]
			}{}
			err = ConnectorSchedules(ctx, app.GetValue(), ver, connectorRes.Data, &listRes)
			Expect(err).To(BeNil())
			Expect(len(listRes.Cursor.Data) > 0).To(BeTrue())
			expectedSchedule = listRes.Cursor.Data[0]
		})

		It("should be ok with v3", func(ctx SpecContext) {
			res := struct {
				Data models.Schedule
			}{}
			err := GetConnectorSchedule(ctx, app.GetValue(), ver, connectorRes.Data, expectedSchedule.ID, &res)
			Expect(err).To(BeNil())
			schedule := res.Data
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.ConnectorID.String()).To(Equal(connectorRes.Data))
			Expect(schedule.ConnectorID.Provider).To(Equal("dummypay"))
			Expect(schedule.ID).To(Equal(expectedSchedule.ID))
			Expect(schedule.CreatedAt).To(Equal(expectedSchedule.CreatedAt))
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

	When("fetching connector configurations", func() {
		type connectorDef struct {
			DataType     string `json:"dataType"`
			Required     bool   `json:"required"`
			DefaultValue string `json:"defaultValue"`
		}
		var res struct {
			Data map[string]map[string]connectorDef
		}

		DescribeTable("should respond with detailed config json for each connector",
			func(ver int) {
				err := ConnectorConfigs(ctx, app.GetValue(), ver, &res)
				Expect(err).To(BeNil())
				Expect(len(res.Data)).To(BeNumerically(">", 1))
				Expect(res.Data["dummypay"]).NotTo(BeNil())
				Expect(res.Data["dummypay"]["pageSize"]).NotTo(BeNil())
				Expect(res.Data["dummypay"]["pageSize"].DataType).To(Equal("unsigned integer"))
				Expect(res.Data["dummypay"]["pageSize"].Required).To(Equal(false))
				Expect(res.Data["dummypay"]["pageSize"].DefaultValue).NotTo(Equal(""))
				pageSize, err := strconv.Atoi(res.Data["dummypay"]["pageSize"].DefaultValue)
				Expect(err).To(BeNil())
				Expect(pageSize).To(BeNumerically(">", 0))
				Expect(res.Data["dummypay"]["pollingPeriod"]).NotTo(BeNil())
				Expect(res.Data["dummypay"]["pollingPeriod"].DataType).To(Equal("duration ns"))
				Expect(res.Data["dummypay"]["pollingPeriod"].Required).To(Equal(false))
				Expect(res.Data["dummypay"]["pollingPeriod"].DefaultValue).NotTo(Equal(""))
				pollingPeriod, err := time.ParseDuration(res.Data["dummypay"]["pollingPeriod"].DefaultValue)
				Expect(err).To(BeNil())
				Expect(pollingPeriod).To(BeNumerically(">", 0))
				Expect(res.Data["dummypay"]["name"]).NotTo(BeNil())
				Expect(res.Data["dummypay"]["name"].DataType).To(Equal("string"))
				Expect(res.Data["dummypay"]["name"].Required).To(Equal(true))
				Expect(res.Data["dummypay"]["name"].DefaultValue).To(Equal(""))
				Expect(res.Data["dummypay"]["directory"]).NotTo(BeNil())
				Expect(res.Data["dummypay"]["directory"].DataType).To(Equal("string"))
				Expect(res.Data["dummypay"]["directory"].Required).To(Equal(true))
				Expect(res.Data["dummypay"]["directory"].DefaultValue).To(Equal(""))
			},
			Entry("with v2", 2),
			Entry("with v3", 3),
		)
	})
})

func blockTillWorkflowComplete(ctx context.Context, connectorIDStr string, searchKeyword string) string {
	var (
		workflowID string
		runID      string
	)

	connectorID := models.MustConnectorIDFromString(connectorIDStr)

	cl := temporalServer.GetValue().DefaultClient()
	req := &workflowservice.ListOpenWorkflowExecutionsRequest{Namespace: temporalServer.GetValue().DefaultNamespace()}
	workflowRes, err := cl.ListOpenWorkflow(ctx, req)
	for _, info := range workflowRes.Executions {
		if (strings.Contains(info.Execution.WorkflowId, connectorID.Reference.String()) ||
			strings.Contains(info.Execution.WorkflowId, connectorID.String())) &&
			strings.HasPrefix(info.Execution.WorkflowId, searchKeyword) {
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
