package increase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/Increase/increase-go"
)

type mockIncrease struct {
	mock.Mock
}

func (m *mockIncrease) Accounts() *increase.AccountService {
	return &increase.AccountService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) Balances() *increase.BalanceService {
	return &increase.BalanceService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) Transactions() *increase.TransactionService {
	return &increase.TransactionService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) ExternalAccounts() *increase.ExternalAccountService {
	return &increase.ExternalAccountService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) AccountTransfers() *increase.AccountTransferService {
	return &increase.AccountTransferService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) ACHTransfers() *increase.ACHTransferService {
	return &increase.ACHTransferService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) WireTransfers() *increase.WireTransferService {
	return &increase.WireTransferService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) CheckTransfers() *increase.CheckTransferService {
	return &increase.CheckTransferService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) RealTimePaymentsTransfers() *increase.RealTimePaymentsTransferService {
	return &increase.RealTimePaymentsTransferService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) EventSubscriptions() *increase.EventSubscriptionService {
	return &increase.EventSubscriptionService{
		Client: &increase.Client{},
	}
}

func (m *mockIncrease) Webhooks() *increase.WebhookService {
	return &increase.WebhookService{
		Client: &increase.Client{},
	}
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

func TestPlugin_DeterminePayoutType(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		want     string
		wantErr  bool
	}{
		{
			name: "valid ach payout type",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeACH,
			},
			want:    PayoutTypeACH,
			wantErr: false,
		},
		{
			name: "valid wire payout type",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeWire,
			},
			want:    PayoutTypeWire,
			wantErr: false,
		},
		{
			name: "valid check payout type",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeCheck,
			},
			want:    PayoutTypeCheck,
			wantErr: false,
		},
		{
			name: "valid rtp payout type",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeRTP,
			},
			want:    PayoutTypeRTP,
			wantErr: false,
		},
		{
			name:     "missing payout type",
			metadata: map[string]string{},
			wantErr:  true,
		},
		{
			name: "invalid payout type",
			metadata: map[string]string{
				PayoutTypeMetadataKey: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlugin()
			got, err := p.determinePayoutType(tt.metadata)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlugin_CreatePayout(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]string
		mockSetup func(*mockIncrease)
		wantErr  bool
	}{
		{
			name: "ach payout",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeACH,
				"business_type":       "business",
			},
			mockSetup: func(m *mockIncrease) {
				m.On("ACHTransfers").Return(&increase.ACHTransferService{
					Client: &increase.Client{},
				})
				m.On("New", mock.Anything, &increase.ACHTransferCreateParams{
					AccountID:              "acc_123",
					Amount:                100,
					Description:           increase.F("test payout"),
					RequireApproval:       increase.F(false),
					StandardEntryClassCode: increase.F(string(increase.ACHTransferStandardEntryClassCodeCCD)),
				}).Return(&increase.ACHTransfer{
					ID:        "transfer_123",
					CreatedAt: time.Now(),
					Status:    increase.ACHTransferStatusPending,
					Type:      increase.ACHTransferTypeCreditCCD,
					Amount:    100,
					Currency:  increase.CurrencyUSD,
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "wire payout",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeWire,
			},
			mockSetup: func(m *mockIncrease) {
				m.On("WireTransfers").Return(&increase.WireTransferService{
					Client: &increase.Client{},
				})
				m.On("New", mock.Anything, &increase.WireTransferCreateParams{
					AccountID:          "acc_123",
					Amount:            100,
					Description:       increase.F("test payout"),
					RequireApproval:   increase.F(false),
					MessageToRecipient: increase.F("test payout"),
				}).Return(&increase.WireTransfer{
					ID:        "transfer_123",
					CreatedAt: time.Now(),
					Status:    increase.WireTransferStatusPending,
					Type:      increase.WireTransferTypeWire,
					Amount:    100,
					Currency:  increase.CurrencyUSD,
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
			mockSetup: func(m *mockIncrease) {
				m.On("CheckTransfers").Return(&increase.CheckTransferService{
					Client: &increase.Client{},
				})
				m.On("New", mock.Anything, &increase.CheckTransferCreateParams{
					AccountID:          "acc_123",
					Amount:            100,
					Description:       increase.F("test payout"),
					RequireApproval:   increase.F(false),
					PhysicalCheck: &increase.CheckTransferCreateParamsPhysicalCheck{
						Memo: increase.F("test memo"),
					},
				}).Return(&increase.CheckTransfer{
					ID:        "transfer_123",
					CreatedAt: time.Now(),
					Status:    increase.CheckTransferStatusPending,
					Type:      increase.CheckTransferTypeCheck,
					Amount:    100,
					Currency:  increase.CurrencyUSD,
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "rtp payout",
			metadata: map[string]string{
				PayoutTypeMetadataKey: PayoutTypeRTP,
			},
			mockSetup: func(m *mockIncrease) {
				m.On("RealTimePaymentsTransfers").Return(&increase.RealTimePaymentsTransferService{
					Client: &increase.Client{},
				})
				m.On("New", mock.Anything, &increase.RealTimePaymentsTransferCreateParams{
					AccountID:          "acc_123",
					Amount:            100,
					Description:       increase.F("test payout"),
					RequireApproval:   increase.F(false),
				}).Return(&increase.RealTimePaymentsTransfer{
					ID:        "transfer_123",
					CreatedAt: time.Now(),
					Status:    increase.RealTimePaymentsTransferStatusPending,
					Type:      increase.RealTimePaymentsTransferTypeRealTimePayments,
					Amount:    100,
					Currency:  increase.CurrencyUSD,
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "missing payout type",
			metadata: map[string]string{
				"foo": "bar",
			},
			mockSetup: func(m *mockIncrease) {},
			wantErr:   true,
		},
		{
			name: "invalid payout type",
			metadata: map[string]string{
				PayoutTypeMetadataKey: "invalid",
			},
			mockSetup: func(m *mockIncrease) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSDK := &mockIncrease{}
			p := NewPlugin()
			p.client = mockSDK

			tt.mockSetup(mockSDK)

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
				mockSDK.AssertExpectations(t)
			}
		})
	}
}

func TestPlugin_FetchNextAccounts(t *testing.T) {
	mockSDK := &mockIncrease{}
	p := NewPlugin()
	p.client = mockSDK

	createdAt := time.Now()
	mockSDK.On("Accounts").Return(&increase.AccountService{
		Client: &increase.Client{},
	})
	mockSDK.On("List", mock.Anything, &increase.AccountListParams{
		Limit: increase.F(int32(10)),
	}).Return(&increase.AccountList{
		Data: []*increase.Account{
			{
				ID:        "acc_123",
				Name:      "Test Account",
				Status:    increase.AccountStatusActive,
				Type:      increase.AccountTypeChecking,
				Currency:  increase.CurrencyUSD,
				Bank:      increase.AccountBankIncrease,
				CreatedAt: createdAt,
			},
		},
		NextCursor: "next_cursor",
		HasMore:    true,
	}, nil)

	resp, err := p.FetchNextAccounts(context.Background(), models.FetchNextAccountsRequest{
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Accounts, 1)
	assert.Equal(t, "acc_123", resp.Accounts[0].Reference)
	assert.Equal(t, "USD", *resp.Accounts[0].DefaultAsset)
	assert.Equal(t, "Test Account", *resp.Accounts[0].Name)
	assert.Equal(t, string(increase.AccountStatusActive), resp.Accounts[0].Metadata["status"])
	assert.Equal(t, string(increase.AccountTypeChecking), resp.Accounts[0].Metadata["type"])
	assert.Equal(t, string(increase.AccountBankIncrease), resp.Accounts[0].Metadata["bank"])
	assert.Equal(t, string(increase.CurrencyUSD), resp.Accounts[0].Metadata["currency"])
	assert.True(t, resp.HasMore)
	mockSDK.AssertExpectations(t)
}

func TestPlugin_FetchNextBalances(t *testing.T) {
	mockSDK := &mockIncrease{}
	p := NewPlugin()
	p.client = mockSDK

	mockSDK.On("Balances").Return(&increase.BalanceService{
		Client: &increase.Client{},
	})
	mockSDK.On("Get", mock.Anything, "acc_123").Return(&increase.Balance{
		Available: increase.Amount{
			MinorUnits: 1000,
		},
		Currency: increase.CurrencyUSD,
	}, nil)

	fromPayload := json.RawMessage(`{"accountID": "acc_123"}`)
	resp, err := p.FetchNextBalances(context.Background(), models.FetchNextBalancesRequest{
		FromPayload: fromPayload,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Balances, 1)
	assert.Equal(t, int64(1000), resp.Balances[0].Amount)
	assert.Equal(t, string(increase.CurrencyUSD), resp.Balances[0].Currency)
	mockSDK.AssertExpectations(t)
}

func TestPlugin_FetchNextExternalAccounts(t *testing.T) {
	mockSDK := &mockIncrease{}
	p := NewPlugin()
	p.client = mockSDK

	mockSDK.On("ExternalAccounts").Return(&increase.ExternalAccountService{
		Client: &increase.Client{},
	})
	mockSDK.On("List", mock.Anything, &increase.ExternalAccountListParams{
		Limit: increase.F(int32(10)),
	}).Return(&increase.ExternalAccountList{
		Data: []*increase.ExternalAccount{
			{
				ID:            "ext_123",
				Name:          "Test External Account",
				AccountNumber: "123456789",
				RoutingNumber: "987654321",
				Status:        increase.ExternalAccountStatusActive,
				Type:          increase.ExternalAccountTypeChecking,
			},
		},
		NextCursor: "next_cursor",
		HasMore:    true,
	}, nil)

	resp, err := p.FetchNextExternalAccounts(context.Background(), models.FetchNextExternalAccountsRequest{
		PageSize: 10,
	})
	assert.NoError(t, err)
	assert.Len(t, resp.ExternalAccounts, 1)
	assert.Equal(t, "ext_123", resp.ExternalAccounts[0].ID)
	assert.True(t, resp.HasMore)
	mockSDK.AssertExpectations(t)
}

func TestPlugin_CreateBankAccount(t *testing.T) {
	mockSDK := &mockIncrease{}
	p := NewPlugin()
	p.client = mockSDK

	mockSDK.On("ExternalAccounts").Return(&increase.ExternalAccountService{
		Client: &increase.Client{},
	})
	mockSDK.On("New", mock.Anything, &increase.ExternalAccountCreateParams{
		Name:          "Test Account",
		AccountNumber: "123456789",
		RoutingNumber: "987654321",
	}).Return(&increase.ExternalAccount{
		ID:            "ext_123",
		Name:          "Test Account",
		AccountNumber: "123456789",
		RoutingNumber: "987654321",
		Status:        increase.ExternalAccountStatusActive,
		Type:          increase.ExternalAccountTypeChecking,
	}, nil)

	resp, err := p.CreateBankAccount(context.Background(), models.CreateBankAccountRequest{
		BankAccount: models.PSPBankAccount{
			Name:          "Test Account",
			AccountNumber: "123456789",
			RoutingNumber: "987654321",
		},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp.RelatedAccount)
	assert.Equal(t, "ext_123", resp.RelatedAccount.ID)
	mockSDK.AssertExpectations(t)
}

func TestPlugin_TranslateWebhook(t *testing.T) {
	mockSDK := &mockIncrease{}

	tests := []struct {
		name      string
		eventType string
		data      interface{}
		signature string
		validate  func(*testing.T, models.WebhookResponse)
		wantErr   bool
	}{
		{
			name:      "account created",
			eventType: webhookTypeAccountCreated,
			data: &increase.Account{
				ID:        "acc_123",
				Name:      "Test Account",
				Status:    increase.AccountStatusActive,
				Type:      increase.AccountTypeChecking,
				Currency:  increase.CurrencyUSD,
				Bank:      increase.AccountBankIncrease,
				CreatedAt: time.Now(),
			},
			signature: "whsig_test_123",
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.NotNil(t, resp.Account)
				assert.Equal(t, "acc_123", resp.Account.Reference)
				assert.Equal(t, string(increase.CurrencyUSD), *resp.Account.DefaultAsset)
				assert.Equal(t, "Test Account", *resp.Account.Name)
				assert.Equal(t, string(increase.AccountStatusActive), resp.Account.Metadata["status"])
				assert.Equal(t, string(increase.AccountTypeChecking), resp.Account.Metadata["type"])
				assert.Equal(t, string(increase.AccountBankIncrease), resp.Account.Metadata["bank"])
				assert.Equal(t, string(increase.CurrencyUSD), resp.Account.Metadata["currency"])
			},
			wantErr: false,
		},
		{
			name:      "transaction created",
			eventType: webhookTypeTransactionCreated,
			data: &increase.Transaction{
				ID:        "txn_123",
				Status:    increase.TransactionStatusPending,
				Type:      increase.TransactionTypeACHTransfer,
				Amount:    increase.Amount{MinorUnits: 100},
				Currency:  increase.CurrencyUSD,
				CreatedAt: time.Now(),
			},
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.NotNil(t, resp.Payment)
				assert.Equal(t, "txn_123", resp.Payment.ID)
				assert.Equal(t, models.PaymentStatusPending, resp.Payment.Status)
				assert.Equal(t, models.PaymentTypeACH, resp.Payment.Type)
				assert.Equal(t, int64(100), resp.Payment.Amount)
				assert.Equal(t, string(increase.CurrencyUSD), resp.Payment.Currency)
			},
			wantErr: false,
		},
		{
			name:      "transfer created",
			eventType: webhookTypeTransferCreated,
			data: &increase.WireTransfer{
				ID:        "transfer_123",
				Status:    increase.WireTransferStatusPending,
				Type:      increase.WireTransferTypeWire,
				Amount:    200,
				Currency:  increase.CurrencyUSD,
				CreatedAt: time.Now(),
			},
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.NotNil(t, resp.Payment)
				assert.Equal(t, "transfer_123", resp.Payment.ID)
				assert.Equal(t, models.PaymentStatusPending, resp.Payment.Status)
				assert.Equal(t, models.PaymentTypeWire, resp.Payment.Type)
				assert.Equal(t, int64(200), resp.Payment.Amount)
				assert.Equal(t, string(increase.CurrencyUSD), resp.Payment.Currency)
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
			signature: "whsig_test_123",
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.Nil(t, resp.Account)
				assert.Nil(t, resp.Payment)
			},
			wantErr: false,
		},
		{
			name:      "invalid signature",
			eventType: webhookTypeAccountCreated,
			data: &increase.Account{
				ID:        "acc_123",
				Name:      "Test Account",
				Status:    increase.AccountStatusActive,
				Type:      increase.AccountTypeChecking,
				Currency:  increase.CurrencyUSD,
				Bank:      increase.AccountBankIncrease,
				CreatedAt: time.Now(),
			},
			signature: "whsig_invalid",
			validate: func(t *testing.T, resp models.WebhookResponse) {
				assert.Nil(t, resp.Account)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlugin()
			p.client = mockSDK

			data, err := json.Marshal(tt.data)
			assert.NoError(t, err)

			webhookEvent := increase.WebhookEvent{
				ID:                  "evt_123",
				Type:               tt.eventType,
				CreatedAt:          time.Now(),
				AssociatedObjectID: "obj_123",
				Category:           "account.created",
				Data:              data,
			}
			webhookEventBytes, err := json.Marshal(webhookEvent)
			assert.NoError(t, err)

			mockSDK.On("Webhooks").Return(&increase.WebhookService{
				Client: &increase.Client{},
			})
			if tt.signature != "" {
			mockSDK.On("ValidateWebhookSignature", webhookEventBytes, tt.signature, mock.Anything).Return(nil)
			}

			resp, err := p.TranslateWebhook(context.Background(), models.TranslateWebhookRequest{
				Webhook: models.PSPWebhook{
					Raw:     webhookEventBytes,
					Headers: map[string]string{"Increase-Webhook-Signature": tt.signature},
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
