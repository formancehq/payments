package core

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	paymentscomponents "github.com/formancehq/payments/pkg/client/models/components"
)

// column is one expected rendered table column.
type column struct{ header, field string }

// resultShape names the Go type the emitted result payload decodes into, so a
// declared table column can be proven against the generated client rather than
// against prose. For a collection command the root is the cursor item type
// because the adapter emits the concatenated `cursor.data` items; for an object
// command the root is the full product response envelope because the adapter
// emits the product body verbatim.
type resultShape struct {
	root      reflect.Type
	collected bool
}

func wantTableColumns() map[string][]column {
	return map[string][]column{
		// Collections: fields address one cursor item.
		"payments.v3.accounts.list": {
			{"ID", "id"}, {"Provider", "provider"}, {"Reference", "reference"},
			{"Default Asset", "defaultAsset"}, {"Created At", "createdAt"},
		},
		"payments.v3.accounts.balances": {
			{"Asset", "asset"}, {"Balance", "balance"},
			{"Created At", "createdAt"}, {"Last Updated At", "lastUpdatedAt"},
		},
		"payments.v3.bank_accounts.list": {
			{"ID", "id"}, {"Name", "name"}, {"Country", "country"}, {"Created At", "createdAt"},
		},
		"payments.v3.connectors.list": {
			{"ID", "id"}, {"Name", "name"}, {"Provider", "provider"},
			{"Reference", "reference"}, {"Created At", "createdAt"},
		},
		"payments.v3.connectors.schedules.list": {
			{"ID", "id"}, {"Connector ID", "connectorID"},
			{"Created At", "createdAt"}, {"Paused At", "pausedAt"},
		},
		"payments.v3.connectors.schedules.instances.list": {
			{"ID", "id"}, {"Schedule ID", "scheduleID"}, {"Created At", "createdAt"},
			{"Updated At", "updatedAt"}, {"Terminated", "terminated"},
		},
		"payments.v3.payments.list": {
			{"ID", "id"}, {"Type", "type"}, {"Amount", "amount"},
			{"Asset", "asset"}, {"Status", "status"}, {"Created At", "createdAt"},
		},
		"payments.v3.transfer_initiation.list": {
			{"ID", "id"}, {"Type", "type"}, {"Amount", "amount"},
			{"Asset", "asset"}, {"Status", "status"}, {"Scheduled At", "scheduledAt"},
		},
		"payments.v3.pools.list": {
			{"ID", "id"}, {"Name", "name"}, {"Type", "type"}, {"Created At", "createdAt"},
		},
		"payments.v3.orders.list": {
			{"ID", "id"}, {"Direction", "direction"}, {"Type", "type"}, {"Status", "status"},
			{"Source Asset", "sourceAsset"}, {"Destination Asset", "destinationAsset"},
		},
		"payments.v3.conversions.list": {
			{"ID", "id"}, {"Status", "status"},
			{"Source Asset", "sourceAsset"}, {"Source Amount", "sourceAmount"},
			{"Destination Asset", "destinationAsset"}, {"Destination Amount", "destinationAmount"},
		},

		// Objects carrying one entity: fields address the product envelope.
		"payments.v3.accounts.get": {
			{"ID", "data.id"}, {"Provider", "data.provider"}, {"Reference", "data.reference"},
			{"Default Asset", "data.defaultAsset"}, {"Created At", "data.createdAt"},
		},
		"payments.v3.accounts.create": {
			{"ID", "data.id"}, {"Provider", "data.provider"}, {"Reference", "data.reference"},
			{"Default Asset", "data.defaultAsset"}, {"Created At", "data.createdAt"},
		},
		"payments.v3.bank_accounts.get": {
			{"ID", "data.id"}, {"Name", "data.name"},
			{"Country", "data.country"}, {"Created At", "data.createdAt"},
		},
		"payments.v3.connectors.schedules.get": {
			{"ID", "data.id"}, {"Connector ID", "data.connectorID"},
			{"Created At", "data.createdAt"}, {"Paused At", "data.pausedAt"},
		},
		"payments.v3.payments.get": {
			{"ID", "data.id"}, {"Type", "data.type"}, {"Amount", "data.amount"},
			{"Asset", "data.asset"}, {"Status", "data.status"}, {"Created At", "data.createdAt"},
		},
		"payments.v3.payments.create": {
			{"ID", "data.id"}, {"Type", "data.type"}, {"Amount", "data.amount"},
			{"Asset", "data.asset"}, {"Status", "data.status"}, {"Created At", "data.createdAt"},
		},
		"payments.v3.transfer_initiation.get": {
			{"ID", "data.id"}, {"Type", "data.type"}, {"Amount", "data.amount"},
			{"Asset", "data.asset"}, {"Status", "data.status"}, {"Scheduled At", "data.scheduledAt"},
		},
		"payments.v3.pools.get": {
			{"ID", "data.id"}, {"Name", "data.name"},
			{"Type", "data.type"}, {"Created At", "data.createdAt"},
		},
		"payments.v3.orders.get": {
			{"ID", "data.id"}, {"Direction", "data.direction"}, {"Type", "data.type"},
			{"Status", "data.status"}, {"Source Asset", "data.sourceAsset"},
			{"Destination Asset", "data.destinationAsset"},
		},
		"payments.v3.conversions.get": {
			{"ID", "data.id"}, {"Status", "data.status"},
			{"Source Asset", "data.sourceAsset"}, {"Source Amount", "data.sourceAmount"},
			{"Destination Asset", "data.destinationAsset"}, {"Destination Amount", "data.destinationAmount"},
		},
		"payments.v3.tasks.get": {
			{"ID", "data.id"}, {"Status", "data.status"}, {"Connector ID", "data.connectorID"},
			{"Created At", "data.createdAt"}, {"Updated At", "data.updatedAt"},
		},

		// Objects whose `data` is the created identifier.
		"payments.v3.bank_accounts.create": {{"Bank Account ID", "data"}},
		"payments.v3.connectors.install":   {{"Connector ID", "data"}},
		"payments.v3.pools.create":         {{"Pool ID", "data"}},

		// Objects tracking asynchronous work.
		"payments.v3.bank_accounts.forward": {{"Task ID", "data.taskID"}},
		"payments.v3.connectors.uninstall":  {{"Task ID", "data.taskID"}},
		"payments.v3.transfer_initiation.create": {
			{"Payment Initiation ID", "data.paymentInitiationID"}, {"Task ID", "data.taskID"},
		},
		"payments.v3.transfer_initiation.approve": {{"Task ID", "data.taskID"}},
		"payments.v3.transfer_initiation.retry":   {{"Task ID", "data.taskID"}},
		"payments.v3.transfer_initiation.reverse": {
			{"Payment Initiation Reversal ID", "data.paymentInitiationReversalID"}, {"Task ID", "data.taskID"},
		},
	}
}

