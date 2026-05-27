package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

var marketEventIDRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

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

	// Deterministic market order required for hasMore behaviour
	sort.Strings(markets)

	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}

	var orders []models.PSPOrder
	skipping := state.HasMoreCurrentMarket != ""

	for _, marketName := range markets {
		// Skip markets until we reach the one where the previous page stopped.
		if skipping {
			if marketName != state.HasMoreCurrentMarket {
				continue
			}
			skipping = false
		}

		sinceID := state.LastSeenEventIDPerMarket[marketName]
		var sinceIDPtr *string
		if isValidMarketEventID(sinceID) {
			sinceIDPtr = &sinceID
		}

		events, err := p.client.GetAccountOrderData(ctx, marketName, sinceIDPtr)
		if err != nil {
			if client.IsNotFoundError(err) {
				p.logger.WithField("market", marketName).
					Infof("order event history not available for this market type, skipping")
				continue
			}
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to fetch market %q order events: %w", marketName, err)
		}

		var lastEventID string
		for _, event := range events {
			// the ID we got from the state was already imported by the previous job
			if sinceID != "" && event.EventID == sinceID {
				continue
			}

			if !isValidMarketEventID(event.EventID) {
				p.logger.WithField("market", marketName).WithField("eventID", event.EventID).WithField("order_id", event.Data.IDStr).
					Debugf("skipping event with invalid ID")
				continue
			}

			order, err := mappers.AccountOrderDataEventToPSPOrder(currencies, accountReference, marketName, event)
			if err != nil {
				p.logger.
					WithField("market", marketName).
					WithField("eventID", event.EventID).
					WithField("orderReference", event.Data.IDStr).
					Errorf("failed to map order event: %v", err)
				continue
			}
			lastEventID = event.EventID
			orders = append(orders, *order)

			if req.PageSize > 0 && len(orders) >= req.PageSize {
				break
			}
		}

		if lastEventID != "" {
			state.LastSeenEventIDPerMarket[marketName] = lastEventID
		}

		if req.PageSize > 0 && len(orders) >= req.PageSize {
			state.HasMoreCurrentMarket = marketName
			payload, err := json.Marshal(state)
			if err != nil {
				return models.FetchNextOrdersResponse{}, fmt.Errorf("marshal orders state: %w", err)
			}
			return models.FetchNextOrdersResponse{
				Orders:   orders,
				NewState: payload,
				HasMore:  true,
			}, nil
		}
	}

	state.HasMoreCurrentMarket = ""

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
		return "", nil, fmt.Errorf("missing payload in FromPayload")
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

// isValidMarketEventID reports whether s is a UUID-formatted event ID
// (e.g. "000652ba-1467-f198-0000-00d800000020") usable as a since_id value.
func isValidMarketEventID(s string) bool {
	return marketEventIDRegexp.MatchString(s)
}
