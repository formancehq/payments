package connectors

import (
	"encoding/json"

	"github.com/numary/payments/internal/pkg/connectors/dummypay"
	"github.com/numary/payments/internal/pkg/connectors/modulr"
	"github.com/numary/payments/internal/pkg/connectors/stripe"
	"github.com/numary/payments/internal/pkg/connectors/wise"
)

type Available struct {
	Dummypay dummypay.Config `json:"dummypay"`
	Modulr   modulr.Config   `json:"modulr"`
	Stripe   stripe.Config   `json:"stripe"`
	Wise     wise.Config     `json:"wise"`
}

type AvailableInfos struct {
	Dummypay map[string]Infos `json:"dummypay"`
	Modulr   map[string]Infos `json:"modulr"`
	Stripe   map[string]Infos `json:"stripe"`
	Wise     map[string]Infos `json:"wise"`
}

type Infos struct {
	DataType string `json:"datatype"`
	Required bool   `json:"required"`
}

func (c Available) MarshalJSON() ([]byte, error) {
	s := AvailableInfos{
		Dummypay: map[string]Infos{
			"directory": {
				DataType: "string",
				Required: true,
			},
			"filePollingPeriod": {
				DataType: "duration ns",
				Required: true,
			},
			"fileGenerationPeriod": {
				DataType: "duration ns",
				Required: true,
			},
		},
		Modulr: map[string]Infos{
			"apiKey": {
				DataType: "string",
				Required: true,
			},
			"apiSecret": {
				DataType: "string",
				Required: true,
			},
			"endpoint": {
				DataType: "string",
				Required: false,
			},
		},
		Stripe: map[string]Infos{
			"pollingPeriod": {
				DataType: "duration ns",
				Required: false,
			},
			"apiKey": {
				DataType: "string",
				Required: true,
			},
			"pageSize": {
				DataType: "unsigned integer",
				Required: false,
			},
		},
		Wise: map[string]Infos{
			"apiKey": {
				DataType: "string",
				Required: true,
			},
		},
	}

	return json.Marshal(s)
}
