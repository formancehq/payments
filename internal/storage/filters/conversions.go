package filters

var Conversions = Spec{
	Resource:      "Conversions",
	EndpointPath:  "/v3/conversions",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "reference", Type: TypeString, Operators: OpMatch},
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "connector_id", Type: TypeString, Operators: OpMatch},
		{Name: "source_asset", Type: TypeString, Operators: OpMatch},
		{Name: "destination_asset", Type: TypeString, Operators: OpMatch},
		{Name: "status", Type: TypeRef, OpenAPIRef: "V3ConversionStatusEnum", Operators: OpMatch},
		{Name: "source_account_id", Type: TypeString, Operators: OpMatch},
		{Name: "destination_account_id", Type: TypeString, Operators: OpMatch},
		{Name: "source_amount", Type: TypeBigInt, Operators: OpMatch | OpComparison},
		{Name: "destination_amount", Type: TypeBigInt, Operators: OpMatch | OpComparison},
	},
}