func wantResultShapes() map[string]resultShape {
	return map[string]resultShape{
		"payments.v3.accounts.list":                       {reflect.TypeOf(paymentscomponents.V3Account{}), true},
		"payments.v3.accounts.balances":                   {reflect.TypeOf(paymentscomponents.V3Balance{}), true},
		"payments.v3.bank_accounts.list":                  {reflect.TypeOf(paymentscomponents.V3BankAccount{}), true},
		"payments.v3.connectors.list":                     {reflect.TypeOf(paymentscomponents.V3Connector{}), true},
		"payments.v3.connectors.schedules.list":           {reflect.TypeOf(paymentscomponents.V3Schedule{}), true},
		"payments.v3.connectors.schedules.instances.list": {reflect.TypeOf(paymentscomponents.V3Instance{}), true},
		"payments.v3.payments.list":                       {reflect.TypeOf(paymentscomponents.V3Payment{}), true},
		"payments.v3.transfer_initiation.list":            {reflect.TypeOf(paymentscomponents.V3PaymentInitiation{}), true},
		"payments.v3.pools.list":                          {reflect.TypeOf(paymentscomponents.V3Pool{}), true},
		"payments.v3.orders.list":                         {reflect.TypeOf(paymentscomponents.V3Order{}), true},
		"payments.v3.conversions.list":                    {reflect.TypeOf(paymentscomponents.V3Conversion{}), true},

		"payments.v3.accounts.get":                {reflect.TypeOf(paymentscomponents.V3GetAccountResponse{}), false},
		"payments.v3.accounts.create":             {reflect.TypeOf(paymentscomponents.V3CreateAccountResponse{}), false},
		"payments.v3.bank_accounts.get":           {reflect.TypeOf(paymentscomponents.V3GetBankAccountResponse{}), false},
		"payments.v3.bank_accounts.create":        {reflect.TypeOf(paymentscomponents.V3CreateBankAccountResponse{}), false},
		"payments.v3.bank_accounts.forward":       {reflect.TypeOf(paymentscomponents.V3ForwardBankAccountResponse{}), false},
		"payments.v3.connectors.install":          {reflect.TypeOf(paymentscomponents.V3InstallConnectorResponse{}), false},
		"payments.v3.connectors.uninstall":        {reflect.TypeOf(paymentscomponents.V3UninstallConnectorResponse{}), false},
		"payments.v3.connectors.schedules.get":    {reflect.TypeOf(paymentscomponents.V3ConnectorScheduleResponse{}), false},
		"payments.v3.payments.get":                {reflect.TypeOf(paymentscomponents.V3GetPaymentResponse{}), false},
		"payments.v3.payments.create":             {reflect.TypeOf(paymentscomponents.V3CreatePaymentResponse{}), false},
		"payments.v3.transfer_initiation.get":     {reflect.TypeOf(paymentscomponents.V3GetPaymentInitiationResponse{}), false},
		"payments.v3.transfer_initiation.create":  {reflect.TypeOf(paymentscomponents.V3InitiatePaymentResponse{}), false},
		"payments.v3.transfer_initiation.approve": {reflect.TypeOf(paymentscomponents.V3ApprovePaymentInitiationResponse{}), false},
		"payments.v3.transfer_initiation.retry":   {reflect.TypeOf(paymentscomponents.V3RetryPaymentInitiationResponse{}), false},
		"payments.v3.transfer_initiation.reverse": {reflect.TypeOf(paymentscomponents.V3ReversePaymentInitiationResponse{}), false},
		"payments.v3.pools.get":                   {reflect.TypeOf(paymentscomponents.V3GetPoolResponse{}), false},
		"payments.v3.pools.create":                {reflect.TypeOf(paymentscomponents.V3CreatePoolResponse{}), false},
		"payments.v3.orders.get":                  {reflect.TypeOf(paymentscomponents.V3GetOrderResponse{}), false},
		"payments.v3.conversions.get":             {reflect.TypeOf(paymentscomponents.V3GetConversionResponse{}), false},
		"payments.v3.tasks.get":                   {reflect.TypeOf(paymentscomponents.V3GetTaskResponse{}), false},
	}
}

