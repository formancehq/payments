package engine

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func getDefaultTaskQueue(stack string) string {
	return fmt.Sprintf("%s-default", stack)
}

func getConnectorTaskQueue(stack string, connectorID models.ConnectorID) string {
	return fmt.Sprintf("%s-%s", stack, connectorID.String())
}
