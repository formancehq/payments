// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

// TransferInitiationResponse - OK
type TransferInitiationResponse struct {
	Data TransferInitiation `json:"data"`
}

func (o *TransferInitiationResponse) GetData() TransferInitiation {
	if o == nil {
		return TransferInitiation{}
	}
	return o.Data
}
