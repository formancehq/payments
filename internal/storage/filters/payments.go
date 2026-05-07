package filters

var Payments = Spec{
	Resource:      "Payments",
	EndpointPath:  "/v3/payments",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "reference", Type: TypeString, Operators: OpMatch},
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "connector_id", Type: TypeString, Operators: OpMatch},
		{Name: "type", Type: TypeRef, OpenAPIRef: "V3PaymentTypeEnum", Operators: OpMatch},
		{Name: "asset", Type: TypeString, Operators: OpMatch},
		{Name: "scheme", Type: TypeString, Operators: OpMatch},
		{Name: "status", Type: TypeRef, OpenAPIRef: "V3PaymentStatusEnum", Operators: OpMatch},
		{Name: "source_account_id", Type: TypeString, Operators: OpMatch},
		{Name: "destination_account_id", Type: TypeString, Operators: OpMatch},
		{Name: "psu_id", Type: TypeString, Operators: OpMatch},
		{Name: "open_banking_connection_id", Type: TypeString, Operators: OpMatch},
		{Name: "initial_amount", Type: TypeBigInt, Operators: OpMatch | OpComparison},
		{Name: "amount", Type: TypeBigInt, Operators: OpMatch | OpComparison},
	},
}
