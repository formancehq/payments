package mappers

import (
	"fmt"
	"time"
)

// BitstampDatetimeLayout is the wire format Bitstamp uses for every
// timestamp on REST v2 endpoints we touch: account balances, user
// transactions, open orders, and order status fills.
const BitstampDatetimeLayout = "2006-01-02 15:04:05.000000"

// ParseBitstampTime parses a Bitstamp datetime string into a UTC
// time.Time. Returns a wrapped error including the offending value so
// the caller can include it in the failed-row's log context.
func ParseBitstampTime(s string) (time.Time, error) {
	t, err := time.Parse(BitstampDatetimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse bitstamp datetime %q: %w", s, err)
	}
	return t.UTC(), nil
}

// BitstampGenesis is the sentinel used as PSPAccount.CreatedAt for the
// synthetic per-currency accounts surfaced by /api/v2/account_balances/.
// Bitstamp does not expose per-currency creation dates; time.Now() would
// make accounts look "new" on every reinstall. The exchange's launch
// date is a stable lower-bound sentinel — readers should treat it as
// "creation date unknown, definitely before this". Documented in
// MAPPINGS.md §7.
var BitstampGenesis = time.Date(2011, 8, 2, 0, 0, 0, 0, time.UTC)
