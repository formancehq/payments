//go:build ee

package registry

// EnterpriseOnlyPlugins is empty in the Enterprise Edition because all
// connectors are available.
var EnterpriseOnlyPlugins = map[string]struct{}{}
