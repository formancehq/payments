package bankingbridge

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
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
	bookedAt := time.Now().Add(-time.Hour).Truncate(time.Millisecond).UTC()
	importedAt1 := time.Now().Add(-time.Minute).Truncate(time.Millisecond).UTC()
	importedAt2 := time.Now().Truncate(time.Millisecond).UTC()
	trxs := []client.Transaction{
		{ID: "someID!", AccountReference: "acc1", BookedAt: bookedAt, ImportedAt: importedAt1, AmountInMinors: int64(5432), Asset: "CAD"},
		{ID: "someID!!", AccountReference: "acc2", BookedAt: bookedAt, ImportedAt: importedAt2, AmountInMinors: int64(5431), Asset: "KRW"},
	}

	newCursor := "newCursor"
	suite.client.EXPECT().GetTransactions(gomock.Any(), "", "", req.PageSize).Return(trxs, true, newCursor, nil)

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

	lastSeenImportedAt, err := time.Parse(ImportedAtLayout, state.LastSeenImportedAt)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), importedAt2, lastSeenImportedAt)
}

func (suite *PluginTestSuite) TestFetchNextPayments_ClientError() {
	ctx := context.Background()
	req := models.FetchNextPaymentsRequest{
		PageSize: 2,
		State:    nil,
	}

	expectedErr := errors.New("new err")
	suite.client.EXPECT().GetTransactions(gomock.Any(), "", "", req.PageSize).Return(nil, false, "", expectedErr)

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

func (suite *PluginTestSuite) TestToPSPPayment() {
	bookedAt := time.Now().Add(-time.Hour)
	importedAt := time.Now()

	rawMessage := json.RawMessage(`{"key":"value"}`)

	tests := []struct {
		name     string
		input    client.Transaction
		raw      json.RawMessage
		expected models.PSPPayment
	}{
		{
			name: "incoming SEPA credit transfer",
			input: client.Transaction{
				ID:                   "123",
				BookedAt:             bookedAt,
				AmountInMinors:       1000,
				Asset:                "USD",
				BankTransactionCode:  "PMNT.RCDT.ESCT",
				AccountReference:     "acc123",
				BookingDate:          "2023-10-01",
				ValutaDate:           "2023-10-02",
				NumberOfTransactions: 1,
				EntryReference:       "entry1",
				ServicerReference:    "servicer1",
				IsReversal:           false,
				IsBatch:              false,
				BatchMessageID:       "batch1",
				BatchPaymentInfoID:   "batchInfo1",
				ImportedAt:           importedAt,
			},
			raw: rawMessage,
			expected: models.PSPPayment{
				Reference:                   "123",
				CreatedAt:                   bookedAt,
				Amount:                      big.NewInt(1000),
				Asset:                       "USD",
				Type:                        models.PAYMENT_TYPE_PAYIN,
				Scheme:                      models.PAYMENT_SCHEME_SEPA_CREDIT,
				Status:                      models.PAYMENT_STATUS_SUCCEEDED,
				DestinationAccountReference: pointer.For("acc123"),
				Raw:                         rawMessage,
				Metadata: map[string]string{
					"bookingDate":          "2023-10-01",
					"valueDate":            "2023-10-02",
					"bankTransactionCode":  "PMNT.RCDT.ESCT",
					"numberofTransactions": "1",
					"entryReference":       "entry1",
					"servicerReference":    "servicer1",
					"isReversal":           "false",
					"isBatch":              "false",
					"batchMessageId":       "batch1",
					"batchPaymentInfoId":   "batchInfo1",
					"importedAt":           importedAt.String(),
				},
			},
		},
		{
			name: "outgoing SEPA credit transfer",
			input: client.Transaction{
				ID:                   "456",
				BookedAt:             bookedAt,
				AmountInMinors:       -500,
				Asset:                "EUR",
				BankTransactionCode:  "PMNT.ICDT.ESCT",
				AccountReference:     "acc456",
				BookingDate:          "2023-10-03",
				ValutaDate:           "2023-10-04",
				NumberOfTransactions: 2,
				EntryReference:       "entry2",
				ServicerReference:    "servicer2",
				IsReversal:           false,
				IsBatch:              true,
				BatchMessageID:       "batch2",
				BatchPaymentInfoID:   "batchInfo2",
				ImportedAt:           importedAt,
			},
			raw: rawMessage,
			expected: models.PSPPayment{
				Reference:              "456",
				CreatedAt:              bookedAt,
				Amount:                 big.NewInt(500),
				Asset:                  "EUR",
				Type:                   models.PAYMENT_TYPE_PAYOUT,
				Scheme:                 models.PAYMENT_SCHEME_SEPA_CREDIT,
				Status:                 models.PAYMENT_STATUS_SUCCEEDED,
				SourceAccountReference: pointer.For("acc456"),
				Metadata: map[string]string{
					"bookingDate":          "2023-10-03",
					"valueDate":            "2023-10-04",
					"bankTransactionCode":  "PMNT.ICDT.ESCT",
					"numberofTransactions": "2",
					"entryReference":       "entry2",
					"servicerReference":    "servicer2",
					"isReversal":           "false",
					"isBatch":              "true",
					"batchMessageId":       "batch2",
					"batchPaymentInfoId":   "batchInfo2",
					"importedAt":           importedAt.String(),
				},
				Raw: rawMessage,
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			result := ToPSPPayment(tt.input, tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}
