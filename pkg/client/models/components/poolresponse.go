// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

// PoolResponse - OK
type PoolResponse struct {
	Data Pool `json:"data"`
}

func (o *PoolResponse) GetData() Pool {
	if o == nil {
		return Pool{}
	}
	return o.Data
}
