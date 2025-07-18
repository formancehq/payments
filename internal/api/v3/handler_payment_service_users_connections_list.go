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

func paymentServiceUsersConnectionsListAll(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentServiceUsersConnectionsList")
		defer span.End()

		span.SetAttributes(attribute.String("paymentServiceUserID", paymentServiceUserID(r)))
		id, err := uuid.Parse(paymentServiceUserID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		query, err := bunpaginate.Extract[storage.ListPsuBankBridgeConnectionsQuery](r, func() (*storage.ListPsuBankBridgeConnectionsQuery, error) {
			options, err := getPagination(span, r, storage.PsuBankBridgeConnectionsQuery{})
			if err != nil {
				return nil, err
			}

			return pointer.For(storage.NewListPsuBankBridgeConnectionsQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		cursor, err := backend.PaymentServiceUsersConnectionsList(ctx, id, nil, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.RenderCursor(w, *cursor)
	}
}

func paymentServiceUsersConnectionsListFromConnectorID(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentServiceUsersConnectionsListFromConnectorID")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		span.SetAttributes(attribute.String("paymentServiceUserID", paymentServiceUserID(r)))
		id, err := uuid.Parse(paymentServiceUserID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		query, err := bunpaginate.Extract[storage.ListPsuBankBridgeConnectionsQuery](r, func() (*storage.ListPsuBankBridgeConnectionsQuery, error) {
			options, err := getPagination(span, r, storage.PsuBankBridgeConnectionsQuery{})
			if err != nil {
				return nil, err
			}

			return pointer.For(storage.NewListPsuBankBridgeConnectionsQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		cursor, err := backend.PaymentServiceUsersConnectionsList(ctx, id, &connectorID, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.RenderCursor(w, *cursor)
	}
}
