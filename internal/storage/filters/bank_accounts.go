package filters

var BankAccounts = Spec{
	Resource:      "BankAccounts",
	EndpointPath:  "/v3/bank-accounts",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "name", Type: TypeString, Operators: OpMatch},
		{Name: "country", Type: TypeString, Operators: OpMatch},
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "psu_id", Type: TypeString, Operators: OpMatch},
	},
}
