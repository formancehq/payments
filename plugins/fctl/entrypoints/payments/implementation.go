//go:build fctl_component_guest

package export_formance_fctl_plugin_lifecycle

import (
	"fmt"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk/portable/component"
	paymentcomponent "github.com/formancehq/payments/plugins/fctl/component"
	"github.com/formancehq/payments/plugins/fctl/core"
)

var lifecycle = mustLifecycle()

func Describe() []byte                                 { return lifecycle.Describe() }
func Start(executionID string, input []byte) [][]byte  { return lifecycle.Start(executionID, input) }
func Resume(executionID string, input []byte) [][]byte { return lifecycle.Resume(executionID, input) }
func Cancel(executionID string) [][]byte               { return lifecycle.Cancel(executionID) }
func Close(executionID string)                         { lifecycle.Close(executionID) }

func mustLifecycle() *component.Component {
	plugin := core.Plugin{}
	configured, err := component.New(component.Providers{Command: plugin}, paymentcomponent.Descriptor())
	if err != nil {
		panic(fmt.Sprintf("configure payments portable component: %v", err))
	}
	return configured
}
