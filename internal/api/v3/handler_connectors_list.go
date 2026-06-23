package v3

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
)

func connectorsList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsList")
		defer span.End()

		query, err := paginate.Extract[storage.ListConnectorsQuery](r, func() (*storage.ListConnectorsQuery, error) {
			options, err := getPagination(span, r, storage.ConnectorQuery{})
			if err != nil {
				return nil, err
			}
			return pointer.For(storage.NewListConnectorsQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectors, err := backend.ConnectorsList(ctx, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		// Resolve capabilities once for the whole page: the registry map is
		// O(plugins) and amortised across N rows, beating N separate lookups.
		caps := backend.ConnectorsCapabilities()
		api.RenderCursor(w, *paginate.MapCursor(connectors, func(c models.Connector) v3Connector {
			// Look up via the same v3 form we emit on the wire so legacy
			// uppercase storage values ("STRIPE", "DUMMY-PAY") still map to
			// their lowercase registry keys.
			provider := models.ToV3Provider(c.Provider)
			return newV3Connector(c, provider, caps[provider])
		}))
	}
}

// v3Connector is the V3 wire format for a connector row. Defined here (rather
// than relying on models.Connector.MarshalJSON) so the response stays the
// single source of truth for the V3 API and we can extend it - e.g. with
// runtime capabilities - without polluting the domain model.
type v3Connector struct {
	ID                   string              `json:"id"`
	Reference            string              `json:"reference"`
	Name                 string              `json:"name"`
	CreatedAt            time.Time           `json:"createdAt"`
	Provider             string              `json:"provider"`
	Config               json.RawMessage     `json:"config"`
	ScheduledForDeletion bool                `json:"scheduledForDeletion"`
	Capabilities         []models.Capability `json:"capabilities"`
	UpdatedAt            *time.Time          `json:"updatedAt,omitempty"`
}

func newV3Connector(c models.Connector, provider string, caps []models.Capability) v3Connector {
	if caps == nil {
		// Keep the wire contract stable: required:true in OpenAPI means we
		// always emit an array, even for connectors whose plugin is no longer
		// registered in this binary.
		caps = []models.Capability{}
	}
	return v3Connector{
		ID:                   c.ID.String(),
		Reference:            c.ID.Reference.String(),
		Name:                 c.Name,
		CreatedAt:            c.CreatedAt,
		Provider:             provider,
		Config:               c.Config,
		ScheduledForDeletion: c.ScheduledForDeletion,
		Capabilities:         caps,
		UpdatedAt:            c.UpdatedAt,
	}
}
