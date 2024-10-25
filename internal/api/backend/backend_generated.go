// Code generated by MockGen. DO NOT EDIT.
// Source: backend.go
//
// Generated by this command:
//
//	mockgen -source backend.go -destination backend_generated.go -package backend . Backend
//

// Package backend is a generated GoMock package.
package backend

import (
	context "context"
	json "encoding/json"
	reflect "reflect"
	time "time"

	bunpaginate "github.com/formancehq/go-libs/v2/bun/bunpaginate"
	plugins "github.com/formancehq/payments/internal/connectors/plugins"
	models "github.com/formancehq/payments/internal/models"
	storage "github.com/formancehq/payments/internal/storage"
	uuid "github.com/google/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockBackend is a mock of Backend interface.
type MockBackend struct {
	ctrl     *gomock.Controller
	recorder *MockBackendMockRecorder
}

// MockBackendMockRecorder is the mock recorder for MockBackend.
type MockBackendMockRecorder struct {
	mock *MockBackend
}

// NewMockBackend creates a new mock instance.
func NewMockBackend(ctrl *gomock.Controller) *MockBackend {
	mock := &MockBackend{ctrl: ctrl}
	mock.recorder = &MockBackendMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackend) EXPECT() *MockBackendMockRecorder {
	return m.recorder
}

// AccountsCreate mocks base method.
func (m *MockBackend) AccountsCreate(ctx context.Context, account models.Account) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AccountsCreate", ctx, account)
	ret0, _ := ret[0].(error)
	return ret0
}

// AccountsCreate indicates an expected call of AccountsCreate.
func (mr *MockBackendMockRecorder) AccountsCreate(ctx, account any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AccountsCreate", reflect.TypeOf((*MockBackend)(nil).AccountsCreate), ctx, account)
}

// AccountsGet mocks base method.
func (m *MockBackend) AccountsGet(ctx context.Context, id models.AccountID) (*models.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AccountsGet", ctx, id)
	ret0, _ := ret[0].(*models.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AccountsGet indicates an expected call of AccountsGet.
func (mr *MockBackendMockRecorder) AccountsGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AccountsGet", reflect.TypeOf((*MockBackend)(nil).AccountsGet), ctx, id)
}

// AccountsList mocks base method.
func (m *MockBackend) AccountsList(ctx context.Context, query storage.ListAccountsQuery) (*bunpaginate.Cursor[models.Account], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AccountsList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.Account])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AccountsList indicates an expected call of AccountsList.
func (mr *MockBackendMockRecorder) AccountsList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AccountsList", reflect.TypeOf((*MockBackend)(nil).AccountsList), ctx, query)
}

// BalancesList mocks base method.
func (m *MockBackend) BalancesList(ctx context.Context, query storage.ListBalancesQuery) (*bunpaginate.Cursor[models.Balance], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BalancesList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.Balance])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BalancesList indicates an expected call of BalancesList.
func (mr *MockBackendMockRecorder) BalancesList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BalancesList", reflect.TypeOf((*MockBackend)(nil).BalancesList), ctx, query)
}

// BankAccountsCreate mocks base method.
func (m *MockBackend) BankAccountsCreate(ctx context.Context, bankAccount models.BankAccount) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BankAccountsCreate", ctx, bankAccount)
	ret0, _ := ret[0].(error)
	return ret0
}

// BankAccountsCreate indicates an expected call of BankAccountsCreate.
func (mr *MockBackendMockRecorder) BankAccountsCreate(ctx, bankAccount any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BankAccountsCreate", reflect.TypeOf((*MockBackend)(nil).BankAccountsCreate), ctx, bankAccount)
}

