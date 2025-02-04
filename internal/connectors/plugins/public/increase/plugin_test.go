package increase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockClient struct {
	mock.Mock
}

func (m *mockClient) GetAccounts(ctx context.Context, lastID string, pageSize int64) ([]*client.Account, string, bool, error) {
	args := m.Called(ctx, lastID, pageSize)
	return args.Get(0).([]*client.Account), args.String(1), args.Bool(2), args.Error(3)
}

func (m *mockClient) GetAccountBalances(ctx context.Context, accountID string) ([]*client.Balance, error) {
	args := m.Called(ctx, accountID)
	return args.Get(0).([]*client.Balance), args.Error(1)
}

func (m *mockClient) GetTransactions(ctx context.Context, lastID string, pageSize int64) ([]*client.Transaction, string, bool, error) {
	args := m.Called(ctx, lastID, pageSize)
	return args.Get(0).([]*client.Transaction), args.String(1), args.Bool(2), args.Error(3)
}

func (m *mockClient) GetPendingTransactions(ctx context.Context, lastID string, pageSize int64) ([]*client.Transaction, string, bool, error) {
	args := m.Called(ctx, lastID, pageSize)
	return args.Get(0).([]*client.Transaction), args.String(1), args.Bool(2), args.Error(3)
}

func (m *mockClient) GetDeclinedTransactions(ctx context.Context, lastID string, pageSize int64) ([]*client.Transaction, string, bool, error) {
	args := m.Called(ctx, lastID, pageSize)
	return args.Get(0).([]*client.Transaction), args.String(1), args.Bool(2), args.Error(3)
}

func (m *mockClient) GetExternalAccounts(ctx context.Context, lastID string, pageSize int64) ([]*client.ExternalAccount, string, bool, error) {
	args := m.Called(ctx, lastID, pageSize)
	return args.Get(0).([]*client.ExternalAccount), args.String(1), args.Bool(2), args.Error(3)
}

func (m *mockClient) CreateExternalAccount(ctx context.Context, req *client.CreateExternalAccountRequest) (*client.ExternalAccount, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.ExternalAccount), args.Error(1)
}

func (m *mockClient) CreateTransfer(ctx context.Context, req *client.CreateTransferRequest) (*client.Transfer, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.Transfer), args.Error(1)
}

func (m *mockClient) CreateACHTransfer(ctx context.Context, req *client.CreateACHTransferRequest) (*client.Transfer, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.Transfer), args.Error(1)
}

func (m *mockClient) CreateWireTransfer(ctx context.Context, req *client.CreateWireTransferRequest) (*client.Transfer, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.Transfer), args.Error(1)
}

func (m *mockClient) CreateCheckTransfer(ctx context.Context, req *client.CreateCheckTransferRequest) (*client.Transfer, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.Transfer), args.Error(1)
}

func (m *mockClient) CreateRTPTransfer(ctx context.Context, req *client.CreateRTPTransferRequest) (*client.Transfer, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*client.Transfer), args.Error(1)
}

func TestPlugin_Install(t *testing.T) {
	p := NewPlugin()

	config := Config{
		APIKey:        "test_key",
		PollingPeriod: 30 * time.Second,
	}
	configBytes, err := json.Marshal(config)
	assert.NoError(t, err)

	_, err = p.Install(context.Background(), models.InstallRequest{
		Config: configBytes,
	})
	assert.NoError(t, err)
	assert.NotNil(t, p.client)
	assert.Equal(t, &config, p.config)
}

func TestPlugin_Install_InvalidConfig(t *testing.T) {
	p := NewPlugin()

	config := Config{
		APIKey:        "",
		PollingPeriod: 1 * time.Second,
	}
	configBytes, err := json.Marshal(config)
	assert.NoError(t, err)

	_, err = p.Install(context.Background(), models.InstallRequest{
		Config: configBytes,
	})
	assert.Error(t, err)
}

