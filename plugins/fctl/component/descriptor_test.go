package component

import (
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

func TestDescriptorContainsThePaymentsCommandFacet(t *testing.T) {
	descriptor := Descriptor()
	if descriptor.Metadata.Name != "payments" || len(descriptor.Commands) != 44 {
		t.Fatalf("descriptor identity/commands = %q/%d", descriptor.Metadata.Name, len(descriptor.Commands))
	}
	if len(descriptor.AuthProviders) != 0 || len(descriptor.TargetProviders) != 0 || len(descriptor.SignerProviders) != 0 {
		t.Fatalf("descriptor unexpectedly owns privileged facets: %#v", descriptor)
	}
	if err := sdk.ValidateCatalogue(descriptor.Commands, descriptor.DocumentationResources); err != nil {
		t.Fatalf("descriptor catalogue invalid: %v", err)
	}
}
