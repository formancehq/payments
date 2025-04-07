// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type V3ConnectorConfigsResponseData struct {
	DataType     string  `json:"dataType"`
	Required     bool    `json:"required"`
	DefaultValue *string `json:"defaultValue,omitempty"`
}

func (o *V3ConnectorConfigsResponseData) GetDataType() string {
	if o == nil {
		return ""
	}
	return o.DataType
}

func (o *V3ConnectorConfigsResponseData) GetRequired() bool {
	if o == nil {
		return false
	}
	return o.Required
}

func (o *V3ConnectorConfigsResponseData) GetDefaultValue() *string {
	if o == nil {
		return nil
	}
	return o.DefaultValue
}

type V3ConnectorConfigsResponse struct {
	Data map[string]map[string]V3ConnectorConfigsResponseData `json:"data"`
}

func (o *V3ConnectorConfigsResponse) GetData() map[string]map[string]V3ConnectorConfigsResponseData {
	if o == nil {
		return map[string]map[string]V3ConnectorConfigsResponseData{}
	}
	return o.Data
}
