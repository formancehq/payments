package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"golang.org/x/sync/errgroup"
)


const enrichmentRefreshInterval = 24 * time.Hour

const (
	pathMarkets         = "/api/v2/markets/"
	pathMyMarkets       = "/api/v2/my_markets/"
	pathFeesTrading     = "/api/v2/fees/trading/"
	pathFeesWithdrawal  = "/api/v2/fees/withdrawal/"
)

// ErrPartialEnrichment signals at least one enrichment source failed.
// The orchestrator continues — accounts ship without the missing
// metadata rather than failing the install / cycle.
var ErrPartialEnrichment = errors.New("install-time enrichment: at least one source failed")

// enrichmentState holds the install-time-loaded caches feeding
// PSPAccount metadata. Refreshed in parallel under a 24h TTL.
type enrichmentState struct {
	mu                 sync.RWMutex
	markets            []client.Market
	myMarkets          []client.MyMarket
	tradingFees        []client.TradingFee
	withdrawalFees     []client.WithdrawalFee
	marketsSync        time.Time
	myMarketsSync      time.Time
	tradingFeesSync    time.Time
	withdrawalFeesSync time.Time
}

// ensureEnrichment refreshes the four enrichment caches in parallel.
// skipMap is read+written under a local mutex so callers can persist
// newly-discovered skip decisions via FetchNextAccounts NewState.
func (p *Plugin) ensureEnrichment(ctx context.Context, skipMap map[string]bool) error {
	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return refreshCache(gCtx, p, pathMarkets, skipMap, &mu,
			&p.enrichment.marketsSync,
			p.client.GetMarkets,
			func(v []client.Market) { p.enrichment.markets = v })
	})
	g.Go(func() error {
		return refreshCache(gCtx, p, pathMyMarkets, skipMap, &mu,
			&p.enrichment.myMarketsSync,
			p.client.GetMyMarkets,
			func(v []client.MyMarket) { p.enrichment.myMarkets = v })
	})
	g.Go(func() error {
		return refreshCache(gCtx, p, pathFeesTrading, skipMap, &mu,
			&p.enrichment.tradingFeesSync,
			p.client.GetTradingFees,
			func(v []client.TradingFee) { p.enrichment.tradingFees = v })
	})
	g.Go(func() error {
		return refreshCache(gCtx, p, pathFeesWithdrawal, skipMap, &mu,
			&p.enrichment.withdrawalFeesSync,
			p.client.GetWithdrawalFees,
			func(v []client.WithdrawalFee) { p.enrichment.withdrawalFees = v })
	})

	if err := g.Wait(); err != nil {
		p.logger.WithField("error", err.Error()).
			Errorf("enrichment refresh: at least one source failed")
		return fmt.Errorf("%w: %v", ErrPartialEnrichment, err)
	}
	return nil
}

// refreshCache is the generic single-source refresh path: skip-map
// gate, TTL gate, client call, derivatives-error skip, write under lock.
// skipMap and skipMu are owned by ensureEnrichment so writes are safe
// across the errgroup goroutines.
func refreshCache[T any](
	ctx context.Context,
	p *Plugin,
	path string,
	skipMap map[string]bool,
	skipMu *sync.Mutex,
	syncTime *time.Time,
	fetch func(context.Context) ([]T, error),
	store func([]T),
) error {
	skipMu.Lock()
	skip := skipMap[path]
	skipMu.Unlock()
	if skip {
		return nil
	}

	p.enrichment.mu.RLock()
	fresh := !syncTime.IsZero() && time.Since(*syncTime) < enrichmentRefreshInterval
	p.enrichment.mu.RUnlock()
	if fresh {
		return nil
	}

	rows, err := fetch(ctx)
	if err != nil {
		var derivErr *client.DerivativesUnsupportedError
		if errors.As(err, &derivErr) {
			skipMu.Lock()
			skipMap[path] = true
			skipMu.Unlock()
			p.logger.WithField("endpoint", path).WithField("reason", derivErr.Error()).
				Infof("marking enrichment endpoint as not-supported; future cycles will skip it")
			return nil
		}
		return fmt.Errorf("failed to refresh %s: %w", path, err)
	}
	if len(rows) == 0 {
		p.logger.WithField("endpoint", path).Infof("enrichment refresh returned 0 rows")
	}
	p.enrichment.mu.Lock()
	store(rows)
	*syncTime = time.Now()
	p.enrichment.mu.Unlock()
	return nil
}

// buildEnrichmentForCurrency assembles the AccountEnrichment payload
// for a single currency from the install-time caches. Returns the
// zero value when nothing relevant is cached.
func (p *Plugin) buildEnrichmentForCurrency(currencies map[string]int, currentCurrency client.Currency, symbol string) mappers.AccountEnrichment {
	p.enrichment.mu.RLock()
	defer p.enrichment.mu.RUnlock()

	enrich := mappers.AccountEnrichment{Networks: currentCurrency.Networks}

	for _, f := range p.enrichment.withdrawalFees {
		if strings.EqualFold(f.Currency, symbol) {
			enrich.WithdrawalFees = append(enrich.WithdrawalFees, f)
		}
	}

	for _, m := range p.enrichment.myMarkets {
		base, quote, ok := splitURLSymbol(m.URLSymbol, currencies)
		if !ok {
			continue
		}
		if base == symbol || quote == symbol {
			enrich.TradableMarkets = append(enrich.TradableMarkets, m)
		}
	}

	// Representative snapshot (not authoritative per-pair). Pick the
	// lexicographically-first match by stable key so the value does
	// not flap if the API returns rows in a different order between
	// cycles. Authoritative per-pair data lives in the cache and can
	// be queried independently.
	var repMarket *client.Market
	for i := range p.enrichment.markets {
		m := &p.enrichment.markets[i]
		if strings.ToUpper(m.BaseCurrency) != symbol {
			continue
		}
		if repMarket == nil || marketKey(*m) < marketKey(*repMarket) {
			repMarket = m
		}
	}
	if repMarket != nil {
		enrich.MinOrderValue = repMarket.MinimumOrderValue
		enrich.MarketType = repMarket.MarketType
	}

	var repFee *client.TradingFee
	for i := range p.enrichment.tradingFees {
		f := &p.enrichment.tradingFees[i]
		base, _, ok := splitURLSymbol(f.CurrencyPair, currencies)
		if !ok || base != symbol {
			continue
		}
		if repFee == nil || f.CurrencyPair < repFee.CurrencyPair {
			repFee = f
		}
	}
	if repFee != nil {
		enrich.MakerFee = repFee.Fees.Maker
		enrich.TakerFee = repFee.Fees.Taker
	}
	return enrich
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
func splitURLSymbol(urlSymbol string, currencies map[string]int) (base, quote string, ok bool) {
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
