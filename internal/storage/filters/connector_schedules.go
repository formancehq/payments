package filters

var ConnectorSchedules = Spec{
	Resource:     "ConnectorSchedules",
	EndpointPath: "/v3/connectors/{connectorID}/schedules",
	Fields: []Field{
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "connector_id", Type: TypeString, Operators: OpMatch},
	},
}

// Internal: handler builds the query from URL path parameters and ignores any
// user-supplied body, so no OpenAPI component is generated.
var ConnectorScheduleInstances = Spec{
	Resource:     "ConnectorScheduleInstances",
	EndpointPath: "/v3/connectors/{connectorID}/schedules/{scheduleID}/instances",
	Internal:     true,
	Fields: []Field{
		{Name: "schedule_id", Type: TypeString, Operators: OpMatch},
		{Name: "connector_id", Type: TypeString, Operators: OpMatch},
	},
}
