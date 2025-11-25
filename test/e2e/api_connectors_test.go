//go:build it

package test_suite

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/testserver"
	"github.com/google/uuid"
	v17 "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Connectors", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()
	)

	app := testserver.NewTestServer(func() testserver.Configuration {
		return testserver.Configuration{
			Stack:                      stack,
			PostgresConfiguration:      db.GetValue().ConnectionOptions(),
			TemporalNamespace:          temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:            temporalServer.GetValue().Address(),
			Output:                     GinkgoWriter,
			Debug:                      true,
			SkipOutboxScheduleCreation: true,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("installing a connector", func() {
		var (
			id         uuid.UUID
			workflowID string
		)

		BeforeEach(func() {
			id = uuid.New()
		})

		It("can install a connector with v3 API", func() {
			connectorConf := newV3ConnectorConfigFn()(id)
			connectorID, err := installV3Connector(ctx, app.GetValue(), connectorConf, id)
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
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.Name).To(Equal(connectorConf.Name))
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.Provider).To(Equal(connectorConf.Provider))
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.Directory).To(Equal(connectorConf.Directory))
			Expect(getRes.V3GetConnectorConfigResponse.Data.Type).To(Equal(components.V3ConnectorConfigTypeDummypay))

			getResPollingPeriod, err := time.ParseDuration(
				*getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.PollingPeriod,
			)
			Expect(err).To(BeNil())
			configPollingPeriod, err := time.ParseDuration(*connectorConf.PollingPeriod)
			Expect(err).To(BeNil())

			Expect(getResPollingPeriod).To(Equal(configPollingPeriod))
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
					FilePollingPeriod: pointer.For("30m"),
					Provider:          pointer.For("Dummypay"),
					Directory:         dir,
				},
			}
			res, err := app.GetValue().SDK().Payments.V1.InstallConnector(ctx, components.ConnectorDummyPay, connectorConf)
			Expect(err).To(BeNil())
			Expect(res.ConnectorResponse).NotTo(BeNil())
			connectorID := res.ConnectorResponse.Data.ConnectorID

			getRes, err := app.GetValue().SDK().Payments.V1.ReadConnectorConfigV1(ctx, components.ConnectorDummyPay, connectorID)
			Expect(err).To(BeNil())
			Expect(getRes.ConnectorConfigResponse).NotTo(BeNil())
			Expect(getRes.ConnectorConfigResponse.Data.DummyPayConfig.Name).To(Equal(connectorConf.DummyPayConfig.Name))
			Expect(getRes.ConnectorConfigResponse.Data.DummyPayConfig.Directory).To(Equal(dir))
			Expect(getRes.ConnectorConfigResponse.Data.Type).To(Equal(components.ConnectorConfigTypeDummypay))
		})

		DescribeTable("should respond with a validation error when plugin-side config invalid",
			func(ver int, dirVal string, expectedErr string) {
				connectorConf := newV3ConnectorConfigFn()(id)
				connectorConf.Directory = dirVal
				var err error
				if ver == 3 {
					_, err = app.GetValue().SDK().Payments.V3.InstallConnector(ctx, "dummypay", &components.V3InstallConnectorRequest{
						V3DummypayConfig: connectorConf,
					})
				} else {
					connectorConf := components.ConnectorConfig{
						DummyPayConfig: &components.DummyPayConfig{
							Name:              connectorConf.Name,
							FilePollingPeriod: connectorConf.PollingPeriod,
							Provider:          pointer.For("Dummypay"),
							Directory:         connectorConf.Directory,
						},
					}
					_, err = app.GetValue().SDK().Payments.V1.InstallConnector(ctx, components.ConnectorDummyPay, connectorConf)
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
		BeforeEach(func() {
			id = uuid.New()
		})

		It("can update a connector config with v2 API", func() {
			var err error
			connectorID, err = installV2Connector(ctx, app.GetValue(), nil, id)
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
					FilePollingPeriod: pointer.For("30m"),
					Provider:          pointer.For("Dummypay"),
					Directory:         dir,
				},
			}
			_, err = app.GetValue().SDK().Payments.V1.UpdateConnectorConfigV1(ctx, components.ConnectorDummyPay, connectorID, config)
			Expect(err).To(BeNil())

			getRes, err := app.GetValue().SDK().Payments.V1.ReadConnectorConfigV1(ctx, components.ConnectorDummyPay, connectorID)
			Expect(err).To(BeNil())
			Expect(getRes.ConnectorConfigResponse).NotTo(BeNil())
			Expect(getRes.ConnectorConfigResponse.Data.DummyPayConfig.Name).To(Equal(config.DummyPayConfig.Name))
			Expect(getRes.ConnectorConfigResponse.Data.DummyPayConfig.Directory).To(Equal(dir))
			Expect(getRes.ConnectorConfigResponse.Data.Type).To(Equal(components.ConnectorConfigTypeDummypay))
		})

		It("can update a connector config with v3 API", func() {
			id := uuid.New()
			config := newV3ConnectorConfigFn()(id)
			connectorID, err := installV3Connector(ctx, app.GetValue(), config, id)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, connectorID, "run-tasks-")

			config.PollingPeriod = pointer.For("30m0s")
			_, err = app.GetValue().SDK().Payments.V3.V3UpdateConnectorConfig(ctx, connectorID, &components.V3UpdateConnectorRequest{
				V3DummypayConfig: config,
			})
			Expect(err).To(BeNil())

			getRes, err := app.GetValue().SDK().Payments.V3.GetConnectorConfig(ctx, connectorID)
			Expect(err).To(BeNil())
			Expect(getRes.V3GetConnectorConfigResponse).NotTo(BeNil())

			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.Directory).To(Equal(config.Directory))
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.LinkFlowError).To(Equal(config.LinkFlowError))
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.Name).To(Equal(config.Name))
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.PageSize).To(Equal(pointer.For(int64(25)))) // the response sets a default value
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.PollingPeriod).To(Equal(config.PollingPeriod))
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.Provider).To(Equal(config.Provider))
			Expect(getRes.V3GetConnectorConfigResponse.Data.V3DummypayConfig.UpdateLinkFlowError).To(Equal(config.UpdateLinkFlowError))
		})

		DescribeTable("should respond with a validation error when plugin-side config invalid",
			func(dirValue string, expectedErr string) {
				config := newV3ConnectorConfigFn()(id)
				connectorID, err := installV3Connector(ctx, app.GetValue(), config, id)
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
		BeforeEach(func() {
			id = uuid.New()
		})

		It("can be uninstalled with v3 API", func() {
			connectorID, err := installV3Connector(ctx, app.GetValue(), nil, id)
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
			connectorID, err := installV3Connector(ctx, app.GetValue(), nil, id)
			Expect(err).To(BeNil())

			_, err = app.GetValue().SDK().Payments.V1.UninstallConnectorV1(ctx, components.ConnectorDummyPay, connectorID)
			Expect(err).To(BeNil())
			blockTillWorkflowComplete(ctx, connectorID, "uninstall")
		})
	})

	When("resetting a connector", func() {
		var (
			id uuid.UUID
		)
		BeforeEach(func() {
			id = uuid.New()
		})

		It("can be reset with v3 API", func() {
			connectorID, err := installV3Connector(ctx, app.GetValue(), nil, id)
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
			connectorID, err := installV3Connector(ctx, app.GetValue(), nil, id)
			Expect(err).To(BeNil())

			_, err = app.GetValue().SDK().Payments.V1.ResetConnectorV1(ctx, components.ConnectorDummyPay, connectorID)
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
		BeforeEach(func() {
			id = uuid.New()

			var err error
			connectorID, err = installV3Connector(ctx, app.GetValue(), nil, id)
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
			connectorID2  string
			expectedTypes = map[string]struct{}{
				"FetchAccounts":         {},
				"FetchExternalAccounts": {},
				"FetchPayments":         {},
				"FetchBalances":         {},
			}
		)
		BeforeEach(func() {
			var err error
			connectorID, err = installV3Connector(ctx, app.GetValue(), nil, uuid.New())
			Expect(err).To(BeNil())

			workflowID := blockTillWorkflowComplete(ctx, connectorID, "run-tasks-")
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorID)))

			connectorID2, err = installV3Connector(ctx, app.GetValue(), nil, uuid.New())
			Expect(err).To(BeNil())

			workflowID = blockTillWorkflowComplete(ctx, connectorID2, "run-tasks-")
			Expect(workflowID).To(Equal(fmt.Sprintf("run-tasks-%s-%s", stack, connectorID2)))
		})

		It("can search for schedules with v3 API - only returns results for applicable connectorID", func(ctx SpecContext) {
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

			res, err = app.GetValue().SDK().Payments.V3.ListConnectorSchedules(ctx, connectorID2, nil, nil, nil)
			Expect(err).To(BeNil())
			Expect(res.V3ConnectorSchedulesCursorResponse).NotTo(BeNil())
			schedules = res.V3ConnectorSchedulesCursorResponse.Cursor.Data
			Expect(len(schedules) > 0).To(BeTrue())
			for _, schedule := range schedules {
				Expect(schedule.ConnectorID).To(Equal(connectorID2))
			}
		})
	})

	When("fetching connector configurations", func() {
		It("should respond with detailed config json for each connector in v3", func() {
			resp, err := app.GetValue().SDK().Payments.V3.ListConnectorConfigs(ctx)
			Expect(err).To(BeNil())
			res := resp.V3ConnectorConfigsResponse
			Expect(len(res.Data)).To(BeNumerically(">", 1))
			Expect(res.Data["dummypay"]).NotTo(BeNil())
			Expect(res.Data["dummypay"]["pollingPeriod"]).NotTo(BeNil())
			Expect(res.Data["dummypay"]["pollingPeriod"].DataType).To(Equal("duration ns"))
			Expect(res.Data["dummypay"]["pollingPeriod"].Required).To(Equal(false))
			Expect(res.Data["dummypay"]["pollingPeriod"].DefaultValue).NotTo(Equal(""))
			Expect(res.Data["dummypay"]["pollingPeriod"].DefaultValue).ToNot(BeNil())
			pollingPeriod, err := time.ParseDuration(*res.Data["dummypay"]["pollingPeriod"].DefaultValue)
			Expect(err).To(BeNil())
			Expect(pollingPeriod).To(BeNumerically(">", 0))
			Expect(res.Data["dummypay"]["name"]).NotTo(BeNil())
			Expect(res.Data["dummypay"]["name"].DataType).To(Equal("string"))
			Expect(res.Data["dummypay"]["name"].Required).To(Equal(true))
			Expect(res.Data["dummypay"]["name"].DefaultValue).ToNot(BeNil())
			Expect(*res.Data["dummypay"]["name"].DefaultValue).To(Equal(""))
			Expect(res.Data["dummypay"]["directory"]).NotTo(BeNil())
			Expect(res.Data["dummypay"]["directory"].DataType).To(Equal("string"))
			Expect(res.Data["dummypay"]["directory"].Required).To(Equal(true))
			Expect(res.Data["dummypay"]["directory"].DefaultValue).ToNot(BeNil())
			Expect(*res.Data["dummypay"]["directory"].DefaultValue).To(Equal(""))
		})

		It("should respond with detailed config json for each connector in v2", func() {
			resp, err := app.GetValue().SDK().Payments.V1.ListConfigsAvailableConnectors(ctx)
			Expect(err).To(BeNil())
			res := resp.ConnectorsConfigsResponse
			Expect(len(res.Data)).To(BeNumerically(">", 1))
			Expect(res.Data["dummypay"]).NotTo(BeNil())
			Expect(res.Data["dummypay"]["pollingPeriod"]).NotTo(BeNil())
			Expect(res.Data["dummypay"]["pollingPeriod"].DataType).To(Equal("duration ns"))
			Expect(res.Data["dummypay"]["pollingPeriod"].Required).To(Equal(false))
			Expect(res.Data["dummypay"]["pollingPeriod"].DefaultValue).NotTo(Equal(""))
			Expect(res.Data["dummypay"]["pollingPeriod"].DefaultValue).ToNot(BeNil())
			pollingPeriod, err := time.ParseDuration(*res.Data["dummypay"]["pollingPeriod"].DefaultValue)
			Expect(err).To(BeNil())
			Expect(pollingPeriod).To(BeNumerically(">", 0))
			Expect(res.Data["dummypay"]["name"]).NotTo(BeNil())
			Expect(res.Data["dummypay"]["name"].DataType).To(Equal("string"))
			Expect(res.Data["dummypay"]["name"].Required).To(Equal(true))
			Expect(res.Data["dummypay"]["name"].DefaultValue).ToNot(BeNil())
			Expect(*res.Data["dummypay"]["name"].DefaultValue).To(Equal(""))
			Expect(res.Data["dummypay"]["directory"]).NotTo(BeNil())
			Expect(res.Data["dummypay"]["directory"].DataType).To(Equal("string"))
			Expect(res.Data["dummypay"]["directory"].Required).To(Equal(true))
			Expect(res.Data["dummypay"]["directory"].DefaultValue).ToNot(BeNil())
			Expect(*res.Data["dummypay"]["directory"].DefaultValue).To(Equal(""))
		})
	})
})

