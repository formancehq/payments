package v3

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type BankAccountsForwardToConnectorRequest struct {
	ConnectorID string `json:"connectorID"`
}

func (f *BankAccountsForwardToConnectorRequest) Validate() error {
	if f.ConnectorID == "" {
		return errors.New("connectorID must be provided")
	}

	return nil
}

type BankAccountsForwardToConnectorResponse struct {
	TaskID string `json:"taskID"`
}

func bankAccountsForwardToConnector(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_bankAccountsForwardToConnector")
		defer span.End()

		span.SetAttributes(attribute.String("bankAccountID", bankAccountID(r)))
		id, err := uuid.Parse(bankAccountID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		var req BankAccountsForwardToConnectorRequest
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		span.SetAttributes(attribute.String("connectorID", req.ConnectorID))

		err = req.Validate()
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		connectorID, err := models.ConnectorIDFromString(req.ConnectorID)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		task, err := backend.BankAccountsForwardToConnector(ctx, id, connectorID, false)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Accepted(w, BankAccountsForwardToConnectorResponse{
			TaskID: task.ID.String(),
		})
	}
}
