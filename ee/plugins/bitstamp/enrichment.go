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

func (p *Plugin) ensureEnrichment(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return refreshCache(ctx, p, pathMarkets,
			&p.enrichment.marketsSync,
			p.client.GetMarkets,
			func(v []client.Market) { p.enrichment.markets = v })
	})
	g.Go(func() error {
		return refreshCache(ctx, p, pathMyMarkets,
			&p.enrichment.myMarketsSync,
			p.client.GetMyMarkets,
			func(v []client.MyMarket) { p.enrichment.myMarkets = v })
	})
	g.Go(func() error {
		return refreshCache(ctx, p, pathFeesTrading,
			&p.enrichment.tradingFeesSync,
			p.client.GetTradingFees,
			func(v []client.TradingFee) { p.enrichment.tradingFees = v })
	})
	g.Go(func() error {
		return refreshCache(ctx, p, pathFeesWithdrawal,
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

// refreshCache is the generic single-source refresh path: TTL gate,
// skip-cache gate (so endpoints flagged DerivativesUnsupported never
// re-poll), client call, derivatives-error swallow, write under lock.
func refreshCache[T any](
	ctx context.Context,
	p *Plugin,
	path string,
	sync *time.Time,
	fetch func(context.Context) ([]T, error),
	store func([]T),
) error {
	if p.shouldSkipEndpoint(path) {
		return nil
	}
	p.enrichment.mu.RLock()
	fresh := !sync.IsZero() && time.Since(*sync) < enrichmentRefreshInterval
	p.enrichment.mu.RUnlock()
	if fresh {
		return nil
	}
	rows, err := fetch(ctx)
	if err != nil {
		var derivErr *client.DerivativesUnsupportedError
		if errors.As(err, &derivErr) {
			p.markEndpointSkipped(path, derivErr.Error())
			return nil
		}
		return fmt.Errorf("failed to refresh %s: %w", path, err)
	}
	if len(rows) == 0 {
		p.logger.WithField("endpoint", path).Infof("enrichment refresh returned 0 rows")
	}
	p.enrichment.mu.Lock()
	store(rows)
	*sync = time.Now()
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

	// First matching row is a representative snapshot, not the
	// authoritative per-pair fee schedule (those live in the cache
	// and can be queried independently). A ticker anchors many
	// pairs with varying tier rates.
	for _, m := range p.enrichment.markets {
		if strings.ToUpper(m.BaseCurrency) != symbol {
			continue
		}
		if enrich.MinOrderValue == "" {
			enrich.MinOrderValue = m.MinimumOrderValue
		}
		if enrich.MarketType == "" {
			enrich.MarketType = m.MarketType
		}
		break
	}
	for _, f := range p.enrichment.tradingFees {
		base, _, ok := splitURLSymbol(f.CurrencyPair, currencies)
		if !ok || base != symbol {
			continue
		}
		if enrich.MakerFee == "" {
			enrich.MakerFee = f.Fees.Maker
		}
		if enrich.TakerFee == "" {
			enrich.TakerFee = f.Fees.Taker
		}
		break
	}
	return enrich
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
