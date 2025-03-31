// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type V3UpdateConnectorConfigRequest struct {
	// The connector ID
	ConnectorID              string                               `pathParam:"style=simple,explode=false,name=connectorID"`
	V3UpdateConnectorRequest *components.V3UpdateConnectorRequest `request:"mediaType=application/json"`
}

func (o *V3UpdateConnectorConfigRequest) GetConnectorID() string {
	if o == nil {
		return ""
	}
	return o.ConnectorID
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequest() *components.V3UpdateConnectorRequest {
	if o == nil {
		return nil
	}
	return o.V3UpdateConnectorRequest
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestAdyen() *components.V3AdyenConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3AdyenConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestAtlar() *components.V3AtlarConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3AtlarConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestBankingcircle() *components.V3BankingcircleConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3BankingcircleConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestCurrencycloud() *components.V3CurrencycloudConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3CurrencycloudConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestDummypay() *components.V3DummypayConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3DummypayConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestGeneric() *components.V3GenericConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3GenericConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestMangopay() *components.V3MangopayConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3MangopayConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestModulr() *components.V3ModulrConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3ModulrConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestMoneycorp() *components.V3MoneycorpConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3MoneycorpConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestStripe() *components.V3StripeConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3StripeConfig
	}
	return nil
}

func (o *V3UpdateConnectorConfigRequest) GetV3UpdateConnectorRequestWise() *components.V3WiseConfig {
	if v := o.GetV3UpdateConnectorRequest(); v != nil {
		return v.V3WiseConfig
	}
	return nil
}

type V3UpdateConnectorConfigResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *V3UpdateConnectorConfigResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *V3UpdateConnectorConfigResponse) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
