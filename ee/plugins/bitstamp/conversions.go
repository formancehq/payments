package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
)

// fetchNextConversions scans /api/v2/user_transactions/ on an
// independent since_id cursor and emits one PSPConversion per
// type-36 row with two non-zero known currencies. See MAPPINGS §4.5.
func (p *Plugin) fetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	var state conversionsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextConversionsResponse{}, fmt.Errorf("failed to unmarshal conversions state: %w", err)
		}
	}

	limit := effectivePageSize(req.PageSize)
	transactions, err := p.client.GetUserTransactions(ctx, sinceIDFor(state.LastTransactionID), limit)
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("failed to fetch conversions: %w", err)
	}

	if len(transactions) == 0 {
		return models.FetchNextConversionsResponse{
			NewState: req.State,
			HasMore:  false,
		}, nil
	}

	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextConversionsResponse{}, err
	}

	conversions := make([]models.PSPConversion, 0, len(transactions))
	lastSeen := state.LastTransactionID
	for _, tx := range transactions {
		lastSeen = advanceInt64Cursor(lastSeen, tx.ID)
		res, mapErr := mappers.UserTransactionToPSPConversion(currencies, tx)
		if mapErr != nil {
			p.logger.WithField("txID", tx.ID).Errorf("failed to map conversion: %v", mapErr)
			continue
		}
		if res.DerivativesRow {
			p.logger.WithField("txID", tx.ID).Infof("skipping derivatives-marked row on spot-only connector")
			continue
		}
		if res.Skip || res.Conversion == nil {
			continue
		}
		conversions = append(conversions, *res.Conversion)
	}

	payload, err := json.Marshal(conversionsState{LastTransactionID: lastSeen})
	if err != nil {
		return models.FetchNextConversionsResponse{}, fmt.Errorf("failed to marshal conversions state: %w", err)
	}

	return models.FetchNextConversionsResponse{
		Conversions: conversions,
		NewState:    payload,
		HasMore:     len(transactions) == limit,
	}, nil
}
