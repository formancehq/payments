package filters

var Pools = Spec{
	Resource:     "Pools",
	EndpointPath: "/v3/pools",
	Fields: []Field{
		{Name: "name", Type: TypeString, Operators: OpMatch},
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "account_id", Type: TypeString, Operators: OpMatch,
			Description: "Filters pools that contain the given account in their members."},
	},
}
