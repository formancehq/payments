package mappers

import (
	"math"
	"time"
)

// KrakenGenesis is the stable lower-bound sentinel used as
// PSPAccount.CreatedAt — Kraken does not expose per-asset creation
// dates. Readers should treat it as "unknown, definitely before
// this".
var KrakenGenesis = time.Date(2011, 8, 1, 0, 0, 0, 0, time.UTC)

// FloatEpochToTime converts a Kraken float epoch (seconds.fractions)
// to time.Time in UTC. Negative / NaN / Inf inputs collapse to the
// zero time so a malformed row never produces a wild timestamp.
func FloatEpochToTime(t float64) time.Time {
	if t <= 0 || math.IsNaN(t) || math.IsInf(t, 0) {
		return time.Time{}
	}
	sec := int64(t)
	nsec := int64((t - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).UTC()
}
