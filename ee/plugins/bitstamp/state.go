package bitstamp


// accountsState persists the set of currencies already emitted so that
// subsequent FetchNextAccounts cycles do not re-emit the same account.
type accountsState struct {
	AccountCurrenciesImportedAt map[string]string `json:"accountCurrenciesImportedAt,omitempty"`
}

// paymentsState carries the since_id watermark for user_transactions.
type paymentsState struct {
	LastTransactionID int64 `json:"lastTransactionID"`
}


// ordersState tracks the last seen MarketEventID per market so that
// GetAccountOrderData since_id only returns unseen events. HasMoreCurrentMarket
// holds the market key at which the previous page stopped so the next call can
// resume from that point instead of restarting from the beginning of the list.
type ordersState struct {
	LastSeenEventIDPerMarket map[string]string `json:"lastSeenEventIDPerMarket"`
	HasMoreCurrentMarket     string            `json:"hasMoreCurrentMarket,omitempty"`
}

type conversionsState struct {
	LastTransactionID int64 `json:"lastTransactionID"`
}

// advanceInt64Cursor never resets the cursor on an empty page —
// an empty response (candidate = 0) must preserve the watermark.
func advanceInt64Cursor(current, candidate int64) int64 {
	if candidate > current {
		return candidate
	}
	return current
}
