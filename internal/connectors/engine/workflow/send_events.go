package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

const RunSendEvents = "SendEvents"

type SendEvents struct {
	Trade *models.Trade `json:"trade"`
}

func (w Workflow) runSendEvents(ctx workflow.Context, req SendEvents) error {
	if req.Trade != nil {
		return activities.EventsSendTrade(
			infiniteRetryContext(ctx),
			*req.Trade,
		)
	}
	return nil
}

