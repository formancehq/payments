package engine_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/client"
	gomock "go.uber.org/mock/gomock"
)

func TestEngine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Engine Suite")
}

func WithWorkflowOptions(idPrefix, taskQueue string) gomock.Matcher {
	return workflowOptionsMatcher{expectedIDPrefix: idPrefix, expectedTaskQueue: taskQueue}
}

type workflowOptionsMatcher struct {
	expectedIDPrefix  string
	expectedTaskQueue string
}

func (m workflowOptionsMatcher) Matches(options any) bool {
	opts, ok := options.(client.StartWorkflowOptions)
	if !ok {
		return false
	}

	if !strings.HasPrefix(opts.ID, m.expectedIDPrefix) {
		return false
	}
	if opts.TaskQueue != m.expectedTaskQueue {
		return false
	}
	return true
}

func (m workflowOptionsMatcher) String() string {
	return "has same options"
}

var _ = Describe("Engine Tests", func() {
	var (
		stackName        string
		defaultTaskQueue string
		eng              engine.Engine
		store            *storage.MockStorage
		plgs             *plugins.MockPlugins
		cl               *activities.MockClient
		wr               *activities.MockWorkflowRun
	)
	BeforeEach(func() {
		stackName = "STACKNAME"
		defaultTaskQueue = engine.GetDefaultTaskQueue(stackName)
		ctrl := gomock.NewController(GinkgoT())
		logger := logging.NewDefaultLogger(GinkgoWriter, false, false, false)
		cl = activities.NewMockClient(ctrl)
		wr = activities.NewMockWorkflowRun(ctrl)
		store = storage.NewMockStorage(ctrl)
		plgs = plugins.NewMockPlugins(ctrl)
		eng = engine.New(logger, cl, store, plgs, stackName)
	})

	Context("on start", func() {
		var (
		//			config json.RawMessage
		)
		BeforeEach(func() {
			//			config = json.RawMessage(`{"name":"somename","pollingPeriod":"30s"}`)
		})

		It("should fail when unable to fetch connectors from storage", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("storage err")
			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(nil, expectedErr)
			err := eng.OnStart(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should do nothing if no connectors are present", func(ctx SpecContext) {
			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(&bunpaginate.Cursor[models.Connector]{}, nil)
			err := eng.OnStart(ctx)
			Expect(err).To(BeNil())
		})

		It("should fail when workflow cannot be launched", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow error")
			connector := models.Connector{Config: json.RawMessage(`{}`)}
			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(&bunpaginate.Cursor[models.Connector]{Data: []models.Connector{connector}}, nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixConnectorInstall, defaultTaskQueue),
				workflow.RunInstallConnector,
				gomock.AssignableToTypeOf(workflow.InstallConnector{}),
			).Return(nil, expectedErr)
			err := eng.OnStart(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should launch a workflow for each connector", func(ctx SpecContext) {
			connectors := []models.Connector{
				{Config: json.RawMessage(`{}`)},
				{Config: json.RawMessage(`{}`)},
			}
			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(&bunpaginate.Cursor[models.Connector]{Data: connectors}, nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixConnectorInstall, defaultTaskQueue),
				workflow.RunInstallConnector,
				gomock.AssignableToTypeOf(workflow.InstallConnector{}),
			).Return(wr, nil).MinTimes(len(connectors))
			err := eng.OnStart(ctx)
			Expect(err).To(BeNil())
		})
	})

	Context("installing a connector", func() {
		var (
			config json.RawMessage
		)
		BeforeEach(func() {
			config = json.RawMessage(`{"name":"somename","pollingPeriod":"30s"}`)
		})

		It("should return error when config has validation issues", func(ctx SpecContext) {
			_, err := eng.InstallConnector(ctx, "psp", json.RawMessage(`{}`))
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(engine.ErrValidation))
		})

		It("should return exact error when plugin registry fails with misc error", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("hi")
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(
				expectedErr,
			)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return validation error when plugin registry fails with validation issues", func(ctx SpecContext) {
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(
				models.ErrInvalidConfig,
			)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(engine.ErrValidation))
		})

		It("should fail when storage error happens", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("storage err")
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
			store.EXPECT().ConnectorsInstall(gomock.Any(), gomock.Any()).Return(expectedErr)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should fail when workflow start fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow err")
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
			store.EXPECT().ConnectorsInstall(gomock.Any(), gomock.Any()).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixConnectorInstall, defaultTaskQueue),
				workflow.RunInstallConnector,
				gomock.AssignableToTypeOf(workflow.InstallConnector{}),
			).Return(nil, expectedErr)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should call WorkflowRun.Get before returning", func(ctx SpecContext) {
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
			store.EXPECT().ConnectorsInstall(gomock.Any(), gomock.Any()).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixConnectorInstall, defaultTaskQueue),
				workflow.RunInstallConnector,
				gomock.AssignableToTypeOf(workflow.InstallConnector{}),
			).Return(wr, nil)
			wr.EXPECT().Get(gomock.Any(), nil).Return(nil)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).To(BeNil())
		})
	})

	Context("uninstalling a connector", func() {
		var (
			connID models.ConnectorID
		)
		BeforeEach(func() {
			connID = models.ConnectorID{Reference: uuid.New(), Provider: "dummypay"}
		})

		It("should return storage error when deletion flag cannot be set", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("storage err")
			store.EXPECT().ConnectorsScheduleForDeletion(gomock.Any(), connID).Return(expectedErr)
			_, err := eng.UninstallConnector(ctx, connID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return storage error when task cannot be inserted", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("task storage err")
			store.EXPECT().ConnectorsScheduleForDeletion(gomock.Any(), connID).Return(nil)
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(expectedErr)
			_, err := eng.UninstallConnector(ctx, connID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("calls task upsert twice on workflow failure", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow storage err")
			store.EXPECT().ConnectorsScheduleForDeletion(gomock.Any(), connID).Return(nil)
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil).MinTimes(2)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixConnectorUninstall, defaultTaskQueue),
				workflow.RunUninstallConnector,
				gomock.AssignableToTypeOf(workflow.UninstallConnector{}),
			).Return(nil, expectedErr)

			_, err := eng.UninstallConnector(ctx, connID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("returns a task without waiting for workflow run", func(ctx SpecContext) {
			store.EXPECT().ConnectorsScheduleForDeletion(gomock.Any(), connID).Return(nil)
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixConnectorUninstall, defaultTaskQueue),
				workflow.RunUninstallConnector,
				gomock.AssignableToTypeOf(workflow.UninstallConnector{}),
			).Return(nil, nil)

			task, err := eng.UninstallConnector(ctx, connID)
			Expect(err).To(BeNil())
			Expect(task.ID.Reference).To(ContainSubstring(engine.IDPrefixConnectorUninstall))
			Expect(task.ID.Reference).To(ContainSubstring(stackName))
			Expect(task.ID.ConnectorID).To(Equal(connID))
			Expect(task.ConnectorID.String()).To(Equal("")) // connID must be nil for uninstall tasks
			Expect(task.Status).To(Equal(models.TASK_STATUS_PROCESSING))
		})
	})

	Context("resetting a connector", func() {
		var (
			connID models.ConnectorID
		)
		BeforeEach(func() {
			connID = models.ConnectorID{Reference: uuid.New(), Provider: "dummypay"}
		})

		It("should return storage error when task cannot be inserted", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("task storage err")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(expectedErr)
			_, err := eng.ResetConnector(ctx, connID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("calls task upsert twice on workflow failure", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow storage err")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil).MinTimes(2)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixConnectorReset, defaultTaskQueue),
				workflow.RunResetConnector,
				gomock.AssignableToTypeOf(workflow.ResetConnector{}),
			).Return(nil, expectedErr)

			_, err := eng.ResetConnector(ctx, connID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("returns a task without waiting for workflow run", func(ctx SpecContext) {
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixConnectorReset, defaultTaskQueue),
				workflow.RunResetConnector,
				gomock.AssignableToTypeOf(workflow.ResetConnector{}),
			).Return(nil, nil)

			task, err := eng.ResetConnector(ctx, connID)
			Expect(err).To(BeNil())
			Expect(task.ID.Reference).To(ContainSubstring(engine.IDPrefixConnectorReset))
			Expect(task.ID.Reference).To(ContainSubstring(stackName))
			Expect(task.ID.ConnectorID).To(Equal(connID))
			Expect(task.ConnectorID.String()).To(Equal("")) // connID must be nil for reset tasks
			Expect(task.Status).To(Equal(models.TASK_STATUS_PROCESSING))
		})
	})

	Context("forwarding a bank account to a connector", func() {
		var (
			ba     models.BankAccount
			connID models.ConnectorID
		)
		BeforeEach(func() {
			connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
			ba = models.BankAccount{}
		})

		It("should return not found error when storage doesn't find connector", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(
				nil, fmt.Errorf("some not found err: %w", storage.ErrNotFound),
			)
			_, err := eng.ForwardBankAccount(ctx, ba, connID, false)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(engine.ErrNotFound))
		})

		It("should return original error when storage returns misc error from connector fetch", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("original")
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(nil, expectedErr)
			_, err := eng.ForwardBankAccount(ctx, ba, connID, false)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return storage error when task cannot be upserted", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(nil, nil)
			expectedErr := fmt.Errorf("fffff")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(
				expectedErr,
			)
			_, err := eng.ForwardBankAccount(ctx, ba, connID, false)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when workflow cannot be started", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(
				&models.Connector{ID: connID}, nil,
			)
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			expectedErr := fmt.Errorf("workflow failed")
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixBankAccountCreate, defaultTaskQueue),
				workflow.RunCreateBankAccount,
				gomock.AssignableToTypeOf(workflow.CreateBankAccount{}),
			).Return(nil, expectedErr)

			_, err := eng.ForwardBankAccount(ctx, ba, connID, false)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should launch workflow and return task", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(
				&models.Connector{ID: connID}, nil,
			)
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixBankAccountCreate, defaultTaskQueue),
				workflow.RunCreateBankAccount,
				gomock.AssignableToTypeOf(workflow.CreateBankAccount{}),
			).Return(nil, nil)

			task, err := eng.ForwardBankAccount(ctx, ba, connID, false)
			Expect(err).To(BeNil())
			Expect(task.ID.Reference).To(ContainSubstring(engine.IDPrefixBankAccountCreate))
			Expect(task.ID.Reference).To(ContainSubstring(stackName))
			Expect(task.ConnectorID.String()).To(Equal(connID.String()))
			Expect(task.Status).To(Equal(models.TASK_STATUS_PROCESSING))
		})
	})
})
