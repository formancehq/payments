package core

import "github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"

// Table render hints are a compact projection over the payload this plugin
// already emits; the host keeps rendering the exhaustive payload for JSON and
// YAML. A column `Field` is a dot-separated JSON path into that payload: for a
// collection result it addresses one emitted item, and for an object result the
// emitted product envelope, whose single `data` member carries the entity.
//
// Only commands whose generated result type can back every declared column
// appear here. render_test.go pins the exact ordered columns, resolves every
// field against the generated Payments client, and records — with its evidence
// — each command deliberately left without hints.
var commandTableColumns = map[string][]sdk.TableColumn{
	"payments.v3.accounts.list": {
		{Header: "ID", Field: "id"}, {Header: "Provider", Field: "provider"}, {Header: "Reference", Field: "reference"},
		{Header: "Default Asset", Field: "defaultAsset"}, {Header: "Created At", Field: "createdAt"},
	},
	"payments.v3.accounts.balances": {
		{Header: "Asset", Field: "asset"}, {Header: "Balance", Field: "balance"},
		{Header: "Created At", Field: "createdAt"}, {Header: "Last Updated At", Field: "lastUpdatedAt"},
	},
	"payments.v3.bank_accounts.list": {
		{Header: "ID", Field: "id"}, {Header: "Name", Field: "name"},
		{Header: "Country", Field: "country"}, {Header: "Created At", Field: "createdAt"},
	},
	"payments.v3.connectors.list": {
		{Header: "ID", Field: "id"}, {Header: "Name", Field: "name"}, {Header: "Provider", Field: "provider"},
		{Header: "Reference", Field: "reference"}, {Header: "Created At", Field: "createdAt"},
	},
	"payments.v3.connectors.schedules.list": {
		{Header: "ID", Field: "id"}, {Header: "Connector ID", Field: "connectorID"},
		{Header: "Created At", Field: "createdAt"}, {Header: "Paused At", Field: "pausedAt"},
	},
	"payments.v3.connectors.schedules.instances.list": {
		{Header: "ID", Field: "id"}, {Header: "Schedule ID", Field: "scheduleID"}, {Header: "Created At", Field: "createdAt"},
		{Header: "Updated At", Field: "updatedAt"}, {Header: "Terminated", Field: "terminated"},
	},
	"payments.v3.payments.list": {
		{Header: "ID", Field: "id"}, {Header: "Type", Field: "type"}, {Header: "Amount", Field: "amount"},
		{Header: "Asset", Field: "asset"}, {Header: "Status", Field: "status"}, {Header: "Created At", Field: "createdAt"},
	},
	"payments.v3.transfer_initiation.list": {
		{Header: "ID", Field: "id"}, {Header: "Type", Field: "type"}, {Header: "Amount", Field: "amount"},
		{Header: "Asset", Field: "asset"}, {Header: "Status", Field: "status"}, {Header: "Scheduled At", Field: "scheduledAt"},
	},
	"payments.v3.pools.list": {
		{Header: "ID", Field: "id"}, {Header: "Name", Field: "name"},
		{Header: "Type", Field: "type"}, {Header: "Created At", Field: "createdAt"},
	},
	"payments.v3.orders.list": {
		{Header: "ID", Field: "id"}, {Header: "Direction", Field: "direction"}, {Header: "Type", Field: "type"},
		{Header: "Status", Field: "status"}, {Header: "Source Asset", Field: "sourceAsset"},
		{Header: "Destination Asset", Field: "destinationAsset"},
	},
	"payments.v3.conversions.list": {
		{Header: "ID", Field: "id"}, {Header: "Status", Field: "status"},
		{Header: "Source Asset", Field: "sourceAsset"}, {Header: "Source Amount", Field: "sourceAmount"},
		{Header: "Destination Asset", Field: "destinationAsset"}, {Header: "Destination Amount", Field: "destinationAmount"},
	},

	"payments.v3.accounts.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Provider", Field: "data.provider"}, {Header: "Reference", Field: "data.reference"},
		{Header: "Default Asset", Field: "data.defaultAsset"}, {Header: "Created At", Field: "data.createdAt"},
	},
	"payments.v3.accounts.create": {
		{Header: "ID", Field: "data.id"}, {Header: "Provider", Field: "data.provider"}, {Header: "Reference", Field: "data.reference"},
		{Header: "Default Asset", Field: "data.defaultAsset"}, {Header: "Created At", Field: "data.createdAt"},
	},
	"payments.v3.bank_accounts.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Name", Field: "data.name"},
		{Header: "Country", Field: "data.country"}, {Header: "Created At", Field: "data.createdAt"},
	},
	"payments.v3.connectors.schedules.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Connector ID", Field: "data.connectorID"},
		{Header: "Created At", Field: "data.createdAt"}, {Header: "Paused At", Field: "data.pausedAt"},
	},
	"payments.v3.payments.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Type", Field: "data.type"}, {Header: "Amount", Field: "data.amount"},
		{Header: "Asset", Field: "data.asset"}, {Header: "Status", Field: "data.status"}, {Header: "Created At", Field: "data.createdAt"},
	},
	"payments.v3.payments.create": {
		{Header: "ID", Field: "data.id"}, {Header: "Type", Field: "data.type"}, {Header: "Amount", Field: "data.amount"},
		{Header: "Asset", Field: "data.asset"}, {Header: "Status", Field: "data.status"}, {Header: "Created At", Field: "data.createdAt"},
	},
	"payments.v3.transfer_initiation.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Type", Field: "data.type"}, {Header: "Amount", Field: "data.amount"},
		{Header: "Asset", Field: "data.asset"}, {Header: "Status", Field: "data.status"}, {Header: "Scheduled At", Field: "data.scheduledAt"},
	},
	"payments.v3.pools.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Name", Field: "data.name"},
		{Header: "Type", Field: "data.type"}, {Header: "Created At", Field: "data.createdAt"},
	},
	"payments.v3.orders.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Direction", Field: "data.direction"}, {Header: "Type", Field: "data.type"},
		{Header: "Status", Field: "data.status"}, {Header: "Source Asset", Field: "data.sourceAsset"},
		{Header: "Destination Asset", Field: "data.destinationAsset"},
	},
	"payments.v3.conversions.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Status", Field: "data.status"},
		{Header: "Source Asset", Field: "data.sourceAsset"}, {Header: "Source Amount", Field: "data.sourceAmount"},
		{Header: "Destination Asset", Field: "data.destinationAsset"}, {Header: "Destination Amount", Field: "data.destinationAmount"},
	},
	"payments.v3.tasks.get": {
		{Header: "ID", Field: "data.id"}, {Header: "Status", Field: "data.status"}, {Header: "Connector ID", Field: "data.connectorID"},
		{Header: "Created At", Field: "data.createdAt"}, {Header: "Updated At", Field: "data.updatedAt"},
	},

	"payments.v3.bank_accounts.create": {{Header: "Bank Account ID", Field: "data"}},
	"payments.v3.connectors.install":   {{Header: "Connector ID", Field: "data"}},
	"payments.v3.pools.create":         {{Header: "Pool ID", Field: "data"}},

	"payments.v3.bank_accounts.forward": {{Header: "Task ID", Field: "data.taskID"}},
	"payments.v3.connectors.uninstall":  {{Header: "Task ID", Field: "data.taskID"}},
	"payments.v3.transfer_initiation.create": {
		{Header: "Payment Initiation ID", Field: "data.paymentInitiationID"}, {Header: "Task ID", Field: "data.taskID"},
	},
	"payments.v3.transfer_initiation.approve": {{Header: "Task ID", Field: "data.taskID"}},
	"payments.v3.transfer_initiation.retry":   {{Header: "Task ID", Field: "data.taskID"}},
	"payments.v3.transfer_initiation.reverse": {
		{Header: "Payment Initiation Reversal ID", Field: "data.paymentInitiationReversalID"}, {Header: "Task ID", Field: "data.taskID"},
	},
}

func renderHints(commandID string) sdk.RenderHints {
	columns, ok := commandTableColumns[commandID]
	if !ok {
		return sdk.RenderHints{}
	}
	return sdk.RenderHints{Table: &sdk.TableRenderHint{Columns: append([]sdk.TableColumn(nil), columns...)}}
}
