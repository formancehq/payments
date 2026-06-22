package assets

import (
	"regexp"
)

// Pattern mirrors Ledger's canonical asset regex (formancehq/ledger/pkg/assets).
const Pattern = `[A-Z][A-Z0-9]{0,16}(_[A-Z]{1,16})?(\/\d{1,6})?`

var Regexp = regexp.MustCompile("^" + Pattern + "$")

func IsValid(v string) bool {
	return Regexp.MatchString(v)
}
