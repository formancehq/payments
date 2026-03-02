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

func (suite *PluginTestSuite) TestFetchNextPayments_Success() {
	ctx := context.Background()
	req := models.FetchNextPaymentsRequest{
		PageSize: 2,
		State:    nil,
	}
	trxs := []client.Transaction{
		{ID: "someID!", AccountReference: "acc1", BookedAt: time.Now(), AmountInMinors: int64(5432), Asset: "CAD"},
		{ID: "someID!!", AccountReference: "acc2", BookedAt: time.Now(), AmountInMinors: int64(5431), Asset: "KRW"},
	}

	newCursor := "newCursor"
	suite.client.EXPECT().GetTransactions(gomock.Any(), "", req.PageSize).Return(trxs, true, newCursor, nil)

	resp, err := suite.plugin.FetchNextPayments(ctx, req)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), resp.Payments, len(trxs))
	assert.Equal(suite.T(), trxs[0].ID, resp.Payments[0].Reference)
	assert.Equal(suite.T(), trxs[0].BookedAt, resp.Payments[0].CreatedAt)
	require.NotNil(suite.T(), resp.Payments[0].Amount)
	assert.Equal(suite.T(), trxs[0].AmountInMinors, resp.Payments[0].Amount.Int64())
	assert.Equal(suite.T(), trxs[0].Asset, resp.Payments[0].Asset)
	assert.Contains(suite.T(), string(resp.Payments[0].Raw), trxs[0].ID)
	assert.True(suite.T(), resp.HasMore)

	var state workflowState
	err = json.Unmarshal(resp.NewState, &state)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newCursor, state.Cursor)
}

func (suite *PluginTestSuite) TestFetchNextPayments_ClientError() {
	ctx := context.Background()
	req := models.FetchNextPaymentsRequest{
		PageSize: 2,
		State:    nil,
	}

	expectedErr := errors.New("new err")
	suite.client.EXPECT().GetTransactions(gomock.Any(), "", req.PageSize).Return(nil, false, "", expectedErr)

	_, err := suite.plugin.FetchNextPayments(ctx, req)
	require.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedErr, err)
}

func (suite *PluginTestSuite) TestFetchNextPayments_NotYetInstalled() {
	ctx := context.Background()
	req := models.FetchNextPaymentsRequest{
		PageSize: 2,
		State:    nil,
	}

	suite.plugin.client = nil

	_, err := suite.plugin.FetchNextPayments(ctx, req)
	require.Error(suite.T(), err)
	assert.Equal(suite.T(), plugins.ErrNotYetInstalled, err)
}
