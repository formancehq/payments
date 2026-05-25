package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"golang.org/x/sync/errgroup"
)

const (
	pathMarkets        = "/api/v2/markets/"
	pathMyMarkets      = "/api/v2/my_markets/"
	pathFeesTrading    = "/api/v2/fees/trading/"
	pathFeesWithdrawal = "/api/v2/fees/withdrawal/"
)

// ErrPartialEnrichment signals at least one enrichment source failed.
// The orchestrator continues — accounts ship without the missing
// metadata rather than failing the cycle.
var ErrPartialEnrichment = errors.New("account data enrichment: at least one source failed")

// enrichmentState holds enrichment data for a single accounts cycle.
// Fetched fresh on each call to fetchAccountEnrichmentData; no caching.
type enrichmentState struct {
	markets        []client.Market
	myMarkets      []client.MyMarket
	tradingFees    []client.TradingFee
	withdrawalFees []client.WithdrawalFee
}

// fetchAccountEnrichmentData fetches the four enrichment sources in parallel and
// returns the combined result. DerivativesUnsupportedError is swallowed
// per source. Any other error wraps ErrPartialEnrichment.
func (p *Plugin) fetchAccountEnrichmentData(ctx context.Context) (enrichmentState, error) {
	var state enrichmentState
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		rows, err := p.client.GetMarkets(gCtx)
		if err != nil {
			return fmt.Errorf("failed to refresh %s: %w", pathMarkets, err)
		}
		state.markets = rows
		return nil
	})
	g.Go(func() error {
		rows, err := p.client.GetMyMarkets(gCtx)
		if err != nil {
			var derivErr *client.DerivativesUnsupportedError
			if errors.As(err, &derivErr) {
				return nil
			}
			return fmt.Errorf("failed to refresh %s: %w", pathMyMarkets, err)
		}
		state.myMarkets = rows
		return nil
	})
	g.Go(func() error {
		rows, err := p.client.GetTradingFees(gCtx)
		if err != nil {
			return fmt.Errorf("failed to refresh %s: %w", pathFeesTrading, err)
		}
		state.tradingFees = rows
		return nil
	})
	g.Go(func() error {
		rows, err := p.client.GetWithdrawalFees(gCtx)
		if err != nil {
			return fmt.Errorf("failed to refresh %s: %w", pathFeesWithdrawal, err)
		}
		state.withdrawalFees = rows
		return nil
	})

	if err := g.Wait(); err != nil {
		p.logger.WithField("error", err.Error()).
			Errorf("enrichment refresh: at least one source failed")
		return state, fmt.Errorf("%w: %v", ErrPartialEnrichment, err)
	}
	return state, nil
}

// buildEnrichmentForCurrency assembles the AccountEnrichment payload
// for a single currency from the provided enrichment data. Returns the
// zero value when nothing relevant is present.
func buildEnrichmentForCurrency(enrich enrichmentState, currencyIndex map[string]client.Currency, symbol string) mappers.AccountEnrichment {
	currentCurrency := currencyIndex[symbol]
	result := mappers.AccountEnrichment{Networks: currentCurrency.Networks}

	for _, f := range enrich.withdrawalFees {
		if strings.EqualFold(f.Currency, symbol) {
			result.WithdrawalFees = append(result.WithdrawalFees, f)
		}
	}

	for _, m := range enrich.myMarkets {
		base, quote, ok := splitURLSymbol(m.URLSymbol, currencyIndex)
		if !ok {
			continue
		}
		if base == symbol || quote == symbol {
			result.TradableMarkets = append(result.TradableMarkets, m)
		}
	}

	// Representative snapshot (not authoritative per-pair). Pick the
	// lexicographically-first match by stable key so the value does
	// not flap if the API returns rows in a different order between
	// cycles.
	var repMarket *client.Market
	for i := range enrich.markets {
		m := &enrich.markets[i]
		if strings.ToUpper(m.BaseCurrency) != symbol {
			continue
		}
		if repMarket == nil || marketKey(*m) < marketKey(*repMarket) {
			repMarket = m
		}
	}
	if repMarket != nil {
		result.MinOrderValue = repMarket.MinimumOrderValue
		result.MarketType = repMarket.MarketType
	}

	var repFee *client.TradingFee
	for i := range enrich.tradingFees {
		f := &enrich.tradingFees[i]
		base, _, ok := splitURLSymbol(f.CurrencyPair, currencyIndex)
		if !ok || base != symbol {
			continue
		}
		if repFee == nil || f.CurrencyPair < repFee.CurrencyPair {
			repFee = f
		}
	}
	if repFee != nil {
		result.MakerFee = repFee.Fees.Maker
		result.TakerFee = repFee.Fees.Taker
	}
	return result
}

// marketKey produces a stable composite ordering key for selecting
// a representative Market row. CounterCurrency disambiguates pairs
// sharing the same base (BTC/USD vs BTC/EUR vs BTC/USDT); MarketType
// tiebreaks SPOT vs other flavours on the same pair.
func marketKey(m client.Market) string {
	return m.CounterCurrency + "|" + m.MarketType
}

// splitURLSymbol parses Bitstamp's lowercase concat pair (e.g.
// "btcusd", "btcusdc") into uppercase (base, quote) using the known
// currencies map. Returns ok=false on unrecognisable shapes.
func splitURLSymbol(urlSymbol string, currencies map[string]client.Currency) (base, quote string, ok bool) {
	s := strings.ToLower(strings.TrimSpace(urlSymbol))
	if s == "" {
		return "", "", false
	}
	for baseLen := 3; baseLen <= 5 && baseLen < len(s); baseLen++ {
		b := strings.ToUpper(s[:baseLen])
		q := strings.ToUpper(s[baseLen:])
		if _, ok1 := currencies[b]; !ok1 {
			continue
		}
		if _, ok2 := currencies[q]; !ok2 {
			continue
		}
		return b, q, true
	}
	return "", "", false
}
