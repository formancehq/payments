package filters

var Accounts = Spec{
	Resource:      "Accounts",
	EndpointPath:  "/v3/accounts",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "id", Type: TypeString, Operators: OpMatch | OpComparison},
		{Name: "reference", Type: TypeString, Operators: OpMatch | OpComparison},
		{Name: "connector_id", Type: TypeString, Operators: OpMatch | OpComparison},
		{Name: "type", Type: TypeRef, OpenAPIRef: "V3AccountTypeEnum", Operators: OpMatch | OpComparison},
		{Name: "default_asset", Type: TypeString, Operators: OpMatch | OpComparison},
		{Name: "name", Type: TypeString, Operators: OpMatch | OpComparison},
		{Name: "psu_id", Type: TypeString, Operators: OpMatch | OpComparison},
		{Name: "open_banking_connection_id", Type: TypeString, Operators: OpMatch | OpComparison},
	},
}
