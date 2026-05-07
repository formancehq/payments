package filters

var Orders = Spec{
	Resource:      "Orders",
	EndpointPath:  "/v3/orders",
	AllowMetadata: true,
	Fields: []Field{
		{Name: "reference", Type: TypeString, Operators: OpMatch},
		{Name: "id", Type: TypeString, Operators: OpMatch},
		{Name: "connector_id", Type: TypeString, Operators: OpMatch},
		{Name: "direction", Type: TypeRef, OpenAPIRef: "V3OrderDirectionEnum", Operators: OpMatch},
		{Name: "source_asset", Type: TypeString, Operators: OpMatch},
		{Name: "destination_asset", Type: TypeString, Operators: OpMatch},
		{Name: "type", Type: TypeRef, OpenAPIRef: "V3OrderTypeEnum", Operators: OpMatch},
		{Name: "status", Type: TypeRef, OpenAPIRef: "V3OrderStatusEnum", Operators: OpMatch},
		{Name: "time_in_force", Type: TypeRef, OpenAPIRef: "V3TimeInForceEnum", Operators: OpMatch},
		{Name: "base_quantity_ordered", Type: TypeBigInt, Operators: OpMatch | OpComparison},
		{Name: "base_quantity_filled", Type: TypeBigInt, Operators: OpMatch | OpComparison},
		{Name: "limit_price", Type: TypeBigInt, Operators: OpMatch | OpComparison},
		{Name: "fee", Type: TypeBigInt, Operators: OpMatch | OpComparison},
	},
}
