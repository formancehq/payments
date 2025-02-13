package engine_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/formancehq/go-libs/v2/logging"
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
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(
				expectedErr,
			)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return validation error when plugin registry fails with validation issues", func(ctx SpecContext) {
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(
				models.ErrInvalidConfig,
			)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(engine.ErrValidation))
		})

		It("should fail when storage error happens", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("storage err")
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
			store.EXPECT().ConnectorsInstall(gomock.Any(), gomock.Any()).Return(expectedErr)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should fail when workflow start fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow err")
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
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
			plgs.EXPECT().RegisterPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
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

	Context("forwarding a bank account to a connector", func() {
		var (
			bankID uuid.UUID
			connID models.ConnectorID
		)
		BeforeEach(func() {
			connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
			bankID = uuid.New()
		})

		It("should return not found error when storage doesn't find connector", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(
				nil, fmt.Errorf("some not found err: %w", storage.ErrNotFound),
			)
			_, err := eng.ForwardBankAccount(ctx, bankID, connID, false)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(engine.ErrNotFound))
		})

		It("should return not found error when storage doesn't find bank account", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(nil, nil)
			store.EXPECT().BankAccountsGet(gomock.Any(), bankID, false).Return(
				nil, fmt.Errorf("some not found err: %w", storage.ErrNotFound),
			)
			_, err := eng.ForwardBankAccount(ctx, bankID, connID, false)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(engine.ErrNotFound))
		})

		It("should return storage error when task cannot be upserted", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(nil, nil)
			store.EXPECT().BankAccountsGet(gomock.Any(), bankID, false).Return(nil, nil)
			expectedErr := fmt.Errorf("fffff")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(
				expectedErr,
			)
			_, err := eng.ForwardBankAccount(ctx, bankID, connID, false)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when workflow cannot be started", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(
				&models.Connector{ID: connID}, nil,
			)
			store.EXPECT().BankAccountsGet(gomock.Any(), bankID, false).Return(nil, nil)
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			expectedErr := fmt.Errorf("workflow failed")
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixBankAccountCreate, defaultTaskQueue),
				workflow.RunCreateBankAccount,
				gomock.AssignableToTypeOf(workflow.CreateBankAccount{}),
			).Return(nil, expectedErr)

			_, err := eng.ForwardBankAccount(ctx, bankID, connID, false)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should launch workflow and return task", func(ctx SpecContext) {
			store.EXPECT().ConnectorsGet(gomock.Any(), connID).Return(
				&models.Connector{ID: connID}, nil,
			)
			store.EXPECT().BankAccountsGet(gomock.Any(), bankID, false).Return(nil, nil)
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions(engine.IDPrefixBankAccountCreate, defaultTaskQueue),
				workflow.RunCreateBankAccount,
				gomock.AssignableToTypeOf(workflow.CreateBankAccount{}),
			).Return(nil, nil)

			task, err := eng.ForwardBankAccount(ctx, bankID, connID, false)
			Expect(err).To(BeNil())
			Expect(task.ID.Reference).To(ContainSubstring(engine.IDPrefixBankAccountCreate))
			Expect(task.ID.Reference).To(ContainSubstring(stackName))
			Expect(task.ConnectorID.String()).To(Equal(connID.String()))
			Expect(task.Status).To(Equal(models.TASK_STATUS_PROCESSING))
		})
	})
})
