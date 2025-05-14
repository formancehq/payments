package services

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestPoolsBalancesAt(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	id := uuid.New()
	poolsAccount := []models.AccountID{{}}
	balancesResponse := []*models.Balance{
		{
			Asset:   "EUR/2",
			Balance: big.NewInt(100),
		},
		{
			Asset:   "USD/2",
			Balance: big.NewInt(200),
		},
		{
			Asset:   "EUR/2",
			Balance: big.NewInt(300),
		},
	}
	at := time.Now().Add(-time.Hour)

	tests := []struct {
		name                  string
		poolsGetStorageErr    error
		accountsBalancesAtErr error
		expectedError         error
	}{
		{
			name:               "success",
			poolsGetStorageErr: nil,
			expectedError:      nil,
		},
		{
			name:               "storage error not found",
			poolsGetStorageErr: storage.ErrNotFound,
			expectedError:      newStorageError(storage.ErrNotFound, "cannot get pool"),
		},
		{
			name:               "other error",
			poolsGetStorageErr: fmt.Errorf("error"),
			expectedError:      newStorageError(fmt.Errorf("error"), "cannot get pool"),
		},
		{
			name:                  "storage error not found",
			accountsBalancesAtErr: storage.ErrNotFound,
			expectedError:         newStorageError(storage.ErrNotFound, "cannot get balances"),
		},
		{
			name:                  "other error",
			accountsBalancesAtErr: fmt.Errorf("error"),
			expectedError:         newStorageError(fmt.Errorf("error"), "cannot get balances"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().PoolsGet(gomock.Any(), id).Return(&models.Pool{
				ID:           id,
				Name:         "test",
				CreatedAt:    at,
				PoolAccounts: poolsAccount,
			}, test.poolsGetStorageErr)
			if test.poolsGetStorageErr == nil {
				store.EXPECT().BalancesGetAt(gomock.Any(), models.AccountID{}, at).Return(balancesResponse, test.accountsBalancesAtErr)
			}

			balances, err := s.PoolsBalancesAt(context.Background(), id, at)
			if test.expectedError == nil {
				require.NoError(t, err)
				require.NotNil(t, balances)
				foundEUR := false
				foundUSD := false
				for _, balance := range balances {
					switch balance.Asset {
					case "EUR/2":
						require.Equal(t, big.NewInt(400), balance.Amount)
						foundEUR = true
					case "USD/2":
						require.Equal(t, big.NewInt(200), balance.Amount)
						foundUSD = true
					default:
						require.Fail(t, "unexpected asset")
					}
				}
				require.True(t, foundEUR)
				require.True(t, foundUSD)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
