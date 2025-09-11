package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

func paymentServiceUsersLinkAttemptList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentServiceUsersLinkAttemptGet")
		defer span.End()

		span.SetAttributes(attribute.String("paymentServiceUserID", paymentServiceUserID(r)))
		psuID, err := uuid.Parse(paymentServiceUserID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		query, err := bunpaginate.Extract[storage.ListOpenBankingConnectionAttemptsQuery](r, func() (*storage.ListOpenBankingConnectionAttemptsQuery, error) {
			options, err := getPagination(span, r, storage.OpenBankingConnectionAttemptsQuery{})
			if err != nil {
				return nil, err
			}
			return pointer.For(storage.NewListOpenBankingConnectionAttemptsQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		cursor, err := backend.PaymentServiceUsersLinkAttemptsList(ctx, psuID, connectorID, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		attempts := make([]attemptResponse, len(cursor.Data))
		for i, attempt := range cursor.Data {
			attempts[i] = attemptResponse{
				ID:                attempt.ID,
				PsuID:             attempt.PsuID,
				ConnectorID:       attempt.ConnectorID,
				CreatedAt:         attempt.CreatedAt,
				Status:            attempt.Status,
				ClientRedirectURL: attempt.ClientRedirectURL,
				Error:             attempt.Error,
			}
		}

		newCursor := bunpaginate.Cursor[attemptResponse]{
			PageSize: cursor.PageSize,
			HasMore:  cursor.HasMore,
			Previous: cursor.Previous,
			Next:     cursor.Next,
			Data:     attempts,
		}

		api.RenderCursor(w, newCursor)
	}
}