// BankAccountsForwardToConnector mocks base method.
func (m *MockBackend) BankAccountsForwardToConnector(ctx context.Context, bankAccountID uuid.UUID, connectorID models.ConnectorID, waitResult bool) (models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BankAccountsForwardToConnector", ctx, bankAccountID, connectorID, waitResult)
	ret0, _ := ret[0].(models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BankAccountsForwardToConnector indicates an expected call of BankAccountsForwardToConnector.
func (mr *MockBackendMockRecorder) BankAccountsForwardToConnector(ctx, bankAccountID, connectorID, waitResult any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BankAccountsForwardToConnector", reflect.TypeOf((*MockBackend)(nil).BankAccountsForwardToConnector), ctx, bankAccountID, connectorID, waitResult)
}

// BankAccountsGet mocks base method.
func (m *MockBackend) BankAccountsGet(ctx context.Context, id uuid.UUID) (*models.BankAccount, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BankAccountsGet", ctx, id)
	ret0, _ := ret[0].(*models.BankAccount)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BankAccountsGet indicates an expected call of BankAccountsGet.
func (mr *MockBackendMockRecorder) BankAccountsGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BankAccountsGet", reflect.TypeOf((*MockBackend)(nil).BankAccountsGet), ctx, id)
}

// BankAccountsList mocks base method.
func (m *MockBackend) BankAccountsList(ctx context.Context, query storage.ListBankAccountsQuery) (*bunpaginate.Cursor[models.BankAccount], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BankAccountsList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.BankAccount])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BankAccountsList indicates an expected call of BankAccountsList.
func (mr *MockBackendMockRecorder) BankAccountsList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BankAccountsList", reflect.TypeOf((*MockBackend)(nil).BankAccountsList), ctx, query)
}

// BankAccountsUpdateMetadata mocks base method.
func (m *MockBackend) BankAccountsUpdateMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BankAccountsUpdateMetadata", ctx, id, metadata)
	ret0, _ := ret[0].(error)
	return ret0
}

// BankAccountsUpdateMetadata indicates an expected call of BankAccountsUpdateMetadata.
func (mr *MockBackendMockRecorder) BankAccountsUpdateMetadata(ctx, id, metadata any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BankAccountsUpdateMetadata", reflect.TypeOf((*MockBackend)(nil).BankAccountsUpdateMetadata), ctx, id, metadata)
}

