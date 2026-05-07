package storage

import (
	"fmt"
	"regexp"
)

var (
	metadataRegex = regexp.MustCompile(`metadata\[(.+)\]`)
)

// matchMetadataKey decodes a "metadata[<key>]"-shaped query key into the SQL
// fragment selecting JSONB-contained metadata. column is the fully-qualified
// metadata column to match against (e.g. "account.metadata" or "metadata"
// when no alias is needed). Returns ok=false when key is not a metadata key,
// in which case the caller should fall through to its column-specific logic.
func matchMetadataKey(column, key, operator string, value any) (clause string, args []any, ok bool, err error) {
	if !metadataRegex.MatchString(key) {
		return "", nil, false, nil
	}
	if operator != "$match" {
		return "", nil, true, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
	}
	m := metadataRegex.FindStringSubmatch(key)
	return column + " @> ?", []any{map[string]any{m[1]: value}}, true, nil
}
