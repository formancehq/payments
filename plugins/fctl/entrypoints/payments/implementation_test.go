//go:build fctl_component_guest

package export_formance_fctl_plugin_lifecycle

import (
	"testing"

	pb "github.com/formancehq/fctl-v2-poc/pkg/plugin/protocol/componentbridgev1alpha1"
)

func TestDescribeExportsAllPaymentsCommands(t *testing.T) {
	descriptor, err := pb.DecodeDescriptorEnvelope(Describe())
	if err != nil {
		t.Fatalf("decode descriptor: %v", err)
	}
	if descriptor.Metadata.Name != "payments" || len(descriptor.Commands) != 44 {
		t.Fatalf("descriptor identity/commands = %q/%d", descriptor.Metadata.Name, len(descriptor.Commands))
	}
}
