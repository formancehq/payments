package assets

import (
	"regexp"
)

// Pattern mirrors the canonical asset format enforced by Formance Ledger
// (see github.com/formancehq/ledger/pkg/assets): an uppercase symbol of up
// to 17 chars optionally followed by a single `_LETTERS` segment and an
// optional `/<precision>` suffix. Keeping the two regexes in sync prevents
// Payments from emitting assets that Ledger would later reject.
const Pattern = `[A-Z][A-Z0-9]{0,16}(_[A-Z]{1,16})?(\/\d{1,6})?`

var Regexp = regexp.MustCompile("^" + Pattern + "$")

func IsValid(v string) bool {
	return Regexp.MatchString(v)
}
