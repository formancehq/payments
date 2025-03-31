// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package operations

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"github.com/formancehq/payments/pkg/client/models/components"
)

type ListBankAccountsRequest struct {
	// The maximum number of results to return per page.
	//
	PageSize *int64 `default:"15" queryParam:"style=form,explode=true,name=pageSize"`
	// Parameter used in pagination requests. Maximum page size is set to 15.
	// Set to the value of next for the next page of results.
	// Set to the value of previous for the previous page of results.
	// No other parameters can be set when this parameter is set.
	//
	Cursor *string `queryParam:"style=form,explode=true,name=cursor"`
	// Fields used to sort payments (default is date:desc).
	Sort []string `queryParam:"style=form,explode=true,name=sort"`
}

func (l ListBankAccountsRequest) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(l, "", false)
}

func (l *ListBankAccountsRequest) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &l, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *ListBankAccountsRequest) GetPageSize() *int64 {
	if o == nil {
		return nil
	}
	return o.PageSize
}

func (o *ListBankAccountsRequest) GetCursor() *string {
	if o == nil {
		return nil
	}
	return o.Cursor
}

func (o *ListBankAccountsRequest) GetSort() []string {
	if o == nil {
		return nil
	}
	return o.Sort
}

type ListBankAccountsResponse struct {
	HTTPMeta components.HTTPMetadata `json:"-"`
	// OK
	BankAccountsCursor *components.BankAccountsCursor
	// Error
	PaymentsErrorResponse *components.PaymentsErrorResponse
}

func (o *ListBankAccountsResponse) GetHTTPMeta() components.HTTPMetadata {
	if o == nil {
		return components.HTTPMetadata{}
	}
	return o.HTTPMeta
}

func (o *ListBankAccountsResponse) GetBankAccountsCursor() *components.BankAccountsCursor {
	if o == nil {
		return nil
	}
	return o.BankAccountsCursor
}

func (o *ListBankAccountsResponse) GetPaymentsErrorResponse() *components.PaymentsErrorResponse {
	if o == nil {
		return nil
	}
	return o.PaymentsErrorResponse
}
