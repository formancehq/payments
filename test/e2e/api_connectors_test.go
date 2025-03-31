//go:build it

package test_suite

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/testserver"
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

	app := testserver.NewTestServer(func() testserver.Configuration {
		return testserver.Configuration{
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

		It("can install a connector with v3 API", func() {
			connectorID, connectorConf, err := installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())
			_, err = models.ConnectorIDFromString(connectorID)
			Expect(err).To(BeNil())

			cl := temporalServer.GetValue().DefaultClient()
			req := &workflowservice.ListOpenWorkflowExecutionsRequest{Namespace: temporalServer.GetValue().DefaultNamespace()}
			workflowRes, err := cl.ListOpenWorkflow(ctx, req)
			Expect(err).To(BeNil())
			for _, info := range workflowRes.Executions {
				if strings.HasPrefix(info.Execution.WorkflowId, "run-tasks-") {
					workflowID = info.Execution.WorkflowId
					break
				}
			}
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorID)))

			getRes, err := app.GetValue().SDK().Payments.V3.GetConnectorConfig(ctx, connectorID)
			Expect(err).To(BeNil())
			Expect(getRes.V3GetConnectorConfigResponse).NotTo(BeNil())
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig).To(Equal(connectorConf))
			Expect(getRes.V3GetConnectorConfigResponse.Data.Type).To(Equal(components.V3ConnectorConfigTypeDummypay))
		})

		It("can install a connector with v2 API", func() {
			dir, err := os.MkdirTemp("", "dummypay")
			Expect(err).To(BeNil())
			GinkgoT().Cleanup(func() {
				os.RemoveAll(dir)
			})

			connectorConf := components.ConnectorConfig{
				DummyPayConfig: &components.DummyPayConfig{
					Name:              fmt.Sprintf("connector-%s", id.String()),
					FilePollingPeriod: pointer.For("30s"),
					Provider:          pointer.For("Dummypay"),
					Directory:         dir,
				},
			}
			res, err := app.GetValue().SDK().Payments.V1.InstallConnector(ctx, components.ConnectorEnumDummyPay, connectorConf)
			Expect(err).To(BeNil())
			Expect(res.ConnectorResponse).NotTo(BeNil())
			connectorID := res.ConnectorResponse.Data.ConnectorID

			getRes, err := app.GetValue().SDK().Payments.V1.ReadConnectorConfigV1(ctx, components.ConnectorEnumDummyPay, connectorID)
			Expect(err).To(BeNil())
			Expect(getRes.ConnectorConfigResponse).NotTo(BeNil())
			Expect(getRes.ConnectorConfigResponse.Data.DummyPayConfig).To(Equal(connectorConf.DummyPayConfig))
			Expect(getRes.ConnectorConfigResponse.Data.Type).To(Equal(components.ConnectorConfigTypeDummypay))
		})

		DescribeTable("should respond with a validation error when plugin-side config invalid",
			func(ver int, dirVal string, expectedErr string) {
				connectorConf := newConnectorConfigFn()(id)
				connectorConf.V3DummypayConfig.Directory = dirVal
				var err error
				if ver == 3 {
					_, err = app.GetValue().SDK().Payments.V3.InstallConnector(ctx, "dummypay", &connectorConf)
				} else {
					connectorConf := components.ConnectorConfig{
						DummyPayConfig: &components.DummyPayConfig{
							Name:              connectorConf.V3DummypayConfig.Name,
							FilePollingPeriod: connectorConf.V3DummypayConfig.PollingPeriod,
							Provider:          pointer.For("Dummypay"),
							Directory:         connectorConf.V3DummypayConfig.Directory,
						},
					}
					_, err = app.GetValue().SDK().Payments.V1.InstallConnector(ctx, components.ConnectorEnumDummyPay, connectorConf)
				}
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
			connectorID string
			id          uuid.UUID
		)
		JustBeforeEach(func() {
			id = uuid.New()
		})

		It("can update a connector config with v2 API", func() {
			var err error
			connectorID, _, err = installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, connectorID, "run-tasks-")

			dir, err := os.MkdirTemp("", "updateddir")
			Expect(err).To(BeNil())
			GinkgoT().Cleanup(func() {
				os.RemoveAll(dir)
			})

			config := components.ConnectorConfig{
				DummyPayConfig: &components.DummyPayConfig{
					Name:              "some name",
					FilePollingPeriod: pointer.For("2m"),
					Provider:          pointer.For("Dummypay"),
					Directory:         dir,
				},
			}
			_, err = app.GetValue().SDK().Payments.V1.UpdateConnectorConfigV1(ctx, components.ConnectorEnumDummyPay, connectorID, config)
			Expect(err).To(BeNil())

			getRes, err := app.GetValue().SDK().Payments.V1.ReadConnectorConfigV1(ctx, components.ConnectorEnumDummyPay, connectorID)
			Expect(err).To(BeNil())
			Expect(getRes.ConnectorConfigResponse).NotTo(BeNil())
			Expect(getRes.ConnectorConfigResponse.Data.DummyPayConfig).To(Equal(config.DummyPayConfig))
			Expect(getRes.ConnectorConfigResponse.Data.Type).To(Equal(components.ConnectorConfigTypeDummypay))
		})

		It("can update a connector config with v3 API", func() {
			id := uuid.New()
			connectorID, config, err := installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, connectorID, "run-tasks-")

			config.PollingPeriod = pointer.For("2m")
			_, err = app.GetValue().SDK().Payments.V3.V3UpdateConnectorConfig(ctx, connectorID, &components.V3UpdateConnectorRequest{
				V3DummypayConfig: config,
			})
			Expect(err).To(BeNil())

			getRes, err := app.GetValue().SDK().Payments.V3.GetConnectorConfig(ctx, connectorID)
			Expect(err).To(BeNil())
			Expect(getRes.V3GetConnectorConfigResponse).NotTo(BeNil())
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig).To(Equal(config))
		})

		DescribeTable("should respond with a validation error when plugin-side config invalid",
			func(dirValue string, expectedErr string) {
				connectorID, config, err := installConnector(ctx, app.GetValue(), id)
				Expect(err).To(BeNil())
				blockTillWorkflowComplete(ctx, connectorID, "run-tasks-")

				config.Directory = dirValue
				_, err = app.GetValue().SDK().Payments.V3.V3UpdateConnectorConfig(ctx, connectorID, &components.V3UpdateConnectorRequest{
					V3DummypayConfig: config,
				})
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("400"))
				Expect(err.Error()).To(ContainSubstring(expectedErr))
			},
			Entry("empty directory", "", "validation for 'Directory' failed on the 'required' tag"),
			Entry("invalid directory", "$#2djskajdj", "validation for 'Directory' failed on the 'dirpath' tag"),
		)
	})

	When("uninstalling a connector", func() {
		var (
			id uuid.UUID
		)
		JustBeforeEach(func() {
			id = uuid.New()
		})

		It("can be uninstalled with v3 API", func() {
			connectorID, _, err := installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())

			resp, err := app.GetValue().SDK().Payments.V3.UninstallConnector(ctx, connectorID)
			Expect(err).To(BeNil())

			delRes := resp.V3UninstallConnectorResponse
			Expect(err).To(BeNil())
			Expect(delRes.Data).NotTo(BeNil())
			taskID, err := models.TaskIDFromString(delRes.Data.TaskID)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("uninstall"))
			taskPoller := testserver.TaskPoller(ctx, GinkgoT(), app.GetValue())
			blockTillWorkflowComplete(ctx, connectorID, "uninstall")
			Eventually(taskPoller(delRes.Data.TaskID)).WithTimeout(models.DefaultConnectorClientTimeout * 2).Should(testserver.HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))
		})

		It("can be uninstalled using v2 API", func() {
			connectorID, _, err := installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())

			_, err = app.GetValue().SDK().Payments.V1.UninstallConnectorV1(ctx, components.ConnectorEnumDummyPay, connectorID)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, connectorID, "uninstall")
		})
	})

	When("resetting a connector", func() {
		var (
			id uuid.UUID
		)
		JustBeforeEach(func() {
			id = uuid.New()
		})

		It("can be reset with v3 API", func() {
			connectorID, _, err := installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())

			reset, err := app.GetValue().SDK().Payments.V3.ResetConnector(ctx, connectorID)
			Expect(err).To(BeNil())
			Expect(reset.V3ResetConnectorResponse.Data).NotTo(BeNil())
			taskID, err := models.TaskIDFromString(reset.V3ResetConnectorResponse.Data)
			Expect(err).To(BeNil())
			Expect(taskID.Reference).To(ContainSubstring("reset"))

			taskPoller := testserver.TaskPoller(ctx, GinkgoT(), app.GetValue())
			blockTillWorkflowComplete(ctx, connectorID, "reset")
			Eventually(taskPoller(taskID.String())).WithTimeout(models.DefaultConnectorClientTimeout * 2).Should(testserver.HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))
		})

		It("can be reset with v2 API", func() {
			connectorID, _, err := installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())

			_, err = app.GetValue().SDK().Payments.V1.ResetConnectorV1(ctx, components.ConnectorEnumDummyPay, connectorID)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, connectorID, "reset")
		})
	})

	When("fetching a single schedule for a connector", func() {
		var (
			connectorID string
			id          uuid.UUID

			expectedSchedule components.V3Schedule
		)
		JustBeforeEach(func() {
			id = uuid.New()

			var err error
			connectorID, _, err = installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())

			workflowID := blockTillWorkflowComplete(ctx, connectorID, "run-tasks-")
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorID)))

			listRes, err := app.GetValue().SDK().Payments.V3.ListConnectorSchedules(ctx, connectorID, nil, nil, nil)
			Expect(err).To(BeNil())
			Expect(listRes.V3ConnectorSchedulesCursorResponse).NotTo(BeNil())
			schedules := listRes.V3ConnectorSchedulesCursorResponse.Cursor.Data
			Expect(len(schedules) > 0).To(BeTrue())
			expectedSchedule = schedules[0]
		})

		It("can fetch a single schedule with v3 API", func(ctx SpecContext) {
			res, err := app.GetValue().SDK().Payments.V3.GetConnectorSchedule(ctx, connectorID, expectedSchedule.ID)
			Expect(err).To(BeNil())
			Expect(res.V3ConnectorScheduleResponse).NotTo(BeNil())
			schedule := res.V3ConnectorScheduleResponse.Data
			Expect(schedule).NotTo(BeNil())
			Expect(schedule.ConnectorID).To(Equal(connectorID))
			Expect(schedule.ID).To(Equal(expectedSchedule.ID))
			Expect(schedule.CreatedAt).To(Equal(expectedSchedule.CreatedAt))
		})
	})

	When("searching for schedules for a connector", func() {
		var (
			connectorID   string
			id            uuid.UUID
			expectedTypes = map[string]struct{}{
				"FetchAccounts":         {},
				"FetchExternalAccounts": {},
				"FetchPayments":         {},
				"FetchBalances":         {},
			}
		)
		JustBeforeEach(func() {
			id = uuid.New()

			var err error
			connectorID, _, err = installConnector(ctx, app.GetValue(), id)
			Expect(err).To(BeNil())

			workflowID := blockTillWorkflowComplete(ctx, connectorID, "run-tasks-")
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorID)))
		})

		It("can search for schedules with v3 API", func(ctx SpecContext) {
			schCl := temporalServer.GetValue().DefaultClient().ScheduleClient()
			list, err := schCl.List(ctx, client.ScheduleListOptions{PageSize: 1})
			Expect(err).To(BeNil())
			Expect(list.HasNext()).To(BeTrue())

			for list.HasNext() {
				schedule, err := list.Next()
				if !strings.Contains(schedule.ID, connectorID) {
					continue
				}
				Expect(err).To(BeNil())
				_, ok := expectedTypes[schedule.WorkflowType.Name]
				Expect(ok).To(BeTrue())
			}

			res, err := app.GetValue().SDK().Payments.V3.ListConnectorSchedules(ctx, connectorID, nil, nil, nil)
			Expect(err).To(BeNil())
			Expect(res.V3ConnectorSchedulesCursorResponse).NotTo(BeNil())
			schedules := res.V3ConnectorSchedulesCursorResponse.Cursor.Data
			Expect(len(schedules) > 0).To(BeTrue())
			for _, schedule := range schedules {
				Expect(schedule.ConnectorID).To(Equal(connectorID))
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
				err := testserver.ConnectorConfigs(ctx, app.GetValue(), ver, &res)
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
	Expect(err).To(BeNil())
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

func installConnector(
	ctx context.Context,
	srv *testserver.Server,
	ref uuid.UUID,
) (connectorID string, config *components.V3DummypayConfig, err error) {
	connectorConf := newConnectorConfigFn()(ref)
	install, err := srv.SDK().Payments.V3.InstallConnector(ctx, "dummypay", &connectorConf)
	if err != nil {
		return "", config, err
	}
	return install.V3InstallConnectorResponse.Data, connectorConf.V3DummypayConfig, nil
}

func newConnectorConfigFn() func(id uuid.UUID) components.V3InstallConnectorRequest {
	return func(id uuid.UUID) components.V3InstallConnectorRequest {
		dir, err := os.MkdirTemp("", "dummypay")
		Expect(err).To(BeNil())
		GinkgoT().Cleanup(func() {
			os.RemoveAll(dir)
		})

		return components.V3InstallConnectorRequest{
			V3DummypayConfig: &components.V3DummypayConfig{
				Name:          fmt.Sprintf("connector-%s", id.String()),
				PollingPeriod: pointer.For("30s"),
				PageSize:      pointer.For(int64(30)),
				Provider:      pointer.For("Dummypay"),
				Directory:     dir,
			},
		}
	}
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
