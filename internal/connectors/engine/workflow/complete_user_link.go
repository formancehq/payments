package workflow

import (
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/workflow"
)

type CompleteUserLink struct {
	HTTPCallInformation models.HTTPCallInformation

	ConnectorID models.ConnectorID
	AttemptID   uuid.UUID
}

func (w Workflow) runCompleteUserLink(
	ctx workflow.Context,
	completeUserLink CompleteUserLink,
) error {
	return w.completeUserLink(
		infiniteRetryContext(ctx),
		completeUserLink,
	)
}

func (w Workflow) completeUserLink(
	ctx workflow.Context,
	completeUserLink CompleteUserLink,
) error {
	attempt, err := activities.StorageOpenBankingConnectionAttemptsGet(
		infiniteRetryContext(ctx),
		completeUserLink.AttemptID,
	)
	if err != nil {
		return err
	}

	resp, err := activities.PluginCompleteUserLink(
		infiniteRetryContext(ctx),
		completeUserLink.ConnectorID,
		models.CompleteUserLinkRequest{
			HTTPCallInformation: completeUserLink.HTTPCallInformation,
			RelatedAttempt:      attempt,
		},
	)
	if err != nil {
		return err
	}

	var pluginError error
	switch {
	case resp.Error != nil && resp.Error.Error != "":
		pluginError = errors.New(resp.Error.Error)
	case resp.Success == nil:
		pluginError = errors.New("unexpected response from plugin")
	default:
		pluginError = nil
	}

	if pluginError != nil {
		attempt.Error = pointer.For(pluginError.Error())
		attempt.Status = models.OpenBankingConnectionAttemptStatusExited

		err = activities.StorageOpenBankingConnectionAttemptsStore(
			infiniteRetryContext(ctx),
			*attempt,
		)
		if err != nil {
			return err
		}

		// Nothing else to do
		return nil
	}

	// Case of success
	attempt.Status = models.OpenBankingConnectionAttemptStatusCompleted
	err = activities.StorageOpenBankingConnectionAttemptsStore(
		infiniteRetryContext(ctx),
		*attempt,
	)
	if err != nil {
		return err
	}

	for _, connection := range resp.Success.Connections {
		c := models.FromPSPOpenBankingConnection(connection, completeUserLink.ConnectorID)
		c.Status = models.ConnectionStatusActive
		if err := activities.StorageOpenBankingConnectionsStore(
			infiniteRetryContext(ctx),
			attempt.PsuID,
			c,
		); err != nil {
			return err
		}
	}

	return nil
}

var RunCompleteUserLink = "RunCompleteUserLink"