// ConnectorsConfig mocks base method.
func (m *MockBackend) ConnectorsConfig(ctx context.Context, connectorID models.ConnectorID) (json.RawMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectorsConfig", ctx, connectorID)
	ret0, _ := ret[0].(json.RawMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConnectorsConfig indicates an expected call of ConnectorsConfig.
func (mr *MockBackendMockRecorder) ConnectorsConfig(ctx, connectorID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectorsConfig", reflect.TypeOf((*MockBackend)(nil).ConnectorsConfig), ctx, connectorID)
}

// ConnectorsConfigs mocks base method.
func (m *MockBackend) ConnectorsConfigs() plugins.Configs {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectorsConfigs")
	ret0, _ := ret[0].(plugins.Configs)
	return ret0
}

// ConnectorsConfigs indicates an expected call of ConnectorsConfigs.
func (mr *MockBackendMockRecorder) ConnectorsConfigs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectorsConfigs", reflect.TypeOf((*MockBackend)(nil).ConnectorsConfigs))
}

// ConnectorsHandleWebhooks mocks base method.
func (m *MockBackend) ConnectorsHandleWebhooks(ctx context.Context, urlPath string, webhook models.Webhook) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectorsHandleWebhooks", ctx, urlPath, webhook)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConnectorsHandleWebhooks indicates an expected call of ConnectorsHandleWebhooks.
func (mr *MockBackendMockRecorder) ConnectorsHandleWebhooks(ctx, urlPath, webhook any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectorsHandleWebhooks", reflect.TypeOf((*MockBackend)(nil).ConnectorsHandleWebhooks), ctx, urlPath, webhook)
}

// ConnectorsInstall mocks base method.
func (m *MockBackend) ConnectorsInstall(ctx context.Context, provider string, config json.RawMessage) (models.ConnectorID, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectorsInstall", ctx, provider, config)
	ret0, _ := ret[0].(models.ConnectorID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConnectorsInstall indicates an expected call of ConnectorsInstall.
func (mr *MockBackendMockRecorder) ConnectorsInstall(ctx, provider, config any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectorsInstall", reflect.TypeOf((*MockBackend)(nil).ConnectorsInstall), ctx, provider, config)
}

// ConnectorsList mocks base method.
func (m *MockBackend) ConnectorsList(ctx context.Context, query storage.ListConnectorsQuery) (*bunpaginate.Cursor[models.Connector], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectorsList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.Connector])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConnectorsList indicates an expected call of ConnectorsList.
func (mr *MockBackendMockRecorder) ConnectorsList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectorsList", reflect.TypeOf((*MockBackend)(nil).ConnectorsList), ctx, query)
}

// ConnectorsReset mocks base method.
func (m *MockBackend) ConnectorsReset(ctx context.Context, connectorID models.ConnectorID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectorsReset", ctx, connectorID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConnectorsReset indicates an expected call of ConnectorsReset.
func (mr *MockBackendMockRecorder) ConnectorsReset(ctx, connectorID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectorsReset", reflect.TypeOf((*MockBackend)(nil).ConnectorsReset), ctx, connectorID)
}

// ConnectorsUninstall mocks base method.
func (m *MockBackend) ConnectorsUninstall(ctx context.Context, connectorID models.ConnectorID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectorsUninstall", ctx, connectorID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConnectorsUninstall indicates an expected call of ConnectorsUninstall.
func (mr *MockBackendMockRecorder) ConnectorsUninstall(ctx, connectorID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectorsUninstall", reflect.TypeOf((*MockBackend)(nil).ConnectorsUninstall), ctx, connectorID)
}

// PaymentInitiationAdjustmentsGetLast mocks base method.
func (m *MockBackend) PaymentInitiationAdjustmentsGetLast(ctx context.Context, id models.PaymentInitiationID) (*models.PaymentInitiationAdjustment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationAdjustmentsGetLast", ctx, id)
	ret0, _ := ret[0].(*models.PaymentInitiationAdjustment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentInitiationAdjustmentsGetLast indicates an expected call of PaymentInitiationAdjustmentsGetLast.
func (mr *MockBackendMockRecorder) PaymentInitiationAdjustmentsGetLast(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationAdjustmentsGetLast", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationAdjustmentsGetLast), ctx, id)
}

// PaymentInitiationAdjustmentsList mocks base method.
func (m *MockBackend) PaymentInitiationAdjustmentsList(ctx context.Context, id models.PaymentInitiationID, query storage.ListPaymentInitiationAdjustmentsQuery) (*bunpaginate.Cursor[models.PaymentInitiationAdjustment], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationAdjustmentsList", ctx, id, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.PaymentInitiationAdjustment])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentInitiationAdjustmentsList indicates an expected call of PaymentInitiationAdjustmentsList.
func (mr *MockBackendMockRecorder) PaymentInitiationAdjustmentsList(ctx, id, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationAdjustmentsList", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationAdjustmentsList), ctx, id, query)
}

// PaymentInitiationAdjustmentsListAll mocks base method.
func (m *MockBackend) PaymentInitiationAdjustmentsListAll(ctx context.Context, id models.PaymentInitiationID) ([]models.PaymentInitiationAdjustment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationAdjustmentsListAll", ctx, id)
	ret0, _ := ret[0].([]models.PaymentInitiationAdjustment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentInitiationAdjustmentsListAll indicates an expected call of PaymentInitiationAdjustmentsListAll.
func (mr *MockBackendMockRecorder) PaymentInitiationAdjustmentsListAll(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationAdjustmentsListAll", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationAdjustmentsListAll), ctx, id)
}

// PaymentInitiationRelatedPaymentListAll mocks base method.
func (m *MockBackend) PaymentInitiationRelatedPaymentListAll(ctx context.Context, id models.PaymentInitiationID) ([]models.Payment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationRelatedPaymentListAll", ctx, id)
	ret0, _ := ret[0].([]models.Payment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentInitiationRelatedPaymentListAll indicates an expected call of PaymentInitiationRelatedPaymentListAll.
func (mr *MockBackendMockRecorder) PaymentInitiationRelatedPaymentListAll(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationRelatedPaymentListAll", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationRelatedPaymentListAll), ctx, id)
}

// PaymentInitiationRelatedPaymentsList mocks base method.
func (m *MockBackend) PaymentInitiationRelatedPaymentsList(ctx context.Context, id models.PaymentInitiationID, query storage.ListPaymentInitiationRelatedPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationRelatedPaymentsList", ctx, id, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.Payment])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentInitiationRelatedPaymentsList indicates an expected call of PaymentInitiationRelatedPaymentsList.
func (mr *MockBackendMockRecorder) PaymentInitiationRelatedPaymentsList(ctx, id, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationRelatedPaymentsList", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationRelatedPaymentsList), ctx, id, query)
}

// PaymentInitiationsApprove mocks base method.
func (m *MockBackend) PaymentInitiationsApprove(ctx context.Context, id models.PaymentInitiationID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationsApprove", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// PaymentInitiationsApprove indicates an expected call of PaymentInitiationsApprove.
func (mr *MockBackendMockRecorder) PaymentInitiationsApprove(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationsApprove", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationsApprove), ctx, id)
}

// PaymentInitiationsCreate mocks base method.
func (m *MockBackend) PaymentInitiationsCreate(ctx context.Context, paymentInitiation models.PaymentInitiation, sendToPSP bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationsCreate", ctx, paymentInitiation, sendToPSP)
	ret0, _ := ret[0].(error)
	return ret0
}

// PaymentInitiationsCreate indicates an expected call of PaymentInitiationsCreate.
func (mr *MockBackendMockRecorder) PaymentInitiationsCreate(ctx, paymentInitiation, sendToPSP any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationsCreate", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationsCreate), ctx, paymentInitiation, sendToPSP)
}

// PaymentInitiationsDelete mocks base method.
func (m *MockBackend) PaymentInitiationsDelete(ctx context.Context, id models.PaymentInitiationID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationsDelete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// PaymentInitiationsDelete indicates an expected call of PaymentInitiationsDelete.
func (mr *MockBackendMockRecorder) PaymentInitiationsDelete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationsDelete", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationsDelete), ctx, id)
}

// PaymentInitiationsGet mocks base method.
func (m *MockBackend) PaymentInitiationsGet(ctx context.Context, id models.PaymentInitiationID) (*models.PaymentInitiation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationsGet", ctx, id)
	ret0, _ := ret[0].(*models.PaymentInitiation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentInitiationsGet indicates an expected call of PaymentInitiationsGet.
func (mr *MockBackendMockRecorder) PaymentInitiationsGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationsGet", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationsGet), ctx, id)
}

// PaymentInitiationsList mocks base method.
func (m *MockBackend) PaymentInitiationsList(ctx context.Context, query storage.ListPaymentInitiationsQuery) (*bunpaginate.Cursor[models.PaymentInitiation], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationsList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.PaymentInitiation])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentInitiationsList indicates an expected call of PaymentInitiationsList.
func (mr *MockBackendMockRecorder) PaymentInitiationsList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationsList", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationsList), ctx, query)
}

// PaymentInitiationsReject mocks base method.
func (m *MockBackend) PaymentInitiationsReject(ctx context.Context, id models.PaymentInitiationID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationsReject", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// PaymentInitiationsReject indicates an expected call of PaymentInitiationsReject.
func (mr *MockBackendMockRecorder) PaymentInitiationsReject(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationsReject", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationsReject), ctx, id)
}

// PaymentInitiationsRetry mocks base method.
func (m *MockBackend) PaymentInitiationsRetry(ctx context.Context, id models.PaymentInitiationID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentInitiationsRetry", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// PaymentInitiationsRetry indicates an expected call of PaymentInitiationsRetry.
func (mr *MockBackendMockRecorder) PaymentInitiationsRetry(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentInitiationsRetry", reflect.TypeOf((*MockBackend)(nil).PaymentInitiationsRetry), ctx, id)
}

// PaymentsCreate mocks base method.
func (m *MockBackend) PaymentsCreate(ctx context.Context, payment models.Payment) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentsCreate", ctx, payment)
	ret0, _ := ret[0].(error)
	return ret0
}

// PaymentsCreate indicates an expected call of PaymentsCreate.
func (mr *MockBackendMockRecorder) PaymentsCreate(ctx, payment any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentsCreate", reflect.TypeOf((*MockBackend)(nil).PaymentsCreate), ctx, payment)
}

// PaymentsGet mocks base method.
func (m *MockBackend) PaymentsGet(ctx context.Context, id models.PaymentID) (*models.Payment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentsGet", ctx, id)
	ret0, _ := ret[0].(*models.Payment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentsGet indicates an expected call of PaymentsGet.
func (mr *MockBackendMockRecorder) PaymentsGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentsGet", reflect.TypeOf((*MockBackend)(nil).PaymentsGet), ctx, id)
}

// PaymentsList mocks base method.
func (m *MockBackend) PaymentsList(ctx context.Context, query storage.ListPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentsList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.Payment])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PaymentsList indicates an expected call of PaymentsList.
func (mr *MockBackendMockRecorder) PaymentsList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentsList", reflect.TypeOf((*MockBackend)(nil).PaymentsList), ctx, query)
}

// PaymentsUpdateMetadata mocks base method.
func (m *MockBackend) PaymentsUpdateMetadata(ctx context.Context, id models.PaymentID, metadata map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PaymentsUpdateMetadata", ctx, id, metadata)
	ret0, _ := ret[0].(error)
	return ret0
}

// PaymentsUpdateMetadata indicates an expected call of PaymentsUpdateMetadata.
func (mr *MockBackendMockRecorder) PaymentsUpdateMetadata(ctx, id, metadata any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PaymentsUpdateMetadata", reflect.TypeOf((*MockBackend)(nil).PaymentsUpdateMetadata), ctx, id, metadata)
}

// PoolsAddAccount mocks base method.
func (m *MockBackend) PoolsAddAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PoolsAddAccount", ctx, id, accountID)
	ret0, _ := ret[0].(error)
	return ret0
}

// PoolsAddAccount indicates an expected call of PoolsAddAccount.
func (mr *MockBackendMockRecorder) PoolsAddAccount(ctx, id, accountID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PoolsAddAccount", reflect.TypeOf((*MockBackend)(nil).PoolsAddAccount), ctx, id, accountID)
}

// PoolsBalancesAt mocks base method.
func (m *MockBackend) PoolsBalancesAt(ctx context.Context, poolID uuid.UUID, at time.Time) ([]models.AggregatedBalance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PoolsBalancesAt", ctx, poolID, at)
	ret0, _ := ret[0].([]models.AggregatedBalance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PoolsBalancesAt indicates an expected call of PoolsBalancesAt.
func (mr *MockBackendMockRecorder) PoolsBalancesAt(ctx, poolID, at any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PoolsBalancesAt", reflect.TypeOf((*MockBackend)(nil).PoolsBalancesAt), ctx, poolID, at)
}

// PoolsCreate mocks base method.
func (m *MockBackend) PoolsCreate(ctx context.Context, pool models.Pool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PoolsCreate", ctx, pool)
	ret0, _ := ret[0].(error)
	return ret0
}

// PoolsCreate indicates an expected call of PoolsCreate.
func (mr *MockBackendMockRecorder) PoolsCreate(ctx, pool any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PoolsCreate", reflect.TypeOf((*MockBackend)(nil).PoolsCreate), ctx, pool)
}

// PoolsDelete mocks base method.
func (m *MockBackend) PoolsDelete(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PoolsDelete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// PoolsDelete indicates an expected call of PoolsDelete.
func (mr *MockBackendMockRecorder) PoolsDelete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PoolsDelete", reflect.TypeOf((*MockBackend)(nil).PoolsDelete), ctx, id)
}

// PoolsGet mocks base method.
func (m *MockBackend) PoolsGet(ctx context.Context, id uuid.UUID) (*models.Pool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PoolsGet", ctx, id)
	ret0, _ := ret[0].(*models.Pool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PoolsGet indicates an expected call of PoolsGet.
func (mr *MockBackendMockRecorder) PoolsGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PoolsGet", reflect.TypeOf((*MockBackend)(nil).PoolsGet), ctx, id)
}

// PoolsList mocks base method.
func (m *MockBackend) PoolsList(ctx context.Context, query storage.ListPoolsQuery) (*bunpaginate.Cursor[models.Pool], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PoolsList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.Pool])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PoolsList indicates an expected call of PoolsList.
func (mr *MockBackendMockRecorder) PoolsList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PoolsList", reflect.TypeOf((*MockBackend)(nil).PoolsList), ctx, query)
}

// PoolsRemoveAccount mocks base method.
func (m *MockBackend) PoolsRemoveAccount(ctx context.Context, id uuid.UUID, accountID models.AccountID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PoolsRemoveAccount", ctx, id, accountID)
	ret0, _ := ret[0].(error)
	return ret0
}

// PoolsRemoveAccount indicates an expected call of PoolsRemoveAccount.
func (mr *MockBackendMockRecorder) PoolsRemoveAccount(ctx, id, accountID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PoolsRemoveAccount", reflect.TypeOf((*MockBackend)(nil).PoolsRemoveAccount), ctx, id, accountID)
}

// SchedulesGet mocks base method.
func (m *MockBackend) SchedulesGet(ctx context.Context, id string, connectorID models.ConnectorID) (*models.Schedule, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SchedulesGet", ctx, id, connectorID)
	ret0, _ := ret[0].(*models.Schedule)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SchedulesGet indicates an expected call of SchedulesGet.
func (mr *MockBackendMockRecorder) SchedulesGet(ctx, id, connectorID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SchedulesGet", reflect.TypeOf((*MockBackend)(nil).SchedulesGet), ctx, id, connectorID)
}

// SchedulesList mocks base method.
func (m *MockBackend) SchedulesList(ctx context.Context, query storage.ListSchedulesQuery) (*bunpaginate.Cursor[models.Schedule], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SchedulesList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.Schedule])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SchedulesList indicates an expected call of SchedulesList.
func (mr *MockBackendMockRecorder) SchedulesList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SchedulesList", reflect.TypeOf((*MockBackend)(nil).SchedulesList), ctx, query)
}

// TaskGet mocks base method.
func (m *MockBackend) TaskGet(ctx context.Context, id models.TaskID) (*models.Task, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TaskGet", ctx, id)
	ret0, _ := ret[0].(*models.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TaskGet indicates an expected call of TaskGet.
func (mr *MockBackendMockRecorder) TaskGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TaskGet", reflect.TypeOf((*MockBackend)(nil).TaskGet), ctx, id)
}

// WorkflowsInstancesList mocks base method.
func (m *MockBackend) WorkflowsInstancesList(ctx context.Context, query storage.ListInstancesQuery) (*bunpaginate.Cursor[models.Instance], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WorkflowsInstancesList", ctx, query)
	ret0, _ := ret[0].(*bunpaginate.Cursor[models.Instance])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WorkflowsInstancesList indicates an expected call of WorkflowsInstancesList.
func (mr *MockBackendMockRecorder) WorkflowsInstancesList(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WorkflowsInstancesList", reflect.TypeOf((*MockBackend)(nil).WorkflowsInstancesList), ctx, query)
}