// if this turns out to be slow we might also consider executing each test suite on its own namespace
// presumably there would be fewer competing workflow executions that way
func blockTillWorkflowComplete(ctx context.Context, connectorIDStr string, searchKeyword string) string {
	var (
		workflowID string
		runID      string
	)

	connectorID := models.MustConnectorIDFromString(connectorIDStr)
	cl := temporalServer.GetValue().DefaultClient()
	iterateThroughTemporalWorkflowExecutions(ctx, cl, 25, func(info *v17.WorkflowExecutionInfo) bool {
		if strings.Contains(info.Execution.WorkflowId, connectorID.String()) && strings.HasPrefix(info.Execution.WorkflowId, searchKeyword) {
			workflowID = info.Execution.WorkflowId
			runID = info.Execution.RunId
			return true
		}
		return false
	})

	// if we couldn't find it either it's already done or it wasn't scheduled
	if workflowID == "" {
		return ""
	}
	workflowRun := cl.GetWorkflow(ctx, workflowID, runID)
	err := workflowRun.Get(ctx, nil) // blocks to ensure workflow is finished
	Expect(err).To(BeNil())
	return workflowID
}

func installConnector(
	ctx context.Context,
	srv *testserver.Server,
	ref uuid.UUID,
	ver int,
) (connectorID string, err error) {
	switch {
	case ver < 3:
		return installV2Connector(ctx, srv, nil, ref)
	default:
		return installV3Connector(ctx, srv, nil, ref)
	}
}

