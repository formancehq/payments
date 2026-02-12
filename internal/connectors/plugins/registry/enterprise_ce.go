//go:build !ee

package registry

// EnterpriseOnlyPlugins lists connectors that are only available in the
// Enterprise Edition. When a user attempts to use one of these connectors
// on a Community Edition binary, we can return a more helpful error message.
var EnterpriseOnlyPlugins = map[string]struct{}{
	"fireblocks": {},
}