// commandsWithoutTableRenderHints records, with its evidence, every command
// whose result shape cannot back a truthful fixed column set. Nothing is
// invented for these: the host still renders their exhaustive JSON and YAML.
func commandsWithoutTableRenderHints() map[string]string {
	return map[string]string{
		"payments.v3.bank_accounts.update-metadata": "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.payments.set-metadata":         "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.transfer_initiation.delete":    "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.transfer_initiation.reject":    "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.pools.delete":                  "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.pools.update-query":            "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.pools.add-account":             "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.pools.remove-account":          "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.connectors.update-config":      "204 No Content: the adapter emits `{}`, which has no properties",
		"payments.v3.pools.balances":                "object result whose `data` is a V3PoolBalance array; a fixed column cannot address an unindexed array element",
		"payments.v3.pools.latest-balances":         "object result whose `data` is a V3PoolBalance array; a fixed column cannot address an unindexed array element",
		"payments.v3.connectors.list-available":     "object result whose `data` is a provider-keyed map of configuration schemas; column fields cannot be fixed across providers",
		"payments.v3.connectors.get-config":         "credential-bearing connector configuration union; every truthful column would be provider-specific and the payload is redacted, so no stable column set exists",
	}
}

func TestCatalogueDeclaresExactOrderedTableColumns(t *testing.T) {
	want := wantTableColumns()
	for _, command := range Catalogue() {
		expected, hinted := want[command.ID]
		if !hinted {
			if command.Render.Table != nil {
				t.Errorf("%s declares table render hints but is recorded as unsupported", command.ID)
			}
			continue
		}
		if command.Render.Table == nil {
			t.Errorf("%s declares no table render hint", command.ID)
			continue
		}
		got := make([]column, 0, len(command.Render.Table.Columns))
		for _, declared := range command.Render.Table.Columns {
			got = append(got, column{declared.Header, declared.Field})
		}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("%s columns = %#v, want %#v", command.ID, got, expected)
		}
	}
}