func TestPlugin_CreatePayout(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		mockSetup func(*mockClient)
		wantErr  bool
	}{
		{
			name: "ach payout",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeACH,
				"business_type":       "business",
			},
			mockSetup: func(m *mockClient) {
				m.On("CreateACHTransfer", mock.Anything, &client.CreateACHTransferRequest{
					CreateTransferRequest: client.CreateTransferRequest{
						AccountID:   "acc_123",
						Amount:      100,
						Description: "test payout",
					},
					StandardEntryClassCode: SECCodeCCD,
				}).Return(&client.Transfer{
					ID:        "transfer_123",
					CreatedAt: time.Now(),
					Status:    "pending",
					Type:      "ach",
					Amount:    100,
					Currency:  "USD",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "wire payout",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeWire,
			},
			mockSetup: func(m *mockClient) {
				m.On("CreateWireTransfer", mock.Anything, &client.CreateWireTransferRequest{
					CreateTransferRequest: client.CreateTransferRequest{
						AccountID:   "acc_123",
						Amount:      100,
						Description: "test payout",
					},
					MessageToRecipient: "test payout",
				}).Return(&client.Transfer{
					ID:        "transfer_123",
					CreatedAt: time.Now(),
					Status:    "pending",
					Type:      "wire",
					Amount:    100,
					Currency:  "USD",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "check payout",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeCheck,
				"check_memo":          "test memo",
			},
			mockSetup: func(m *mockClient) {
				m.On("CreateCheckTransfer", mock.Anything, &client.CreateCheckTransferRequest{
					CreateTransferRequest: client.CreateTransferRequest{
						AccountID:   "acc_123",
						Amount:      100,
						Description: "test payout",
					},
					PhysicalCheck: client.PhysicalCheck{
						Memo: "test memo",
					},
				}).Return(&client.Transfer{
					ID:        "transfer_123",
					CreatedAt: time.Now(),
					Status:    "pending",
					Type:      "check",
					Amount:    100,
					Currency:  "USD",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "rtp payout",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeRTP,
			},
			mockSetup: func(m *mockClient) {
				m.On("CreateRTPTransfer", mock.Anything, &client.CreateRTPTransferRequest{
					CreateTransferRequest: client.CreateTransferRequest{
						AccountID:   "acc_123",
						Amount:      100,
						Description: "test payout",
					},
				}).Return(&client.Transfer{
					ID:        "transfer_123",
					CreatedAt: time.Now(),
					Status:    "pending",
					Type:      "rtp",
					Amount:    100,
					Currency:  "USD",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "missing payout type",
			metadata: map[string]string{
				"foo": "bar",
			},
			mockSetup: func(m *mockClient) {},
			wantErr:   true,
		},
		{
			name: "invalid payout type",
			metadata: map[string]string{
				PayoutTypeMetadataKey: "invalid",
			},
			mockSetup: func(m *mockClient) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockClient{}
			p := NewPlugin()
			p.client = mockClient

			tt.mockSetup(mockClient)

			_, err := p.CreatePayout(context.Background(), models.CreatePayoutRequest{
				PaymentInitiation: models.PSPPaymentInitiation{
					SourceAccountID: "acc_123",
					Amount:         100,
					Currency:       "USD",
					Description:    "test payout",
					Metadata:       tt.metadata,
				},
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				mockClient.AssertExpectations(t)
			}
		})
	}
}

func TestPlugin_FetchNextAccounts(t *testing.T) {
	mockClient := &mockClient{}
	p := NewPlugin()
	p.client = mockClient

	accounts := []*client.Account{
		{
			ID:        "acc_123",
			Name:      "Test Account",
			Status:    "active",
			Type:      "checking",
			Currency:  "USD",
			CreatedAt: time.Now(),
		},
	}

	mockClient.On("GetAccounts", mock.Anything, "", int64(10)).Return(accounts, "next_cursor", true, nil)

	resp, err := p.FetchNextAccounts(context.Background(), models.FetchNextAccountsRequest{
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Accounts, 1)
	assert.Equal(t, accounts[0].ID, resp.Accounts[0].ID)
	assert.True(t, resp.HasMore)
	mockClient.AssertExpectations(t)
}

func TestPlugin_FetchNextBalances(t *testing.T) {
	mockClient := &mockClient{}
	p := NewPlugin()
	p.client = mockClient

	balances := []*client.Balance{
		{
			Available: 1000,
			Currency:  "USD",
		},
	}

	mockClient.On("GetAccountBalances", mock.Anything, "acc_123").Return(balances, nil)

	fromPayload := json.RawMessage(`{"accountID": "acc_123"}`)
	resp, err := p.FetchNextBalances(context.Background(), models.FetchNextBalancesRequest{
		FromPayload: fromPayload,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Balances, 1)
	assert.Equal(t, balances[0].Available, resp.Balances[0].Amount)
	assert.Equal(t, balances[0].Currency, resp.Balances[0].Currency)
	mockClient.AssertExpectations(t)
}

func TestPlugin_FetchNextExternalAccounts(t *testing.T) {
	mockClient := &mockClient{}
	p := NewPlugin()
	p.client = mockClient

	externalAccounts := []*client.ExternalAccount{
		{
			ID:            "ext_123",
			Name:          "Test External Account",
			AccountNumber: "123456789",
			RoutingNumber: "987654321",
			Status:        "active",
			Type:          "checking",
		},
	}

	mockClient.On("GetExternalAccounts", mock.Anything, "", int64(10)).Return(externalAccounts, "next_cursor", true, nil)

	resp, err := p.FetchNextExternalAccounts(context.Background(), models.FetchNextExternalAccountsRequest{
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.ExternalAccounts, 1)
	assert.Equal(t, externalAccounts[0].ID, resp.ExternalAccounts[0].ID)
	assert.True(t, resp.HasMore)
	mockClient.AssertExpectations(t)
}

func TestPlugin_CreateBankAccount(t *testing.T) {
	mockClient := &mockClient{}
	p := NewPlugin()
	p.client = mockClient

	bankAccount := &client.ExternalAccount{
		ID:            "ext_123",
		Name:          "Test Account",
		AccountNumber: "123456789",
		RoutingNumber: "987654321",
		Status:        "active",
		Type:          "checking",
	}

	mockClient.On("CreateExternalAccount", mock.Anything, &client.CreateExternalAccountRequest{
		Name:          "Test Account",
		AccountNumber: "123456789",
		RoutingNumber: "987654321",
	}).Return(bankAccount, nil)

	resp, err := p.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{
		BankAccount: models.PSPBankAccount{
			Name:          "Test Account",
			AccountNumber: "123456789",
			RoutingNumber: "987654321",
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp.RelatedAccount)
	assert.Equal(t, bankAccount.ID, resp.RelatedAccount.ID)
	mockClient.AssertExpectations(t)
}

func TestPlugin_TranslateWebhook(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		data      interface{}
		validate  func(*testing.T, models.WebhookResponse)
		wantErr   bool
	}{
		{
			name:      "account created",
			eventType: webhookTypeAccountCreated,
			data: &client.Account{
				ID:        "acc_123",
				Name:      "Test Account",
				Status:    "active",
				Type:      "checking",
				Currency:  "USD",
				CreatedAt: time.Now(),
			},
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.NotNil(t, resp.Account)
				assert.Equal(t, "acc_123", resp.Account.ID)
				assert.Equal(t, models.AccountTypeChecking, resp.Account.Type)
				assert.Equal(t, models.AccountStatusActive, resp.Account.Status)
			},
			wantErr: false,
		},
		{
			name:      "transaction created",
			eventType: webhookTypeTransactionCreated,
			data: &client.Transaction{
				ID:        "txn_123",
				Status:    "pending",
				Type:      "ach",
				Amount:    100,
				Currency:  "USD",
				CreatedAt: time.Now(),
			},
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.NotNil(t, resp.Payment)
				assert.Equal(t, "txn_123", resp.Payment.ID)
				assert.Equal(t, models.PaymentStatusPending, resp.Payment.Status)
				assert.Equal(t, models.PaymentTypeACH, resp.Payment.Type)
				assert.Equal(t, int64(100), resp.Payment.Amount)
				assert.Equal(t, "USD", resp.Payment.Currency)
			},
			wantErr: false,
		},
		{
			name:      "transfer created",
			eventType: webhookTypeTransferCreated,
			data: &client.Transfer{
				ID:        "transfer_123",
				Status:    "pending",
				Type:      "wire",
				Amount:    200,
				Currency:  "USD",
				CreatedAt: time.Now(),
			},
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.NotNil(t, resp.Payment)
				assert.Equal(t, "transfer_123", resp.Payment.ID)
				assert.Equal(t, models.PaymentStatusPending, resp.Payment.Status)
				assert.Equal(t, models.PaymentTypeWire, resp.Payment.Type)
				assert.Equal(t, int64(200), resp.Payment.Amount)
				assert.Equal(t, "USD", resp.Payment.Currency)
			},
			wantErr: false,
		},
		{
			name:      "invalid webhook data",
			eventType: webhookTypeAccountCreated,
			data:      "invalid",
			wantErr:   true,
		},
		{
			name:      "unknown event type",
			eventType: "unknown",
			data:      struct{}{},
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.Nil(t, resp.Account)
				assert.Nil(t, resp.Payment)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlugin()

			data, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			webhookEvent := client.WebhookEvent{
				ID:        "evt_123",
				Type:      tt.eventType,
				CreatedAt: time.Now(),
				Data:      data,
			}
			webhookEventBytes, err := json.Marshal(webhookEvent)
			assert.NoError(t, err)

			resp, err := p.TranslateWebhook(context.Background(), models.TranslateWebhookRequest{
				Webhook: models.PSPWebhook{
					Raw: webhookEventBytes,
				},
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, webhookEvent.ID, resp.Responses[0].IdempotencyKey)
			if tt.validate != nil {
				tt.validate(t, resp.Responses[0])
			}
		})
	}
}
