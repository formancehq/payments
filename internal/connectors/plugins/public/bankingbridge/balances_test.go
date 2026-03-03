package bankingbridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingbridge/client"
	"github.com/formancehq/payments/internal/models"
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
	reportedAt := time.Now().Add(-time.Hour).UTC()
	importedAt1 := time.Now().Add(-time.Minute).UTC()
	importedAt2 := time.Now().UTC()
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