func installV3Connector(
	ctx context.Context,
	srv *testserver.Server,
	connectorConf *components.V3DummypayConfig,
	ref uuid.UUID,
) (connectorID string, err error) {
	if connectorConf == nil {
		connectorConf = newV3ConnectorConfigFn()(ref)
	}
	install, err := srv.SDK().Payments.V3.InstallConnector(ctx, "dummypay", &components.V3InstallConnectorRequest{
		V3DummypayConfig: connectorConf,
	})
	if err != nil {
		return "", err
	}
	return install.V3InstallConnectorResponse.Data, nil
}

func installV2Connector(
	ctx context.Context,
	srv *testserver.Server,
	connectorConf *components.DummyPayConfig,
	ref uuid.UUID,
) (connectorID string, err error) {
	if connectorConf == nil {
		connectorConf = newV2ConnectorConfigFn()(ref)
	}
	install, err := srv.SDK().Payments.V1.InstallConnector(ctx, "dummypay", components.ConnectorConfig{
		DummyPayConfig: connectorConf,
	})
	if err != nil {
		return "", err
	}
	return install.GetConnectorResponse().Data.ConnectorID, nil
}

func uninstallConnector(
	ctx context.Context,
	srv *testserver.Server,
	connectorID string,
) {
	resp, err := srv.SDK().Payments.V3.UninstallConnector(ctx, connectorID)
	Expect(err).To(BeNil())

	delRes := resp.V3UninstallConnectorResponse
	Expect(err).To(BeNil())
	Expect(delRes.Data).NotTo(BeNil())
	taskID, err := models.TaskIDFromString(delRes.Data.TaskID)
	Expect(err).To(BeNil())
	Expect(taskID.Reference).To(ContainSubstring("uninstall"))
	taskPoller := testserver.TaskPoller(ctx, GinkgoT(), srv)
	blockTillWorkflowComplete(ctx, connectorID, "uninstall")
	Eventually(taskPoller(delRes.Data.TaskID)).WithTimeout(models.DefaultConnectorClientTimeout * 2).Should(testserver.HaveTaskStatus(models.TASK_STATUS_SUCCEEDED))
}

