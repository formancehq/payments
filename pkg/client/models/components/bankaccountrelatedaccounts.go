// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/payments/pkg/client/internal/utils"
	"time"
)

type BankAccountRelatedAccounts struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	Provider    string    `json:"provider"`
	ConnectorID string    `json:"connectorID"`
	AccountID   string    `json:"accountID"`
}

func (b BankAccountRelatedAccounts) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(b, "", false)
}

func (b *BankAccountRelatedAccounts) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &b, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *BankAccountRelatedAccounts) GetID() string {
	if o == nil {
		return ""
	}
	return o.ID
}

func (o *BankAccountRelatedAccounts) GetCreatedAt() time.Time {
	if o == nil {
		return time.Time{}
	}
	return o.CreatedAt
}

func (o *BankAccountRelatedAccounts) GetProvider() string {
	if o == nil {
		return ""
	}
	return o.Provider
}

func (o *BankAccountRelatedAccounts) GetConnectorID() string {
	if o == nil {
		return ""
	}
	return o.ConnectorID
}

func (o *BankAccountRelatedAccounts) GetAccountID() string {
	if o == nil {
		return ""
	}
	return o.AccountID
}
