// Code generated by MockGen. DO NOT EDIT.
// Source: client.go
//
// Generated by this command:
//
//	mockgen -source client.go -destination client_generated.go -package client . Client
//

// Package client is a generated GoMock package.
package client

import (
	context "context"
	reflect "reflect"
	time "time"

	genericclient "github.com/formancehq/payments/genericclient"
	gomock "go.uber.org/mock/gomock"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
	isgomock struct{}
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// GetBalances mocks base method.
func (m *MockClient) GetBalances(ctx context.Context, accountID string) (*genericclient.Balances, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalances", ctx, accountID)
	ret0, _ := ret[0].(*genericclient.Balances)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBalances indicates an expected call of GetBalances.
func (mr *MockClientMockRecorder) GetBalances(ctx, accountID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalances", reflect.TypeOf((*MockClient)(nil).GetBalances), ctx, accountID)
}

// ListAccounts mocks base method.
func (m *MockClient) ListAccounts(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Account, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAccounts", ctx, page, pageSize, createdAtFrom)
	ret0, _ := ret[0].([]genericclient.Account)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAccounts indicates an expected call of ListAccounts.
func (mr *MockClientMockRecorder) ListAccounts(ctx, page, pageSize, createdAtFrom any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAccounts", reflect.TypeOf((*MockClient)(nil).ListAccounts), ctx, page, pageSize, createdAtFrom)
}

// ListBeneficiaries mocks base method.
func (m *MockClient) ListBeneficiaries(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Beneficiary, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListBeneficiaries", ctx, page, pageSize, createdAtFrom)
	ret0, _ := ret[0].([]genericclient.Beneficiary)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListBeneficiaries indicates an expected call of ListBeneficiaries.
func (mr *MockClientMockRecorder) ListBeneficiaries(ctx, page, pageSize, createdAtFrom any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListBeneficiaries", reflect.TypeOf((*MockClient)(nil).ListBeneficiaries), ctx, page, pageSize, createdAtFrom)
}

// ListTransactions mocks base method.
func (m *MockClient) ListTransactions(ctx context.Context, page, pageSize int64, updatedAtFrom time.Time) ([]genericclient.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListTransactions", ctx, page, pageSize, updatedAtFrom)
	ret0, _ := ret[0].([]genericclient.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListTransactions indicates an expected call of ListTransactions.
func (mr *MockClientMockRecorder) ListTransactions(ctx, page, pageSize, updatedAtFrom any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListTransactions", reflect.TypeOf((*MockClient)(nil).ListTransactions), ctx, page, pageSize, updatedAtFrom)
}