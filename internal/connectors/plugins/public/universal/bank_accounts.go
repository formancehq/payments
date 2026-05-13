package universal

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.CreateBankAccountResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_CREATE_BANK_ACCOUNT); err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	ba := req.BankAccount
	resp, err := p.client.CreateBankAccount(ctx, ba.ID.String(), &client.BankAccountRequest{
		ID:            ba.ID.String(),
		CreatedAt:     ba.CreatedAt,
		Name:          ba.Name,
		AccountNumber: ba.AccountNumber,
		IBAN:          ba.IBAN,
		SwiftBicCode:  ba.SwiftBicCode,
		Country:       ba.Country,
		Metadata:      ba.Metadata,
	})
	if err != nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("creating bank account %s: %w", ba.ID, err)
	}
	related, err := mappers.AccountToPSPAccount(resp.RelatedAccount)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}
	return models.CreateBankAccountResponse{RelatedAccount: related}, nil
}
