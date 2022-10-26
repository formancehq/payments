package connectors_test

import (
	"encoding/json"
	"fmt"

	"github.com/numary/payments/internal/pkg/connectors"
)

func ExampleAvailable_MarshalJSON() {
	a := connectors.Available{}
	by, err := json.MarshalIndent(a, "", "\t")
	if err != nil { //nolint:wsl
		panic(err)
	}

	fmt.Println(string(by))
	// Output:
	//{
	//	"dummypay": {
	//		"directory": {
	//			"datatype": "string",
	//			"required": true
	//		},
	//		"fileGenerationPeriod": {
	//			"datatype": "duration ns",
	//			"required": true
	//		},
	//		"filePollingPeriod": {
	//			"datatype": "duration ns",
	//			"required": true
	//		}
	//	},
	//	"modulr": {
	//		"apiKey": {
	//			"datatype": "string",
	//			"required": true
	//		},
	//		"apiSecret": {
	//			"datatype": "string",
	//			"required": true
	//		},
	//		"endpoint": {
	//			"datatype": "string",
	//			"required": false
	//		}
	//	},
	//	"stripe": {
	//		"apiKey": {
	//			"datatype": "string",
	//			"required": true
	//		},
	//		"pageSize": {
	//			"datatype": "unsigned integer",
	//			"required": false
	//		},
	//		"pollingPeriod": {
	//			"datatype": "duration ns",
	//			"required": false
	//		}
	//	},
	//	"wise": {
	//		"apiKey": {
	//			"datatype": "string",
	//			"required": true
	//		}
	//	}
	//}
}
