package filters

var PaymentInitiationReversals = Spec{
	Resource:      "PaymentInitiationReversals",
	EndpointPath:  "/v3/payment-initiation-reversals",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "reference", Type: TypeString, Operators: OpMatch},
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "connector_id", Type: TypeString, Operators: OpMatch},
		{Name: "asset", Type: TypeString, Operators: OpMatch},
		{Name: "payment_initiation_id", Type: TypeString, Operators: OpMatch},
		{Name: "amount", Type: TypeBigInt, Operators: OpMatch | OpComparison},
	},
}