func TestEveryCommandIsEitherRenderedOrExplicitlyRecordedAsUnsupported(t *testing.T) {
	hinted, unsupported := wantTableColumns(), commandsWithoutTableRenderHints()
	commands := Catalogue()
	if got, want := len(hinted)+len(unsupported), len(commands); got != want {
		t.Fatalf("rendered %d + unsupported %d = %d, want %d commands", len(hinted), len(unsupported), got, want)
	}
	for _, command := range commands {
		_, rendered := hinted[command.ID]
		reason, recorded := unsupported[command.ID]
		switch {
		case rendered && recorded:
			t.Errorf("%s is both rendered and recorded as unsupported", command.ID)
		case !rendered && !recorded:
			t.Errorf("%s is neither rendered nor recorded as unsupported", command.ID)
		case recorded && strings.TrimSpace(reason) == "":
			t.Errorf("%s is recorded as unsupported without a reason", command.ID)
		case recorded && command.Render.Table != nil:
			t.Errorf("%s is recorded as unsupported but declares table render hints", command.ID)
		}
	}
	if t.Failed() {
		return
	}
	missing := make([]string, 0)
	for id := range unsupported {
		found := false
		for _, command := range commands {
			if command.ID == id {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("unsupported record names commands that do not exist: %v", missing)
	}
}

func TestEveryTableColumnFieldResolvesToAScalarGeneratedProperty(t *testing.T) {
	shapes := wantResultShapes()
	for _, command := range Catalogue() {
		if command.Render.Table == nil {
			continue
		}
		shape, ok := shapes[command.ID]
		if !ok {
			t.Errorf("%s declares table render hints without a recorded result shape", command.ID)
			continue
		}
		wantCollection := command.Pagination.Supported
		if shape.collected != wantCollection {
			t.Errorf("%s recorded collection = %t, catalogue pagination = %t", command.ID, shape.collected, wantCollection)
		}
		seen := map[string]struct{}{}
		for _, declared := range command.Render.Table.Columns {
			if _, exists := seen[declared.Field]; exists {
				t.Errorf("%s repeats column field %q", command.ID, declared.Field)
			}
			seen[declared.Field] = struct{}{}
			resolved, err := resolveGeneratedField(shape.root, declared.Field)
			if err != nil {
				t.Errorf("%s column %q: %v", command.ID, declared.Field, err)
				continue
			}
			if !isScalarGeneratedType(resolved) {
				t.Errorf("%s column %q resolves to non-scalar %s", command.ID, declared.Field, resolved)
			}
			if isCredentialFieldName(lastSegment(declared.Field)) {
				t.Errorf("%s column %q addresses a credential-bearing property", command.ID, declared.Field)
			}
		}
	}
}

func resolveGeneratedField(root reflect.Type, field string) (reflect.Type, error) {
	current := root
	for _, segment := range strings.Split(field, ".") {
		for current.Kind() == reflect.Pointer {
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return nil, fmt.Errorf("segment %q cannot be resolved on non-struct %s", segment, current)
		}
		next, ok := generatedFieldByJSONName(current, segment)
		if !ok {
			return nil, fmt.Errorf("segment %q is not a JSON property of %s", segment, current)
		}
		current = next
	}
	for current.Kind() == reflect.Pointer {
		current = current.Elem()
	}
	return current, nil
}

func generatedFieldByJSONName(structType reflect.Type, name string) (reflect.Type, bool) {
	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		if strings.Split(field.Tag.Get("json"), ",")[0] == name {
			return field.Type, true
		}
	}
	return nil, false
}

func isScalarGeneratedType(candidate reflect.Type) bool {
	switch candidate {
	case reflect.TypeOf(time.Time{}), reflect.TypeOf(big.Int{}):
		return true
	}
	switch candidate.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func lastSegment(field string) string {
	parts := strings.Split(field, ".")
	return parts[len(parts)-1]
}

func TestRenderHintsRemainValidUnderTheSDKCatalogueContract(t *testing.T) {
	if err := sdk.ValidateCommands(Catalogue()); err != nil {
		t.Fatalf("catalogue with render hints is invalid: %v", err)
	}
}
