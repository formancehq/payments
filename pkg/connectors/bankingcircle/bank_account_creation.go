package bankingcircle

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createBankAccount(req connector.CreateBankAccountRequest) (connector.CreateBankAccountResponse, error) {
	// We can't create bank accounts in Banking Circle since they do not store
	// the bank account information. We just have to return the related formance
	// account in order to use it in the future.
	raw, err := json.Marshal(req.BankAccount)
	if err != nil {
		return connector.CreateBankAccountResponse{}, err
	}

	return connector.CreateBankAccountResponse{
		RelatedAccount: connector.PSPAccount{
			Reference: req.BankAccount.ID.String(),
			CreatedAt: req.BankAccount.CreatedAt,
			Name:      &req.BankAccount.Name,
			Metadata:  req.BankAccount.Metadata,
			Raw:       raw,
		},
	}, nil
}
