package engine

import (
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func GetDefaultTaskQueue(stack string) string {
	return fmt.Sprintf("%s-default", stack)
}

func GetPayoutTaskQueue(stack string, connectorID models.ConnectorID) string {
	return fmt.Sprintf("%s-%s-payout", stack, connectorID.String())
}
