package bankingbridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/payments/ee/plugins/bankingbridge/client"
	"github.com/formancehq/payments/pkg/domain/plugins"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func (suite *PluginTestSuite) TestFetchNextBalances_Success() {
	ctx := context.Background()
	req := models.FetchNextBalancesRequest{
		PageSize: 2,
		State:    nil,
	}
	reportedAt := time.Now().Add(-time.Hour).Truncate(time.Millisecond).UTC()
	importedAt1 := time.Now().Add(-time.Minute).Truncate(time.Millisecond).UTC()
	importedAt2 := time.Now().Truncate(time.Millisecond).UTC()
	bals := []client.Balance{
		{AccountReference: "acc1", AmountInMinors: int64(1234), Asset: "EUR", ReportedAt: reportedAt, ImportedAt: importedAt1},
		{AccountReference: "acc2", AmountInMinors: int64(999), Asset: "USD", ReportedAt: reportedAt, ImportedAt: importedAt2},
	}

	newCursor := "newCursor"
	suite.client.EXPECT().GetAccountBalances(gomock.Any(), "", "", req.PageSize).Return(bals, true, newCursor, nil)

	resp, err := suite.plugin.FetchNextBalances(ctx, req)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), resp.Balances, len(bals))
	assert.Equal(suite.T(), bals[0].AccountReference, resp.Balances[0].AccountReference)
	assert.Equal(suite.T(), bals[0].Asset, resp.Balances[0].Asset)
	assert.Equal(suite.T(), bals[0].ReportedAt, resp.Balances[0].CreatedAt)
	require.NotNil(suite.T(), resp.Balances[0].Amount)
	assert.Equal(suite.T(), bals[0].AmountInMinors, resp.Balances[0].Amount.Int64())
	assert.True(suite.T(), resp.HasMore)

	var state workflowState
	err = json.Unmarshal(resp.NewState, &state)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newCursor, state.Cursor)

	lastSeenImportedAt, err := time.Parse(ImportedAtLayout, state.LastSeenImportedAt)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), importedAt2, lastSeenImportedAt)
}

func (suite *PluginTestSuite) TestFetchNextBalances_WithoutReportedAt() {
	ctx := context.Background()
	req := models.FetchNextBalancesRequest{PageSize: 4}
	importedAt := time.Now().Truncate(time.Millisecond).UTC()
	bals := []client.Balance{
		{AccountReference: "acc1", AmountInMinors: 100, Asset: "EUR/2", BalanceType: "OPBD", BalanceDate: "2026-03-01", ImportedAt: importedAt.Add(-3 * time.Minute)},
		{AccountReference: "acc1", AmountInMinors: 200, Asset: "EUR/2", BalanceType: "CLBD", BalanceDate: "2026-03-01", ImportedAt: importedAt.Add(-2 * time.Minute)},
		{AccountReference: "acc2", AmountInMinors: 300, Asset: "EUR/2", BalanceType: "ITBD", BalanceDate: "2026-03-02", ImportedAt: importedAt.Add(-time.Minute)},
		{AccountReference: "acc2", AmountInMinors: 400, Asset: "EUR/2", BalanceType: "ITAV", BalanceDate: "2026-03-02", ImportedAt: importedAt},
	}
	suite.client.EXPECT().GetAccountBalances(gomock.Any(), "", "", req.PageSize).Return(bals, false, "", nil)

	resp, err := suite.plugin.FetchNextBalances(ctx, req)
	require.NoError(suite.T(), err)

	// Only the booked balances are kept: CLBD at midnight UTC on its date,
	// ITBD at its import.
	require.Len(suite.T(), resp.Balances, 2)
	assert.Equal(suite.T(), "acc1", resp.Balances[0].AccountReference)
	assert.Equal(suite.T(), int64(200), resp.Balances[0].Amount.Int64())
	assert.Equal(suite.T(), time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), resp.Balances[0].CreatedAt)
	assert.Equal(suite.T(), "acc2", resp.Balances[1].AccountReference)
	assert.Equal(suite.T(), int64(300), resp.Balances[1].Amount.Int64())
	assert.Equal(suite.T(), bals[2].ImportedAt, resp.Balances[1].CreatedAt)

	// The state still moves past the discarded balances.
	var state workflowState
	require.NoError(suite.T(), json.Unmarshal(resp.NewState, &state))
	lastSeenImportedAt, err := time.Parse(ImportedAtLayout, state.LastSeenImportedAt)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), importedAt, lastSeenImportedAt)
}

func (suite *PluginTestSuite) TestFetchNextBalances_InvalidBalanceDate() {
	ctx := context.Background()
	req := models.FetchNextBalancesRequest{PageSize: 1}
	bals := []client.Balance{
		{AccountReference: "acc1", AmountInMinors: 200, Asset: "EUR/2", BalanceType: "CLBD", BalanceDate: "not-a-date", ImportedAt: time.Now().UTC()},
	}
	suite.client.EXPECT().GetAccountBalances(gomock.Any(), "", "", req.PageSize).Return(bals, false, "", nil)

	_, err := suite.plugin.FetchNextBalances(ctx, req)
	require.ErrorContains(suite.T(), err, "invalid balance date")
}

func (suite *PluginTestSuite) TestFetchNextBalances_ClientError() {
	ctx := context.Background()
	req := models.FetchNextBalancesRequest{
		PageSize: 2,
		State:    nil,
	}

	expectedErr := errors.New("expected")
	suite.client.EXPECT().GetAccountBalances(gomock.Any(), "", "", req.PageSize).Return(nil, false, "", expectedErr)

	_, err := suite.plugin.FetchNextBalances(ctx, req)
	require.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedErr, err)
}

func (suite *PluginTestSuite) TestFetchNextBalances_NotYetInstalled() {
	ctx := context.Background()
	req := models.FetchNextBalancesRequest{
		PageSize: 2,
		State:    nil,
	}

	suite.plugin.client = nil

	_, err := suite.plugin.FetchNextBalances(ctx, req)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), plugins.ErrNotYetInstalled, err)
}
