package engine

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
)

// scopedAccountLookup is the engine's implementation of models.AccountLookup.
// Each instance is bound to a single connector ID: it can only return
// accounts belonging to that connector. The binding is enforced by the
// storage query, not by in-memory filtering.
type scopedAccountLookup struct {
	storage     storage.Storage
	connectorID models.ConnectorID
}

func newScopedAccountLookup(s storage.Storage, connectorID models.ConnectorID) *scopedAccountLookup {
	return &scopedAccountLookup{storage: s, connectorID: connectorID}
}

func (l *scopedAccountLookup) ListAccountsByConnector(ctx context.Context) ([]models.PSPAccount, error) {
	accounts, err := l.storage.AccountsListAllByConnectorID(ctx, l.connectorID)
	if err != nil {
		return nil, err
	}

	pspAccounts := make([]models.PSPAccount, 0, len(accounts))
	for i := range accounts {
		psp := models.ToPSPAccount(&accounts[i])
		if psp == nil {
			continue
		}
		pspAccounts = append(pspAccounts, *psp)
	}
	return pspAccounts, nil
}

// NewAccountLookupFactory returns a factory that the connectors manager can
// use to produce a per-connector AccountLookup at plugin-load time.
func NewAccountLookupFactory(s storage.Storage) models.AccountLookupFactory {
	return func(connectorID models.ConnectorID) models.AccountLookup {
		return newScopedAccountLookup(s, connectorID)
	}
}
