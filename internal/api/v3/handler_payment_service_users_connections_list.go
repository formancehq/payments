package v3

import (
	"net/http"
	"time"

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

// Custom response type for the connection because we don't want to expose the
// access token. We still need them in the models object because of temporal,
// that's why we need to create a custom response type.
type bankBridgeConnection struct {
	// ID of the connection, given by the banking bridge
	ConnectionID string `json:"connectionID"`
	// Connector ID
	ConnectorID string `json:"connectorID"`
	// Creation date of the connection
	CreatedAt time.Time `json:"createdAt"`
	// Date of the last update of the connection's data
	DataUpdatedAt time.Time `json:"dataUpdatedAt"`
	// Status of the connection
	Status models.ConnectionStatus `json:"status"`

	// Optional
	// Error message in case of failure
	Error *string `json:"error"`
	// Additional information about the connection depending on the connector
	Metadata map[string]string `json:"metadata"`
}

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

		connections := make([]bankBridgeConnection, 0, len(cursor.Data))
		for _, connection := range cursor.Data {
			connections = append(connections, bankBridgeConnection{
				ConnectionID:  connection.ConnectionID,
				ConnectorID:   connection.ConnectorID.String(),
				CreatedAt:     connection.CreatedAt,
				DataUpdatedAt: connection.DataUpdatedAt,
				Status:        connection.Status,
				Error:         connection.Error,
				Metadata:      connection.Metadata,
			})
		}

		api.RenderCursor(w, bunpaginate.Cursor[bankBridgeConnection]{
			PageSize: cursor.PageSize,
			HasMore:  cursor.HasMore,
			Previous: cursor.Previous,
			Next:     cursor.Next,
			Data:     connections,
		})
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

		connections := make([]bankBridgeConnection, 0, len(cursor.Data))
		for _, connection := range cursor.Data {
			connections = append(connections, bankBridgeConnection{
				ConnectionID:  connection.ConnectionID,
				ConnectorID:   connection.ConnectorID.String(),
				CreatedAt:     connection.CreatedAt,
				DataUpdatedAt: connection.DataUpdatedAt,
				Status:        connection.Status,
				Error:         connection.Error,
				Metadata:      connection.Metadata,
			})
		}

		api.RenderCursor(w, bunpaginate.Cursor[bankBridgeConnection]{
			PageSize: cursor.PageSize,
			HasMore:  cursor.HasMore,
			Previous: cursor.Previous,
			Next:     cursor.Next,
			Data:     connections,
		})
	}
}
