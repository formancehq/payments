package bitstamp

import (
	"encoding/json"
	"time"
)

// orderRetentionMax is 5 days less than Bitstamp's 30-day
// order_status/ retention window. Tracked orders past this are
// force-emitted with com.bitstamp.spec/retention_expired and dropped.
const orderRetentionMax = 25 * 24 * time.Hour

// paymentsState carries one watermark per payment source. Each
// substate advances independently because the three Bitstamp streams
// have different pagination primitives. See MAPPINGS §6.
type paymentsState struct {
	UserTransactions   userTransactionsState   `json:"userTransactions"`
	CryptoTransactions cryptoTransactionsState `json:"cryptoTransactions"`
	WithdrawalRequests withdrawalRequestsState `json:"withdrawalRequests"`
}

type userTransactionsState struct {
	// since_id is inclusive — the row at the watermark reappears next
	// cycle. The framework dedupes by PSPPayment.Reference.
	LastTransactionID int64 `json:"lastTransactionID"`
}

type cryptoTransactionsState struct {
	// Per-bucket Unix-seconds watermarks. Bitstamp does not expose an
	// id cursor on this endpoint. Populated only on Main-account
	// scopes; sub-account scopes trigger the try-and-skip cache.
	DepositsSinceTs    int64 `json:"depositsSinceTs"`
	WithdrawalsSinceTs int64 `json:"withdrawalsSinceTs"`
	RipplesSinceTs     int64 `json:"ripplesSinceTs"`
}

type withdrawalRequestsState struct {
	LastID int64 `json:"lastID"`
}

// UnmarshalJSON migrates the pre-multi-source flat shape
// {"lastTransactionID": N} into UserTransactions.LastTransactionID
// so existing installs keep their watermark on upgrade.
func (s *paymentsState) UnmarshalJSON(data []byte) error {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	if raw, hasLegacy := probe["lastTransactionID"]; hasLegacy && len(probe) == 1 {
		var legacy int64
		if err := json.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		*s = paymentsState{UserTransactions: userTransactionsState{LastTransactionID: legacy}}
		return nil
	}
	type alias paymentsState
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = paymentsState(a)
	return nil
}

// trackedOrder is the slim first-sight capture: only the original
// limit price needs persisting because order_status/ returns market,
// type, subtype, datetime, amount_remaining live every cycle.
type trackedOrder struct {
	LastStatus  string    `json:"lastStatus"`
	FirstSeenAt time.Time `json:"firstSeenAt"`
	LimitPrice  string    `json:"limitPrice"`
}

type ordersState struct {
	TrackedOrders map[string]trackedOrder `json:"trackedOrders"`
}

// UnmarshalJSON tolerates the legacy fuller trackedOrder shape
// (Amount / CurrencyPair / Type) by silently ignoring obsolete fields.
func (s *ordersState) UnmarshalJSON(data []byte) error {
	type alias ordersState
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	if a.TrackedOrders == nil {
		a.TrackedOrders = map[string]trackedOrder{}
	}
	*s = ordersState(a)
	return nil
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

// maxInt64 returns 0 for an empty slice (unlike slices.Max which panics).
func maxInt64(values []int64) int64 {
	var out int64
	for _, v := range values {
		if v > out {
			out = v
		}
	}
	return out
}
