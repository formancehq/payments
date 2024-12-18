package engine

import (
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"go.temporal.io/api/enums/v1"
)

var (
	SearchAttributes = map[string]enums.IndexedValueType{
		workflow.SearchAttributeScheduleID: enums.INDEXED_VALUE_TYPE_KEYWORD,
		workflow.SearchAttributeStack:      enums.INDEXED_VALUE_TYPE_KEYWORD,
	}
)
