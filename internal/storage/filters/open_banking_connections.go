package filters

var OpenBankingConnectionAttempts = Spec{
	Resource:     "OpenBankingConnectionAttempts",
	EndpointPath: "/v3/payment-service-users/{paymentServiceUserID}/connectors/{connectorID}/attempts",
	Fields: []Field{
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "status", Type: TypeRef, OpenAPIRef: "V3OpenBankingConnectionAttemptStatusEnum", Operators: OpMatch},
	},
}

var OpenBankingConnections = Spec{
	Resource:      "OpenBankingConnections",
	EndpointPath:  "/v3/payment-service-users/{paymentServiceUserID}/connections",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "connection_id", Type: TypeString, Operators: OpMatch},
		{Name: "status", Type: TypeRef, OpenAPIRef: "V3ConnectionStatusEnum", Operators: OpMatch},
	},
}

// Internal: no user-facing v3 endpoint accepts a query body for this resource.
var OpenBankingForwardedUsers = Spec{
	Resource:      "OpenBankingForwardedUsers",
	EndpointPath:  "/v3/payment-service-users/{paymentServiceUserID}/forwarded-users",
	Internal:      true,
	AllowMetadata: true,
	Fields: []Field{
		{Name: "connector_id", Type: TypeString, Operators: OpMatch},
		{Name: "psu_id", Type: TypeString, Operators: OpMatch},
	},
}
