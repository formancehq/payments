// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

// AccountResponse - OK
type AccountResponse struct {
	Data Account `json:"data"`
}

func (o *AccountResponse) GetData() Account {
	if o == nil {
		return Account{}
	}
	return o.Data
}
