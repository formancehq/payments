package filters

var PaymentServiceUsers = Spec{
	Resource:      "PaymentServiceUsers",
	EndpointPath:  "/v3/payment-service-users",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "id", Type: TypeString, Operators: OpMatch},
	},
}
