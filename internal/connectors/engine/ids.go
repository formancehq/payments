package engine

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

const (
	IDPrefixBankAccountCreate  = "create-bank-account"
	IDPrefixConnectorInstall   = "install"
	IDPrefixConnectorUninstall = "uninstall"
)

func (e *engine) taskIDReferenceFor(prefix string, connectorID models.ConnectorID, objectID string) string {
	withStack := fmt.Sprintf("%s-%s", prefix, e.stack)
	return models.TaskIDReference(withStack, connectorID, objectID)
}
