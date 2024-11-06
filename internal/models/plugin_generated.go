// Code generated by MockGen. DO NOT EDIT.
// Source: plugin.go
//
// Generated by this command:
//
//	mockgen -source plugin.go -destination plugin_generated.go -package models . Plugin
//

// Package models is a generated GoMock package.
package models

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockPlugin is a mock of Plugin interface.
type MockPlugin struct {
	ctrl     *gomock.Controller
	recorder *MockPluginMockRecorder
}

// MockPluginMockRecorder is the mock recorder for MockPlugin.
type MockPluginMockRecorder struct {
	mock *MockPlugin
}

// NewMockPlugin creates a new mock instance.
func NewMockPlugin(ctrl *gomock.Controller) *MockPlugin {
	mock := &MockPlugin{ctrl: ctrl}
	mock.recorder = &MockPluginMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPlugin) EXPECT() *MockPluginMockRecorder {
	return m.recorder
}

// CreateBankAccount mocks base method.
func (m *MockPlugin) CreateBankAccount(arg0 context.Context, arg1 CreateBankAccountRequest) (CreateBankAccountResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBankAccount", arg0, arg1)
	ret0, _ := ret[0].(CreateBankAccountResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBankAccount indicates an expected call of CreateBankAccount.
func (mr *MockPluginMockRecorder) CreateBankAccount(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBankAccount", reflect.TypeOf((*MockPlugin)(nil).CreateBankAccount), arg0, arg1)
}

// CreatePayout mocks base method.
func (m *MockPlugin) CreatePayout(arg0 context.Context, arg1 CreatePayoutRequest) (CreatePayoutResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePayout", arg0, arg1)
	ret0, _ := ret[0].(CreatePayoutResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreatePayout indicates an expected call of CreatePayout.
func (mr *MockPluginMockRecorder) CreatePayout(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePayout", reflect.TypeOf((*MockPlugin)(nil).CreatePayout), arg0, arg1)
}

// CreateTransfer mocks base method.
func (m *MockPlugin) CreateTransfer(arg0 context.Context, arg1 CreateTransferRequest) (CreateTransferResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTransfer", arg0, arg1)
	ret0, _ := ret[0].(CreateTransferResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTransfer indicates an expected call of CreateTransfer.
func (mr *MockPluginMockRecorder) CreateTransfer(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTransfer", reflect.TypeOf((*MockPlugin)(nil).CreateTransfer), arg0, arg1)
}

// CreateWebhooks mocks base method.
func (m *MockPlugin) CreateWebhooks(arg0 context.Context, arg1 CreateWebhooksRequest) (CreateWebhooksResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateWebhooks", arg0, arg1)
	ret0, _ := ret[0].(CreateWebhooksResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateWebhooks indicates an expected call of CreateWebhooks.
func (mr *MockPluginMockRecorder) CreateWebhooks(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateWebhooks", reflect.TypeOf((*MockPlugin)(nil).CreateWebhooks), arg0, arg1)
}

// FetchNextAccounts mocks base method.
func (m *MockPlugin) FetchNextAccounts(arg0 context.Context, arg1 FetchNextAccountsRequest) (FetchNextAccountsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchNextAccounts", arg0, arg1)
	ret0, _ := ret[0].(FetchNextAccountsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchNextAccounts indicates an expected call of FetchNextAccounts.
func (mr *MockPluginMockRecorder) FetchNextAccounts(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchNextAccounts", reflect.TypeOf((*MockPlugin)(nil).FetchNextAccounts), arg0, arg1)
}

// FetchNextBalances mocks base method.
func (m *MockPlugin) FetchNextBalances(arg0 context.Context, arg1 FetchNextBalancesRequest) (FetchNextBalancesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchNextBalances", arg0, arg1)
	ret0, _ := ret[0].(FetchNextBalancesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchNextBalances indicates an expected call of FetchNextBalances.
func (mr *MockPluginMockRecorder) FetchNextBalances(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchNextBalances", reflect.TypeOf((*MockPlugin)(nil).FetchNextBalances), arg0, arg1)
}

// FetchNextExternalAccounts mocks base method.
func (m *MockPlugin) FetchNextExternalAccounts(arg0 context.Context, arg1 FetchNextExternalAccountsRequest) (FetchNextExternalAccountsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchNextExternalAccounts", arg0, arg1)
	ret0, _ := ret[0].(FetchNextExternalAccountsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchNextExternalAccounts indicates an expected call of FetchNextExternalAccounts.
func (mr *MockPluginMockRecorder) FetchNextExternalAccounts(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchNextExternalAccounts", reflect.TypeOf((*MockPlugin)(nil).FetchNextExternalAccounts), arg0, arg1)
}

// FetchNextOthers mocks base method.
func (m *MockPlugin) FetchNextOthers(arg0 context.Context, arg1 FetchNextOthersRequest) (FetchNextOthersResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchNextOthers", arg0, arg1)
	ret0, _ := ret[0].(FetchNextOthersResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchNextOthers indicates an expected call of FetchNextOthers.
func (mr *MockPluginMockRecorder) FetchNextOthers(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchNextOthers", reflect.TypeOf((*MockPlugin)(nil).FetchNextOthers), arg0, arg1)
}

// FetchNextPayments mocks base method.
func (m *MockPlugin) FetchNextPayments(arg0 context.Context, arg1 FetchNextPaymentsRequest) (FetchNextPaymentsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchNextPayments", arg0, arg1)
	ret0, _ := ret[0].(FetchNextPaymentsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchNextPayments indicates an expected call of FetchNextPayments.
func (mr *MockPluginMockRecorder) FetchNextPayments(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchNextPayments", reflect.TypeOf((*MockPlugin)(nil).FetchNextPayments), arg0, arg1)
}

// Install mocks base method.
func (m *MockPlugin) Install(arg0 context.Context, arg1 InstallRequest) (InstallResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Install", arg0, arg1)
	ret0, _ := ret[0].(InstallResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Install indicates an expected call of Install.
func (mr *MockPluginMockRecorder) Install(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Install", reflect.TypeOf((*MockPlugin)(nil).Install), arg0, arg1)
}

// Name mocks base method.
func (m *MockPlugin) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockPluginMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockPlugin)(nil).Name))
}

// PollPayoutStatus mocks base method.
func (m *MockPlugin) PollPayoutStatus(arg0 context.Context, arg1 PollPayoutStatusRequest) (PollPayoutStatusResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PollPayoutStatus", arg0, arg1)
	ret0, _ := ret[0].(PollPayoutStatusResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PollPayoutStatus indicates an expected call of PollPayoutStatus.
func (mr *MockPluginMockRecorder) PollPayoutStatus(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PollPayoutStatus", reflect.TypeOf((*MockPlugin)(nil).PollPayoutStatus), arg0, arg1)
}

// PollTransferStatus mocks base method.
func (m *MockPlugin) PollTransferStatus(arg0 context.Context, arg1 PollTransferStatusRequest) (PollTransferStatusResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PollTransferStatus", arg0, arg1)
	ret0, _ := ret[0].(PollTransferStatusResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PollTransferStatus indicates an expected call of PollTransferStatus.
func (mr *MockPluginMockRecorder) PollTransferStatus(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PollTransferStatus", reflect.TypeOf((*MockPlugin)(nil).PollTransferStatus), arg0, arg1)
}

// TranslateWebhook mocks base method.
func (m *MockPlugin) TranslateWebhook(arg0 context.Context, arg1 TranslateWebhookRequest) (TranslateWebhookResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TranslateWebhook", arg0, arg1)
	ret0, _ := ret[0].(TranslateWebhookResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TranslateWebhook indicates an expected call of TranslateWebhook.
func (mr *MockPluginMockRecorder) TranslateWebhook(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TranslateWebhook", reflect.TypeOf((*MockPlugin)(nil).TranslateWebhook), arg0, arg1)
}

// Uninstall mocks base method.
func (m *MockPlugin) Uninstall(arg0 context.Context, arg1 UninstallRequest) (UninstallResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Uninstall", arg0, arg1)
	ret0, _ := ret[0].(UninstallResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Uninstall indicates an expected call of Uninstall.
func (mr *MockPluginMockRecorder) Uninstall(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Uninstall", reflect.TypeOf((*MockPlugin)(nil).Uninstall), arg0, arg1)
}