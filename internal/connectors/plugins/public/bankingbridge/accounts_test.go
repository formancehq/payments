package bankingbridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingbridge/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func (suite *PluginTestSuite) TestFetchNextAccounts_Success() {
	ctx := context.Background()
	req := models.FetchNextAccountsRequest{
		PageSize: 2,
		State:    nil,
	}

	accs := []client.Account{
		{Reference: "acc1", ImportedAt: time.Now(), Name: pointer.For("name1"), DefaultAsset: pointer.For("JPY")},
		{Reference: "acc2", ImportedAt: time.Now(), Name: pointer.For("name1"), DefaultAsset: pointer.For("AUD")},
	}

	newCursor := "newCursor"
	suite.client.EXPECT().GetAccounts(gomock.Any(), "", req.PageSize).Return(accs, true, newCursor, nil)

	resp, err := suite.plugin.FetchNextAccounts(ctx, req)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), resp.Accounts, len(accs))
	assert.Equal(suite.T(), accs[0].Reference, resp.Accounts[0].Reference)
	require.NotNil(suite.T(), resp.Accounts[0].DefaultAsset)
	assert.Equal(suite.T(), accs[0].DefaultAsset, resp.Accounts[0].DefaultAsset)
	require.NotNil(suite.T(), resp.Accounts[0].Name)
	assert.Equal(suite.T(), accs[0].Name, resp.Accounts[0].Name)
	assert.Equal(suite.T(), accs[0].ImportedAt, resp.Accounts[0].CreatedAt)
	assert.Contains(suite.T(), string(resp.Accounts[0].Raw), accs[0].Reference)
	assert.True(suite.T(), resp.HasMore)

	var state workflowState
	err = json.Unmarshal(resp.NewState, &state)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newCursor, state.Cursor)
}

func (suite *PluginTestSuite) TestFetchNextAccounts_ClientError() {
	ctx := context.Background()
	req := models.FetchNextAccountsRequest{
		PageSize: 2,
		State:    nil,
	}

	expectedErr := errors.New("new err")
	suite.client.EXPECT().GetAccounts(gomock.Any(), "", req.PageSize).Return(nil, false, "", expectedErr)

	_, err := suite.plugin.FetchNextAccounts(ctx, req)
	require.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedErr, err)
}

func (suite *PluginTestSuite) TestFetchNextAccounts_NotYetInstalled() {
	ctx := context.Background()
	req := models.FetchNextAccountsRequest{
		PageSize: 2,
		State:    nil,
	}

	suite.plugin.client = nil

	_, err := suite.plugin.FetchNextAccounts(ctx, req)
	require.Error(suite.T(), err)
	assert.Equal(suite.T(), plugins.ErrNotYetInstalled, err)
}
