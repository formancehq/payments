package bankingcircle

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createBankAccountFromBankAccount(ba *models.BankAccount) (models.CreateBankAccountResponse, error) {
	// We can't create bank accounts in Banking Circle since they do not store
	// the bank account information. We just have to return the related formance
	// account in order to use it in the future.
	raw, err := json.Marshal(ba)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: models.PSPAccount{
			Reference: ba.ID.String(),
			CreatedAt: ba.CreatedAt,
			Name:      &ba.Name,
			Metadata:  ba.Metadata,
			Raw:       raw,
		},
	}, nil
}

func (p *Plugin) createBankAccountFromCounterParty(cp *models.PSPCounterParty) (models.CreateBankAccountResponse, error) {
	if cp.BankAccount == nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("counter party %s does not have a bank account: %w", cp.ID, models.ErrInvalidRequest)
	}

	raw, err := json.Marshal(cp)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: models.PSPAccount{
			Reference: cp.ID.String(),
			CreatedAt: cp.CreatedAt,
			Name:      &cp.Name,
			Metadata:  cp.Metadata,
			Raw:       raw,
		},
	}, nil
}
