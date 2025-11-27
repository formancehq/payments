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
	gomock "github.com/golang/mock/gomock"
)

func TestPoolsBalancesLatest(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	id := uuid.New()
	poolsAccount := []models.AccountID{{}}
	balancesResponse := []models.AggregatedBalance{
		{
			RelatedAccounts: []models.AccountID{
				{
					Reference:   "test1",
					ConnectorID: models.ConnectorID{},
				},
				{
					Reference:   "test2",
					ConnectorID: models.ConnectorID{},
				},
			},
			Asset:  "EUR/2",
			Amount: big.NewInt(400),
		},
		{
			RelatedAccounts: []models.AccountID{
				{
					Reference:   "test1",
					ConnectorID: models.ConnectorID{},
				},
			},
			Asset:  "USD/2",
			Amount: big.NewInt(200),
		},
	}

	tests := []struct {
		name                string
		poolsGetStorageErr  error
		accountsBalancesErr error
		expectedError       error
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
			name:                "storage error not found",
			accountsBalancesErr: storage.ErrNotFound,
			expectedError:       newStorageError(storage.ErrNotFound, "cannot get latest balances"),
		},
		{
			name:                "other error",
			accountsBalancesErr: fmt.Errorf("error"),
			expectedError:       newStorageError(fmt.Errorf("error"), "cannot get latest balances"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.EXPECT().PoolsGet(gomock.Any(), id).Return(&models.Pool{
				ID:           id,
				Name:         "test",
				CreatedAt:    time.Now().Add(-time.Hour),
				Type:         models.POOL_TYPE_STATIC,
				PoolAccounts: poolsAccount,
			}, test.poolsGetStorageErr)
			if test.poolsGetStorageErr == nil {
				store.EXPECT().BalancesGetFromAccountIDs(gomock.Any(), gomock.Any(), nil).Return(balancesResponse, test.accountsBalancesErr)
			}

			balances, err := s.PoolsBalances(context.Background(), id)
			if test.expectedError == nil {
				require.NoError(t, err)
				require.NotNil(t, balances)
				foundEUR := false
				foundUSD := false
				for _, balance := range balances {
					switch balance.Asset {
					case "EUR/2":
						require.Equal(t, big.NewInt(400), balance.Amount)
						require.Equal(t, []models.AccountID{
							{Reference: "test1", ConnectorID: models.ConnectorID{}},
							{Reference: "test2", ConnectorID: models.ConnectorID{}},
						}, balance.RelatedAccounts)
						foundEUR = true
					case "USD/2":
						require.Equal(t, []models.AccountID{
							{Reference: "test1", ConnectorID: models.ConnectorID{}},
						}, balance.RelatedAccounts)
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
