package filters

var PaymentInitiations = Spec{
	Resource:      "PaymentInitiations",
	EndpointPath:  "/v3/payment-initiations",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "reference", Type: TypeString, Operators: OpMatch},
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "connector_id", Type: TypeString, Operators: OpMatch},
		{Name: "type", Type: TypeRef, OpenAPIRef: "V3PaymentInitiationTypeEnum", Operators: OpMatch},
		{Name: "asset", Type: TypeString, Operators: OpMatch},
		{Name: "source_account_id", Type: TypeString, Operators: OpMatch},
		{Name: "destination_account_id", Type: TypeString, Operators: OpMatch},
		{Name: "status", Type: TypeRef, OpenAPIRef: "V3PaymentInitiationStatusEnum", Operators: OpMatch,
			Description: "Latest adjustment status of the payment initiation."},
		{Name: "amount", Type: TypeBigInt, Operators: OpMatch | OpComparison},
	},
}

var PaymentInitiationAdjustments = Spec{
	Resource:      "PaymentInitiationAdjustments",
	EndpointPath:  "/v3/payment-initiations/{paymentInitiationID}/adjustments",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "status", Type: TypeRef, OpenAPIRef: "V3PaymentInitiationStatusEnum", Operators: OpMatch},
	},
}
