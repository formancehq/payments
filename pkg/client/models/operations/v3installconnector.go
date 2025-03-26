// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/models/components"
)

type V3InstallConnectorRequest struct {
	// The connector to filter by
	Connector                 string                                `pathParam:"style=simple,explode=false,name=connector"`
	V3InstallConnectorRequest *components.V3InstallConnectorRequest `request:"mediaType=application/json"`
}

func (o *V3InstallConnectorRequest) GetConnector() string {
	if o == nil {
		return ""
	}
	return o.Connector
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequest() *components.V3InstallConnectorRequest {
	if o == nil {
		return nil
	}
	return o.V3InstallConnectorRequest
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestAdyen() *components.V3AdyenConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3AdyenConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestAtlar() *components.V3AtlarConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3AtlarConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestBankingcircle() *components.V3BankingcircleConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3BankingcircleConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestCurrencycloud() *components.V3CurrencycloudConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3CurrencycloudConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestDummypay() *components.V3DummypayConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3DummypayConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestGeneric() *components.V3GenericConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3GenericConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestMangopay() *components.V3MangopayConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3MangopayConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestModulr() *components.V3ModulrConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3ModulrConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestMoneycorp() *components.V3MoneycorpConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3MoneycorpConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestStripe() *components.V3StripeConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3StripeConfig
	}
	return nil
}

func (o *V3InstallConnectorRequest) GetV3InstallConnectorRequestWise() *components.V3WiseConfig {
	if v := o.GetV3InstallConnectorRequest(); v != nil {
		return v.V3WiseConfig
	}
	return nil
}

type V3InstallConnectorResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// Accepted
	V3InstallConnectorResponse *components.V3InstallConnectorResponse
	// Error
	V3ErrorResponse *components.V3ErrorResponse
}

func (o *V3InstallConnectorResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *V3InstallConnectorResponse) GetV3InstallConnectorResponse() *components.V3InstallConnectorResponse {
	if o == nil {
		return nil
	}
	return o.V3InstallConnectorResponse
}

func (o *V3InstallConnectorResponse) GetV3ErrorResponse() *components.V3ErrorResponse {
	if o == nil {
		return nil
	}
	return o.V3ErrorResponse
}
