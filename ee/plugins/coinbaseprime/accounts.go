package coinbaseprime

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

// walletTypes enumerates the wallet `type` values Coinbase Prime documents
// as required on GET /wallets. Order is stable so mid-cycle resumability
// is deterministic: the plugin stores the current type by name
// (CurrentType), and on resume locates its index in this slice; unknown
// names fall back to the first type.
//
// Do not mutate — treat as constant. Declared `var` because Go does not
// allow `const` slices.
var walletTypes = []string{
	"TRADING",
	"VAULT",
	"ONCHAIN",
	"QC",
	"WALLET_TYPE_OTHER",
}

// accountsState tracks per-cycle progress across the five-type wallet
// iteration. Each wallet type has its own Coinbase pagination cursor
// persisted in Cursors[type], so successive cycles pick up where the
// previous cycle left off for that type without re-listing from history.
//
// Cursors are treated as opaque Coinbase tokens (see state.go /
// advanceCursor). Processing is idempotent — the framework dedupes
// emitted accounts by reference — so no client-side timestamp watermark
// is needed.
type accountsState struct {
	// Cursors maps wallet type → opaque Coinbase cursor representing the
	// latest position already consumed for that type.
	Cursors map[string]string `json:"cursors"`
	// CurrentType names the wallet type currently being paginated inside
	// the current cycle. Empty means the previous pass finished and the
	// next call should start at walletTypes[0].
	CurrentType string `json:"currentType"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	typeIndex := resolveTypeIndex(oldState.CurrentType)
	currentType := walletTypes[typeIndex]
	cursor := oldState.Cursors[currentType]

	response, err := p.client.GetWallets(ctx, currentType, cursor, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	p.assetsMu.RLock()
	currencies := p.currencies
	p.assetsMu.RUnlock()

	accounts := make([]models.PSPAccount, 0, len(response.Wallets))
	for _, wallet := range response.Wallets {
		symbol := strings.ToUpper(strings.TrimSpace(wallet.Symbol))
		if _, ok := currencies[symbol]; !ok {
			p.logger.Infof("skipping wallet %s: unsupported currency %q", wallet.ID, wallet.Symbol)
			continue
		}

		raw, err := json.Marshal(wallet)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		defaultAsset := currency.FormatAsset(currencies, symbol)

		accounts = append(accounts, models.PSPAccount{
			Reference:    wallet.ID,
			CreatedAt:    wallet.CreatedAt,
			Name:         &wallet.Name,
			DefaultAsset: &defaultAsset,
			Metadata: map[string]string{
				"wallet_type": wallet.Type,
			},
			Raw: raw,
		})
	}

	newCursors := copyCursors(oldState.Cursors)
	newCursors[currentType] = advanceCursor(cursor, response.Pagination.NextCursor)

	newState, hasMore := advanceAccountsState(newCursors, typeIndex, response.Pagination.HasNext)

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

// resolveTypeIndex locates currentType in walletTypes. Empty names or
// names not present in the slice (e.g. after a code-level rename) reset
// the iteration to index 0.
func resolveTypeIndex(currentType string) int {
	if currentType == "" {
		return 0
	}
	for i, t := range walletTypes {
		if t == currentType {
			return i
		}
	}
	return 0
}

// advanceAccountsState computes the state to persist after a single page
// response, plus the framework's HasMore flag. The wallet-type loop
// progresses as: paginate current type to HasNext=false, then step to
// the next type; when the last type finishes, clear CurrentType so the
// next cycle restarts at walletTypes[0].
func advanceAccountsState(cursors map[string]string, typeIndex int, hasNext bool) (accountsState, bool) {
	if hasNext {
		// Still paginating the current type within the current cycle.
		return accountsState{
			Cursors:     cursors,
			CurrentType: walletTypes[typeIndex],
		}, true
	}

	// Current type is done. If more types remain in this pass, advance.
	if typeIndex+1 < len(walletTypes) {
		return accountsState{
			Cursors:     cursors,
			CurrentType: walletTypes[typeIndex+1],
		}, true
	}

	// Final type finished — pass complete; next cycle restarts at
	// walletTypes[0] (CurrentType left empty).
	return accountsState{
		Cursors: cursors,
	}, false
}

func copyCursors(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}