func newV3ConnectorConfigFn() func(id uuid.UUID) *components.V3DummypayConfig {
	return func(id uuid.UUID) *components.V3DummypayConfig {
		dir, err := os.MkdirTemp("", "dummypay")
		Expect(err).To(BeNil())
		GinkgoT().Cleanup(func() {
			os.RemoveAll(dir)
		})

		return &components.V3DummypayConfig{
			Directory:           dir,
			LinkFlowError:       pointer.For(false),
			Name:                fmt.Sprintf("connector-%s", id.String()),
			PollingPeriod:       pointer.For("30m"),
			Provider:            pointer.For("Dummypay"),
			UpdateLinkFlowError: pointer.For(false),
		}
	}
}

func newV2ConnectorConfigFn() func(id uuid.UUID) *components.DummyPayConfig {
	return func(id uuid.UUID) *components.DummyPayConfig {
		dir, err := os.MkdirTemp("", "dummypay")
		Expect(err).To(BeNil())
		GinkgoT().Cleanup(func() {
			os.RemoveAll(dir)
		})

		return &components.DummyPayConfig{
			Name:              fmt.Sprintf("connector-%s", id.String()),
			FilePollingPeriod: pointer.For("30m"),
			Provider:          pointer.For("Dummypay"),
			Directory:         dir,
		}
	}
}
