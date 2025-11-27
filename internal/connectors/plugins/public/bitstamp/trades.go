package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	bsclient "github.com/formancehq/payments/internal/connectors/plugins/public/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// tradesState tracks pagination when fetching trades.
type tradesState struct {
	LastOffset       int       `json:"lastOffset"`
	LastCreationDate time.Time `json:"lastCreationDate"`
}

// fetchNextTrades retrieves trade transactions from all configured Bitstamp accounts.
// The method:
// 1. Fetches transactions from each account's user_transactions endpoint
// 2. Filters for exchange transactions (those with exchange rates)
// 3. Deduplicates by transaction ID
// 4. Converts to PSPTrade format
func (p *Plugin) fetchNextTrades(ctx context.Context, req models.FetchNextTradesRequest) (models.FetchNextTradesResponse, error) {
	// ---- load previous state ----
	var oldState tradesState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextTradesResponse{}, err
		}
	}

	newState := tradesState{
		LastOffset:       oldState.LastOffset,
		LastCreationDate: oldState.LastCreationDate,
	}

	// Get all configured accounts
	accounts := p.client.GetAllAccounts()
	if len(accounts) == 0 {
		return models.FetchNextTradesResponse{}, fmt.Errorf("no accounts configured")
	}

	p.logger.Infof("Fetching trades for %d accounts", len(accounts))

	var (
		allTrades []models.PSPTrade
		hasMore   bool
	)

	offset := oldState.LastOffset
	limit := req.PageSize
	if limit < 10 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	p.logger.Infof("Using offset=%d, limit=%d", offset, limit)

	// Fetch transactions for each account and deduplicate by ID
	seenTxIDs := make(map[string]bool)
	uniqueTransactionCount := 0
	duplicateCount := 0
	for _, account := range accounts {
		params := bsclient.TransactionsParams{
			Offset: offset,
			Limit:  limit,
			Sort:   "desc",
		}
		if !oldState.LastCreationDate.IsZero() {
			params.SinceTimestamp = oldState.LastCreationDate.Unix()
			p.logger.Infof("Using since timestamp: %v", oldState.LastCreationDate)
		}

		p.logger.Infof("Fetching transactions for account: %s (ID: %s)", account.Name, account.ID)
		rows, err := p.client.GetTransactionsForAccount(ctx, account, params)
		if err != nil {
			return models.FetchNextTradesResponse{}, fmt.Errorf("failed to fetch transactions for account %s: %w", account.Name, err)
		}

		p.logger.Infof("Account %s returned %d transactions", account.Name, len(rows))

		// Only add transactions we haven't seen yet
		for _, tx := range rows {
			txID := string(tx.ID)
			if seenTxIDs[txID] {
				duplicateCount++
				p.logger.Debugf("Skipping duplicate transaction ID: %s", txID)
				continue
			}
			seenTxIDs[txID] = true
			uniqueTransactionCount++

			// Filter for exchange transactions only (those with exchange rates)
			pair, rate := tx.GetExchangeRate()
			if rate == "" || pair == "" {
				continue
			}

			// Convert to PSPTrade immediately to attach account reference
			trade, err := transactionToTrade(tx, account.ID)
			if err != nil {
				p.logger.Infof("Failed to convert transaction %s to trade: %v", tx.ID, err)
				continue
			}
			allTrades = append(allTrades, trade)
		}

		if len(rows) >= limit {
			hasMore = true
		}
	}

	p.logger.Infof("Total unique trades: %d, duplicates filtered: %d", len(allTrades), duplicateCount)

	// Sort by date (descending)
	sort.Slice(allTrades, func(i, j int) bool {
		return allTrades[i].CreatedAt.After(allTrades[j].CreatedAt)
	})

	p.logger.Infof("Converted to %d trades", len(allTrades))

	// Update state from the *last* trade (descending order)
	if len(allTrades) > 0 {
		last := allTrades[len(allTrades)-1]
		newState.LastCreationDate = last.CreatedAt
		newState.LastOffset = offset + uniqueTransactionCount
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextTradesResponse{}, err
	}

	return models.FetchNextTradesResponse{
		Trades:   allTrades,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

// transactionToTrade converts a Bitstamp exchange transaction to a PSPTrade
func transactionToTrade(tx bsclient.Transaction, accountID string) (models.PSPTrade, error) {
	// Raw
	raw, err := json.Marshal(&tx)
	if err != nil {
		return models.PSPTrade{}, err
	}

	// Parse time
	createdAt, err := parseBitstampTime(tx.Datetime)
	if err != nil {
		return models.PSPTrade{}, fmt.Errorf("parse datetime: %w", err)
	}

	// Get exchange rate to determine market pair
	pair, rate := tx.GetExchangeRate()
	if rate == "" || pair == "" {
		return models.PSPTrade{}, fmt.Errorf("not an exchange transaction (no exchange rate)")
	}

	// Parse the pair (e.g., "usdc_eur" -> base=USDC, quote=EUR)
	parts := strings.Split(strings.ToUpper(pair), "_")
	if len(parts) != 2 {
		return models.PSPTrade{}, fmt.Errorf("invalid exchange rate pair format: %s", pair)
	}
	baseCode := parts[0]
	quoteCode := parts[1]

	// Validate currencies are supported
	if _, ok := supportedCurrenciesWithDecimal[baseCode]; !ok {
		return models.PSPTrade{}, fmt.Errorf("unsupported base currency: %s", baseCode)
	}
	if _, ok := supportedCurrenciesWithDecimal[quoteCode]; !ok {
		return models.PSPTrade{}, fmt.Errorf("unsupported quote currency: %s", quoteCode)
	}

	// Format assets as "CODE/scale"
	baseAsset := currency.FormatAsset(supportedCurrenciesWithDecimal, baseCode)
	quoteAsset := currency.FormatAsset(supportedCurrenciesWithDecimal, quoteCode)

	// Collect non-zero legs to determine trade side and amounts
	legs := map[string]string{
		"EUR":  string(tx.EUR),
		"USD":  string(tx.USD),
		"USDC": string(tx.USDC),
		"BTC":  string(tx.BTC),
		"DOGE": string(tx.DOGE),
	}

	type leg struct {
		Code   string
		Val    string
		Neg    bool
		Amount string // decimal string
	}
	nonZero := make([]leg, 0, 2)
	for code, s := range legs {
		if isZeroNumStr(s) {
			continue
		}
		ns := strings.TrimSpace(s)
		isNeg := strings.HasPrefix(ns, "-")

		// Clean and convert to decimal string
		cleanVal := strings.TrimPrefix(ns, "-")
		cleanVal = strings.TrimPrefix(cleanVal, "+")

		nonZero = append(nonZero, leg{
			Code:   code,
			Val:    ns,
			Neg:    isNeg,
			Amount: cleanVal,
		})
	}

	if len(nonZero) != 2 {
		return models.PSPTrade{}, fmt.Errorf("expected exactly 2 legs for exchange transaction, got %d", len(nonZero))
	}

	// Determine which leg is base and which is quote, and the trade side
	var baseLeg, quoteLeg leg
	var side models.TradeSide

	for _, l := range nonZero {
		if l.Code == baseCode {
			baseLeg = l
		} else if l.Code == quoteCode {
			quoteLeg = l
		}
	}

	// Validate we found both legs
	if baseLeg.Code == "" || quoteLeg.Code == "" {
		return models.PSPTrade{}, fmt.Errorf("missing base or quote leg in transaction")
	}

	// Determine trade side:
	// - BUY: base is positive (receiving base), quote is negative (spending quote)
	// - SELL: base is negative (giving base), quote is positive (receiving quote)
	if !baseLeg.Neg && quoteLeg.Neg {
		side = models.TRADE_SIDE_BUY
	} else if baseLeg.Neg && !quoteLeg.Neg {
		side = models.TRADE_SIDE_SELL
	} else {
		return models.PSPTrade{}, fmt.Errorf("invalid leg signs for trade: base=%s, quote=%s", baseLeg.Val, quoteLeg.Val)
	}

	// Parse fee (Bitstamp uses single fee field)
	feeStr := strings.TrimSpace(string(tx.Fee))
	fees := make([]models.TradeFee, 0)
	if !isZeroNumStr(feeStr) {
		// Fee currency is typically the quote currency
		feeAsset := quoteAsset
		fees = append(fees, models.TradeFee{
			Asset:  feeAsset,
			Amount: strings.TrimPrefix(feeStr, "-"),
		})
	}

	// Create a single fill for this trade (Bitstamp transactions are atomic)
	fills := []models.TradeFill{
		{
			TradeReference: string(tx.ID),
			Timestamp:      createdAt,
			Price:          string(rate), // Exchange rate is the price
			Quantity:       baseLeg.Amount,
			QuoteAmount:    quoteLeg.Amount,
			Fees:           fees,
			Raw:            raw,
		},
	}

	// Calculate executed values
	executed := models.TradeExecuted{
		Quantity:     &baseLeg.Amount,
		QuoteAmount:  &quoteLeg.Amount,
		AveragePrice: ptrString(string(rate)),
		CompletedAt:  &createdAt,
	}

	// Create market symbol
	symbol := fmt.Sprintf("%s-%s", baseCode, quoteCode)

	return models.PSPTrade{
		Reference:                 string(tx.ID),
		CreatedAt:                 createdAt,
		PortfolioAccountReference: &accountID,
		InstrumentType:            models.TRADE_INSTRUMENT_TYPE_SPOT,
		ExecutionModel:            models.TRADE_EXECUTION_MODEL_ORDER_BOOK,
		Market: models.TradeMarket{
			Symbol:     symbol,
			BaseAsset:  baseAsset,
			QuoteAsset: quoteAsset,
		},
		Side:     side,
		Status:   models.TRADE_STATUS_FILLED,
		Executed: executed,
		Fees:     fees,
		Fills:    fills,
		Metadata: map[string]string{
			"bitstamp_type": string(tx.Type),
			"exchange_pair": pair,
		},
		Raw: raw,
	}, nil
}

func ptrString(s string) *string {
	return &s
}
