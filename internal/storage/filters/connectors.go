package filters

var Connectors = Spec{
	Resource:     "Connectors",
	EndpointPath: "/v3/connectors",
	Fields: []Field{
		{Name: "provider", Type: TypeString, Operators: OpMatch | OpComparison,
			Description: "Connector provider name. Compared case-insensitively."},
		{Name: "name", Type: TypeString, Operators: OpMatch | OpComparison},
		{Name: "id", Type: TypeString, Operators: OpMatch | OpComparison},
	},
}
