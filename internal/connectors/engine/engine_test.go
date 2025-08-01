package engine_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

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
	"github.com/pkg/errors"
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
		eng = engine.New(logger, cl, store, plgs, stackName, "")
	})

	Context("on start", func() {
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
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
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
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil).MinTimes(len(connectors))
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

		It("prefills config with default values when empty", func(ctx SpecContext) {
			connectorName := "non-default-val"
			expectedConfig := models.DefaultConfig()
			expectedConfig.Name = connectorName

			registerErr := errors.New("stop here")
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), expectedConfig, gomock.Any(), false).Return(registerErr)
			_, err := eng.InstallConnector(ctx, "psp", json.RawMessage(fmt.Sprintf(`{"name":"%s","pollingPeriod":"0s","pageSize":0}`, connectorName)))
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(registerErr))
		})

		It("should return exact error when plugin registry fails with misc error", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("hi")
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(
				expectedErr,
			)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return validation error when plugin registry fails with validation issues", func(ctx SpecContext) {
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(
				models.ErrInvalidConfig,
			)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(engine.ErrValidation))
		})

		It("should fail when storage error happens", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("storage err")
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
			store.EXPECT().ConnectorsInstall(gomock.Any(), gomock.Any()).Return(expectedErr)
			_, err := eng.InstallConnector(ctx, "psp", config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should fail when workflow start fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow err")
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
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
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(nil)
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

	Context("updating a connector", func() {
		var (
			config      json.RawMessage
			connectorID models.ConnectorID
		)
		BeforeEach(func() {
			config = json.RawMessage(`{"name":"somename","pollingPeriod":"30s"}`)
			connectorID = models.ConnectorID{Provider: "dummypay", Reference: uuid.New()}
		})

		It("should return error when config has validation issues", func(ctx SpecContext) {
			err := eng.UpdateConnector(ctx, connectorID, json.RawMessage(`{}`))

			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(engine.ErrValidation))
		})

		It("prefills config with default values when empty", func(ctx SpecContext) {
			connectorName := "new-name"
			expectedConfig := models.DefaultConfig()
			expectedConfig.Name = connectorName

			connector := &models.Connector{
				ID:     connectorID,
				Config: json.RawMessage(`{"name":"original-name"}`),
			}

			registerErr := errors.New("stop here")
			store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), expectedConfig, gomock.Any(), true).Return(registerErr)
			err := eng.UpdateConnector(ctx, connectorID, json.RawMessage(fmt.Sprintf(`{"name":"%s","pollingPeriod":"0s","pageSize":0}`, connectorName)))
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(registerErr))
		})

		It("should return exact error when plugin registry fails with misc error", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("hi")
			connector := &models.Connector{
				ID: connectorID,
			}
			store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return(
				expectedErr,
			)
			err := eng.UpdateConnector(ctx, connectorID, config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when storage fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("storage err")
			connector := &models.Connector{
				ID: connectorID,
			}
			store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return(nil)
			store.EXPECT().ConnectorsConfigUpdate(gomock.Any(), gomock.Any()).Return(expectedErr)
			err := eng.UpdateConnector(ctx, connectorID, config)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should store the updated config", func(ctx SpecContext) {
			newName := "new-name"
			inputJson := json.RawMessage(fmt.Sprintf(`{"name":"%s","pollingPeriod":"2m","pageSize":25}`, newName))
			connector := &models.Connector{
				ID:        connectorID,
				CreatedAt: time.Now().UTC(),
				Provider:  connectorID.Provider,
				Config:    json.RawMessage(`{"name":"original-name"}`),
			}
			store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(connector, nil)
			plgs.EXPECT().LoadPlugin(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return(nil)

			expectedConnector := models.Connector{
				ID:        connectorID,
				Name:      newName,
				CreatedAt: connector.CreatedAt,
				Provider:  connector.Provider,
				Config:    inputJson,
			}
			store.EXPECT().ConnectorsConfigUpdate(gomock.Any(), expectedConnector).Return(nil)
			err := eng.UpdateConnector(ctx, connectorID, inputJson)
			Expect(err).To(BeNil())
		})
	})

	Context("create pool", func() {
		var (
			poolID uuid.UUID
			acc1   models.AccountID
			acc2   models.AccountID
		)
		BeforeEach(func() {
			poolID = uuid.New()
			acc1 = models.AccountID{
				Reference:   "test",
				ConnectorID: models.ConnectorID{},
			}
			acc2 = models.AccountID{
				Reference:   "test",
				ConnectorID: models.ConnectorID{},
			}
		})

		It("should return error when pool creation fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("pool creation failed")
			store.EXPECT().PoolsUpsert(gomock.Any(), gomock.Any()).Return(expectedErr)
			err := eng.CreatePool(ctx, models.Pool{
				ID: poolID,
			})
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return a validation error when one of the accounts is not an INTERNAL one", func(ctx SpecContext) {
			store.EXPECT().AccountsGet(gomock.Any(), acc1).Return(&models.Account{
				ID:   acc1,
				Type: models.ACCOUNT_TYPE_INTERNAL,
			}, nil)
			store.EXPECT().AccountsGet(gomock.Any(), acc2).Return(&models.Account{
				ID:   acc2,
				Type: models.ACCOUNT_TYPE_EXTERNAL,
			}, nil)
			err := eng.CreatePool(ctx, models.Pool{
				ID:           poolID,
				PoolAccounts: []models.AccountID{acc1, acc2},
			})
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(engine.ErrValidation))
		})

		It("should work if the pool is created successfully", func(ctx SpecContext) {
			store.EXPECT().AccountsGet(gomock.Any(), acc1).Return(&models.Account{
				ID:   acc1,
				Type: models.ACCOUNT_TYPE_INTERNAL,
			}, nil)
			store.EXPECT().AccountsGet(gomock.Any(), acc2).Return(&models.Account{
				ID:   acc2,
				Type: models.ACCOUNT_TYPE_INTERNAL,
			}, nil)
			store.EXPECT().PoolsUpsert(gomock.Any(), gomock.Any()).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("pools-creation", defaultTaskQueue),
				workflow.RunSendEvents,
				gomock.AssignableToTypeOf(workflow.SendEvents{}),
			).Return(nil, nil)
			err := eng.CreatePool(ctx, models.Pool{
				ID:           poolID,
				PoolAccounts: []models.AccountID{acc1, acc2},
			})
			Expect(err).To(BeNil())
		})
	})

	Context("add account to pool", func() {
		var (
			poolID    uuid.UUID
			accountID models.AccountID
		)

		BeforeEach(func() {
			poolID = uuid.New()
			accountID = models.AccountID{
				Reference: "test",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test",
				},
			}
		})

		It("should return a storage error if account is not found", func(ctx SpecContext) {
			store.EXPECT().AccountsGet(gomock.Any(), accountID).Return(nil, storage.ErrNotFound)
			err := eng.AddAccountToPool(ctx, poolID, accountID)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(storage.ErrNotFound))
		})

		It("should return a validation error if account is not an internal account", func(ctx SpecContext) {
			store.EXPECT().AccountsGet(gomock.Any(), accountID).Return(&models.Account{
				ID:   accountID,
				Type: models.ACCOUNT_TYPE_EXTERNAL,
			}, nil)
			err := eng.AddAccountToPool(ctx, poolID, accountID)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError(engine.ErrValidation))
		})

		It("should return an error if the account is failing to be added to the pool", func(ctx SpecContext) {
			store.EXPECT().AccountsGet(gomock.Any(), accountID).Return(&models.Account{
				ID:   accountID,
				Type: models.ACCOUNT_TYPE_INTERNAL,
			}, nil)
			store.EXPECT().PoolsAddAccount(gomock.Any(), poolID, accountID).Return(fmt.Errorf("failed to add account to pool"))
			err := eng.AddAccountToPool(ctx, poolID, accountID)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to add account to pool"))
		})

		It("should work if the account is successfully added to the pool", func(ctx SpecContext) {
			store.EXPECT().AccountsGet(gomock.Any(), accountID).Return(&models.Account{
				ID:   accountID,
				Type: models.ACCOUNT_TYPE_INTERNAL,
			}, nil)
			store.EXPECT().PoolsAddAccount(gomock.Any(), poolID, accountID).Return(nil)
			store.EXPECT().PoolsGet(gomock.Any(), poolID).Return(&models.Pool{}, nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("pools-add-account", defaultTaskQueue),
				workflow.RunSendEvents,
				gomock.AssignableToTypeOf(workflow.SendEvents{}),
			).Return(nil, nil)
			err := eng.AddAccountToPool(ctx, poolID, accountID)
			Expect(err).To(BeNil())
		})
	})

	Context("forward payment service user", func() {
		var (
			psuID       uuid.UUID
			connectorID models.ConnectorID
			psu         *models.PaymentServiceUser
		)

		BeforeEach(func() {
			psuID = uuid.New()
			connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
			psu = &models.PaymentServiceUser{
				ID:   psuID,
				Name: "Test User",
			}
		})

		It("should return error when user already exists on connector", func(ctx SpecContext) {
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(&models.PSUBankBridge{}, nil)
			err := eng.ForwardPaymentServiceUser(ctx, psuID, connectorID)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("user already exists on this connector"))
		})

		It("should return error when storage returns misc error from bank bridge fetch", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("storage error")
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(nil, expectedErr)
			err := eng.ForwardPaymentServiceUser(ctx, psuID, connectorID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when payment service user not found", func(ctx SpecContext) {
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(nil, storage.ErrNotFound)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(nil, storage.ErrNotFound)
			err := eng.ForwardPaymentServiceUser(ctx, psuID, connectorID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(storage.ErrNotFound))
		})

		It("should return error when plugin not found", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("plugin not found")
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(nil, storage.ErrNotFound)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			plgs.EXPECT().Get(connectorID).Return(nil, expectedErr)
			err := eng.ForwardPaymentServiceUser(ctx, psuID, connectorID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when plugin CreateUser fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("plugin error")
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(nil, storage.ErrNotFound)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(models.CreateUserRequest{})).Return(models.CreateUserResponse{}, expectedErr)
			err := eng.ForwardPaymentServiceUser(ctx, psuID, connectorID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when bank bridge upsert fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("upsert error")
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(nil, storage.ErrNotFound)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(models.CreateUserRequest{})).Return(models.CreateUserResponse{}, nil)
			store.EXPECT().PSUBankBridgesUpsert(gomock.Any(), psuID, gomock.AssignableToTypeOf(models.PSUBankBridge{})).Return(expectedErr)
			err := eng.ForwardPaymentServiceUser(ctx, psuID, connectorID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should successfully forward payment service user", func(ctx SpecContext) {
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(nil, storage.ErrNotFound)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(gomock.Any(), gomock.AssignableToTypeOf(models.CreateUserRequest{})).Return(models.CreateUserResponse{}, nil)
			store.EXPECT().PSUBankBridgesUpsert(gomock.Any(), psuID, gomock.AssignableToTypeOf(models.PSUBankBridge{})).Return(nil)
			err := eng.ForwardPaymentServiceUser(ctx, psuID, connectorID)
			Expect(err).To(BeNil())
		})
	})

	Context("delete payment service user", func() {
		var (
			psuID uuid.UUID
		)

		BeforeEach(func() {
			psuID = uuid.New()
		})

		It("should return error when task upsert fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("task storage error")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(expectedErr)
			_, err := eng.DeletePaymentServiceUser(ctx, psuID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when workflow execution fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow error")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("delete-user", defaultTaskQueue),
				workflow.RunDeletePSU,
				gomock.AssignableToTypeOf(workflow.DeletePSU{}),
			).Return(nil, expectedErr)
			_, err := eng.DeletePaymentServiceUser(ctx, psuID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should successfully delete payment service user and return task", func(ctx SpecContext) {
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("delete-user", defaultTaskQueue),
				workflow.RunDeletePSU,
				gomock.AssignableToTypeOf(workflow.DeletePSU{}),
			).Return(nil, nil)
			task, err := eng.DeletePaymentServiceUser(ctx, psuID)
			Expect(err).To(BeNil())
			Expect(task.ID.Reference).To(ContainSubstring("delete-user"))
			Expect(task.ID.Reference).To(ContainSubstring(stackName))
			Expect(task.ID.Reference).To(ContainSubstring(psuID.String()))
			Expect(task.Status).To(Equal(models.TASK_STATUS_PROCESSING))
		})
	})

	Context("delete payment service user connector", func() {
		var (
			psuID       uuid.UUID
			connectorID models.ConnectorID
		)

		BeforeEach(func() {
			psuID = uuid.New()
			connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		})

		It("should return error when task upsert fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("task storage error")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(expectedErr)
			_, err := eng.DeletePaymentServiceUserConnector(ctx, psuID, connectorID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when workflow execution fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow error")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("delete-user-connector", defaultTaskQueue),
				workflow.RunDeletePSUConnector,
				gomock.AssignableToTypeOf(workflow.DeletePSUConnector{}),
			).Return(nil, expectedErr)
			_, err := eng.DeletePaymentServiceUserConnector(ctx, psuID, connectorID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should successfully delete payment service user connector and return task", func(ctx SpecContext) {
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("delete-user-connector", defaultTaskQueue),
				workflow.RunDeletePSUConnector,
				gomock.AssignableToTypeOf(workflow.DeletePSUConnector{}),
			).Return(nil, nil)
			task, err := eng.DeletePaymentServiceUserConnector(ctx, psuID, connectorID)
			Expect(err).To(BeNil())
			Expect(task.ID.Reference).To(ContainSubstring("delete-user-connector"))
			Expect(task.ID.Reference).To(ContainSubstring(stackName))
			Expect(task.ID.ConnectorID).To(Equal(connectorID))
			Expect(task.ConnectorID.String()).To(Equal(connectorID.String()))
			Expect(task.Status).To(Equal(models.TASK_STATUS_PROCESSING))
		})
	})

	Context("delete payment service user connection", func() {
		var (
			psuID        uuid.UUID
			connectorID  models.ConnectorID
			connectionID string
		)

		BeforeEach(func() {
			psuID = uuid.New()
			connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
			connectionID = "connection-123"
		})

		It("should return error when task upsert fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("task storage error")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(expectedErr)
			_, err := eng.DeletePaymentServiceUserConnection(ctx, connectorID, psuID, connectionID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when workflow execution fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow error")
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("delete-user-connection", defaultTaskQueue),
				workflow.RunDeletePSUConnection,
				gomock.AssignableToTypeOf(workflow.DeletePSUConnection{}),
			).Return(nil, expectedErr)
			_, err := eng.DeletePaymentServiceUserConnection(ctx, connectorID, psuID, connectionID)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should successfully delete payment service user connection and return task", func(ctx SpecContext) {
			store.EXPECT().TasksUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.Task{})).Return(nil)
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("delete-user-connection", defaultTaskQueue),
				workflow.RunDeletePSUConnection,
				gomock.AssignableToTypeOf(workflow.DeletePSUConnection{}),
			).Return(nil, nil)
			task, err := eng.DeletePaymentServiceUserConnection(ctx, connectorID, psuID, connectionID)
			Expect(err).To(BeNil())
			Expect(task.ID.Reference).To(ContainSubstring("delete-user-connection"))
			Expect(task.ID.Reference).To(ContainSubstring(stackName))
			Expect(task.ID.ConnectorID).To(Equal(connectorID))
			Expect(task.ConnectorID.String()).To(Equal(connectorID.String()))
			Expect(task.Status).To(Equal(models.TASK_STATUS_PROCESSING))
		})
	})

	Context("create payment service user link", func() {
		var (
			psuID             uuid.UUID
			connectorID       models.ConnectorID
			psu               *models.PaymentServiceUser
			bankBridge        *models.PSUBankBridge
			idempotencyKey    *uuid.UUID
			clientRedirectURL *string
		)

		BeforeEach(func() {
			psuID = uuid.New()
			connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
			psu = &models.PaymentServiceUser{
				ID:   psuID,
				Name: "Test User",
			}
			bankBridge = &models.PSUBankBridge{
				ConnectorID: connectorID,
			}
			redirectURL := "https://example.com/redirect"
			clientRedirectURL = &redirectURL
		})

		It("should return error when plugin not found", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("plugin not found")
			plgs.EXPECT().Get(connectorID).Return(nil, expectedErr)
			_, _, err := eng.CreatePaymentServiceUserLink(ctx, "Test", psuID, connectorID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when payment service user not found", func(ctx SpecContext) {
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(nil, storage.ErrNotFound)
			_, _, err := eng.CreatePaymentServiceUserLink(ctx, "Test", psuID, connectorID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(storage.ErrNotFound))
		})

		It("should return error when bank bridge not found", func(ctx SpecContext) {
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(nil, storage.ErrNotFound)
			_, _, err := eng.CreatePaymentServiceUserLink(ctx, "Test", psuID, connectorID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(storage.ErrNotFound))
		})

		It("should return error when connection attempt upsert fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("upsert error")
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(bankBridge, nil)
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(expectedErr)
			_, _, err := eng.CreatePaymentServiceUserLink(ctx, "Test", psuID, connectorID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when plugin CreateUserLink fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("plugin error")
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(bankBridge, nil)
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(nil)
			plugin.EXPECT().CreateUserLink(gomock.Any(), gomock.AssignableToTypeOf(models.CreateUserLinkRequest{})).Return(models.CreateUserLinkResponse{}, expectedErr)
			_, _, err := eng.CreatePaymentServiceUserLink(ctx, "Test", psuID, connectorID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when final attempt upsert fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("final upsert error")
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(bankBridge, nil)
			// First call to create the attempt should succeed
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(nil)
			plugin.EXPECT().CreateUserLink(gomock.Any(), gomock.AssignableToTypeOf(models.CreateUserLinkRequest{})).Return(models.CreateUserLinkResponse{Link: "https://example.com/link"}, nil)
			// Second call to update the attempt with temporary token should fail
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(expectedErr)
			_, _, err := eng.CreatePaymentServiceUserLink(ctx, "Test", psuID, connectorID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should successfully create payment service user link", func(ctx SpecContext) {
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(bankBridge, nil)
			// First call to create the attempt
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(nil)
			plugin.EXPECT().CreateUserLink(gomock.Any(), gomock.AssignableToTypeOf(models.CreateUserLinkRequest{})).Return(models.CreateUserLinkResponse{Link: "https://example.com/link"}, nil)
			// Second call to update the attempt with temporary token
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(nil)
			attemptID, link, err := eng.CreatePaymentServiceUserLink(ctx, "Test", psuID, connectorID, idempotencyKey, clientRedirectURL)
			Expect(err).To(BeNil())
			Expect(attemptID).NotTo(BeEmpty())
			Expect(link).To(Equal("https://example.com/link"))
		})
	})

	Context("update payment service user link", func() {
		var (
			psuID             uuid.UUID
			connectorID       models.ConnectorID
			connectionID      string
			psu               *models.PaymentServiceUser
			bankBridge        *models.PSUBankBridge
			connection        *models.PSUBankBridgeConnection
			idempotencyKey    *uuid.UUID
			clientRedirectURL *string
		)

		BeforeEach(func() {
			psuID = uuid.New()
			connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
			connectionID = "connection-123"
			psu = &models.PaymentServiceUser{
				ID:   psuID,
				Name: "Test User",
			}
			bankBridge = &models.PSUBankBridge{
				ConnectorID: connectorID,
			}
			connection = &models.PSUBankBridgeConnection{
				ConnectionID: connectionID,
				ConnectorID:  connectorID,
			}
			redirectURL := "https://example.com/redirect"
			clientRedirectURL = &redirectURL
		})

		It("should return error when plugin not found", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("plugin not found")
			plgs.EXPECT().Get(connectorID).Return(nil, expectedErr)
			_, _, err := eng.UpdatePaymentServiceUserLink(ctx, psuID, connectorID, connectionID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when payment service user not found", func(ctx SpecContext) {
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(nil, storage.ErrNotFound)
			_, _, err := eng.UpdatePaymentServiceUserLink(ctx, psuID, connectorID, connectionID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(storage.ErrNotFound))
		})

		It("should return error when bank bridge not found", func(ctx SpecContext) {
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(nil, storage.ErrNotFound)
			_, _, err := eng.UpdatePaymentServiceUserLink(ctx, psuID, connectorID, connectionID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(storage.ErrNotFound))
		})

		It("should return error when connection not found", func(ctx SpecContext) {
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(bankBridge, nil)
			store.EXPECT().PSUBankBridgeConnectionsGetFromConnectionID(gomock.Any(), connectorID, connectionID).Return(nil, uuid.Nil, storage.ErrNotFound)
			_, _, err := eng.UpdatePaymentServiceUserLink(ctx, psuID, connectorID, connectionID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(storage.ErrNotFound))
		})

		It("should return error when connection attempt upsert fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("upsert error")
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(bankBridge, nil)
			store.EXPECT().PSUBankBridgeConnectionsGetFromConnectionID(gomock.Any(), connectorID, connectionID).Return(connection, psuID, nil)
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(expectedErr)
			_, _, err := eng.UpdatePaymentServiceUserLink(ctx, psuID, connectorID, connectionID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should return error when plugin UpdateUserLink fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("plugin error")
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(bankBridge, nil)
			store.EXPECT().PSUBankBridgeConnectionsGetFromConnectionID(gomock.Any(), connectorID, connectionID).Return(connection, psuID, nil)
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(nil)
			plugin.EXPECT().UpdateUserLink(gomock.Any(), gomock.AssignableToTypeOf(models.UpdateUserLinkRequest{})).Return(models.UpdateUserLinkResponse{}, expectedErr)
			_, _, err := eng.UpdatePaymentServiceUserLink(ctx, psuID, connectorID, connectionID, idempotencyKey, clientRedirectURL)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should successfully update payment service user link", func(ctx SpecContext) {
			plugin := models.NewMockPlugin(gomock.NewController(GinkgoT()))
			plgs.EXPECT().Get(connectorID).Return(plugin, nil)
			store.EXPECT().PaymentServiceUsersGet(gomock.Any(), psuID).Return(psu, nil)
			store.EXPECT().PSUBankBridgesGet(gomock.Any(), psuID, connectorID).Return(bankBridge, nil)
			store.EXPECT().PSUBankBridgeConnectionsGetFromConnectionID(gomock.Any(), connectorID, connectionID).Return(connection, psuID, nil)
			store.EXPECT().PSUBankBridgeConnectionAttemptsUpsert(gomock.Any(), gomock.AssignableToTypeOf(models.PSUBankBridgeConnectionAttempt{})).Return(nil).MinTimes(2)
			plugin.EXPECT().UpdateUserLink(gomock.Any(), gomock.AssignableToTypeOf(models.UpdateUserLinkRequest{})).Return(models.UpdateUserLinkResponse{Link: "https://example.com/update-link"}, nil)
			attemptID, link, err := eng.UpdatePaymentServiceUserLink(ctx, psuID, connectorID, connectionID, idempotencyKey, clientRedirectURL)
			Expect(err).To(BeNil())
			Expect(attemptID).NotTo(BeEmpty())
			Expect(link).To(Equal("https://example.com/update-link"))
		})
	})

	Context("complete payment service user link", func() {
		var (
			connectorID  models.ConnectorID
			attemptID    uuid.UUID
			httpCallInfo models.HTTPCallInformation
		)

		BeforeEach(func() {
			connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
			attemptID = uuid.New()
			httpCallInfo = models.HTTPCallInformation{
				Headers: map[string][]string{"Content-Type": {"application/json"}},
				Body:    []byte(`{"test": "data"}`),
			}
		})

		It("should return error when workflow execution fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("workflow error")
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("complete-user-link", defaultTaskQueue),
				workflow.RunCompleteUserLink,
				gomock.AssignableToTypeOf(workflow.CompleteUserLink{}),
			).Return(nil, expectedErr)
			err := eng.CompletePaymentServiceUserLink(ctx, connectorID, attemptID, httpCallInfo)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should successfully complete payment service user link", func(ctx SpecContext) {
			cl.EXPECT().ExecuteWorkflow(gomock.Any(), WithWorkflowOptions("complete-user-link", defaultTaskQueue),
				workflow.RunCompleteUserLink,
				gomock.AssignableToTypeOf(workflow.CompleteUserLink{}),
			).Return(nil, nil)
			err := eng.CompletePaymentServiceUserLink(ctx, connectorID, attemptID, httpCallInfo)
			Expect(err).To(BeNil())
		})
	})
})
