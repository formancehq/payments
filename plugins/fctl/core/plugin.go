package core

import (
	"context"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const Name = "payments"

// Version is the plugin's SemVer identity. It is a variable, not a constant, so
// a release build can inject the exact published version at link time with
// `-ldflags -X`; the default is the development value.
var Version = "0.1.0"

type Plugin struct{}

var _ sdk.Plugin = Plugin{}

func (Plugin) Metadata() sdk.Metadata {
	commands := Catalogue()
	return sdk.Metadata{Name: Name, Version: Version, Facets: []sdk.Facet{{
		Kind: sdk.FacetCommandProvider, ProtocolVersion: sdk.CurrentCommandProviderFacetProtocolVersion,
		RequiredHostCapabilities: sdk.RequiredHostCapabilitiesForCommands(commands),
	}}}
}
func (Plugin) Commands() []sdk.Command                             { return Catalogue() }
func (Plugin) DocumentationResources() []sdk.DocumentationResource { return nil }
func (Plugin) Execute(ctx context.Context, request sdk.ExecuteRequest, host sdk.Host) error {
	return executeV3(ctx, request, host)
}
