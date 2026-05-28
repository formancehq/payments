package mappers

import (
	"fmt"
	"time"
)

// BitstampDatetimeLayout: account_balances, user_transactions,
// open_orders, order_status.transactions[]. withdrawal-requests uses
// the no-microsecond variant — see withdrawalRequestDatetimeLayout.
const BitstampDatetimeLayout = "2006-01-02 15:04:05.000000"

func ParseBitstampTime(s string) (time.Time, error) {
	t, err := time.Parse(BitstampDatetimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse bitstamp datetime %q: %w", s, err)
	}
	return t.UTC(), nil
}

// BitstampGenesis is the stable lower-bound sentinel used as
// PSPAccount.CreatedAt — Bitstamp does not expose per-currency
// creation dates. Readers should treat it as "unknown, definitely
// before this".
var BitstampGenesis = time.Date(2011, 8, 2, 0, 0, 0, 0, time.UTC)
