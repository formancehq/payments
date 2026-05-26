package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextOrders queries account_order_data for every tradeable market
// listed in the parent account's metadata, emitting one PSPOrder per
// event. The last-seen order ID per market is persisted in state so
// subsequent calls use it as since_id to avoid re-fetching processed events.
func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	state := ordersState{LastSeenEventIDPerMarket: map[string]string{}}
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("unmarshal orders state: %w", err)
		}
		if state.LastSeenEventIDPerMarket == nil {
			state.LastSeenEventIDPerMarket = map[string]string{}
		}
	}

	accountReference, markets, err := tradeableMarketsFromPayload(req.FromPayload)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}

	// Deterministic market order so logs are stable.
	sort.Strings(markets)

	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}

	var orders []models.PSPOrder
	for _, marketName := range markets {
		// Markets in the metadata are stored as URL symbols (e.g. "btcusd").
		// The API and mapper both expect slash format (e.g. "BTC/USD").
		base, quote, err := mappers.SplitCurrencyPair(marketName)
		if err != nil {
			p.logger.WithField("market", marketName).Errorf("cannot parse market symbol, skipping: %v", err)
			continue
		}
		slashMarket := base + "/" + quote

		sinceID := state.LastSeenEventIDPerMarket[marketName]
		var sinceIDPtr *string
		if sinceID != "" {
			sinceIDPtr = &sinceID
		}

		events, err := p.client.GetAccountOrderData(ctx, slashMarket, sinceIDPtr)
		if err != nil {
			if client.IsNotFoundError(err) {
				p.logger.WithField("market", slashMarket).
					Infof("order event history not available for this market type, skipping")
				continue
			}
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to fetch market %q order events: %w", slashMarket, err)
		}

		var lastEventID string
		for _, event := range events {
			order, err := mappers.AccountOrderDataEventToPSPOrder(currencies, accountReference, slashMarket, event)
			if err != nil {
				p.logger.WithField("market", marketName).WithField("eventID", event.EventID).
					Errorf("failed to map order event: %v", err)
				continue
			}
			orders = append(orders, *order)
			if isValidMarketEventID(event.EventID) {
				lastEventID = event.EventID
			}
		}

		if lastEventID != "" {
			state.LastSeenEventIDPerMarket[marketName] = lastEventID
		}
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("marshal orders state: %w", err)
	}

	return models.FetchNextOrdersResponse{
		Orders:   orders,
		NewState: payload,
		HasMore:  false,
	}, nil
}

// tradeableMarketsFromPayload extracts the account reference and tradeable
// market symbols from the PSPAccount JSON passed as FromPayload. The engine
// unwraps its own {"id":…,"payload":…} envelope before invoking the plugin,
// so this function receives the raw PSPAccount JSON directly.
func tradeableMarketsFromPayload(payload json.RawMessage) (string, []string, error) {
	if len(payload) == 0 {
		return "", nil, nil
	}
	var account models.PSPAccount
	if err := json.Unmarshal(payload, &account); err != nil {
		return "", nil, fmt.Errorf("unmarshal from payload: %w", err)
	}
	raw, ok := account.Metadata[mappers.MetadataKeyTradableMarkets]
	if !ok || raw == "" {
		return account.Reference, nil, nil
	}
	var markets []string
	if err := json.Unmarshal([]byte(raw), &markets); err != nil {
		return "", nil, fmt.Errorf("unmarshal tradeable markets: %w", err)
	}
	return account.Reference, markets, nil
}

// isValidMarketEventID reports whether s is a 32-character lowercase hex
// string as required by Bitstamp's since_id parameter. The API rejects
// anything else with a 400, even though its own responses sometimes return
// shorter event_id values for certain markets.
func isValidMarketEventID(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
