package v3

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

// The catalog is build-time immutable: the only way it changes is via a new
// deployment of this binary. An hour-long max-age + revalidate makes repeat
// requests body-less 304s while still letting an upgraded fleet roll the
// catalog without operator intervention. The stateless console is the primary
// consumer this protects from per-page-load fan-out.
const capabilitiesCacheControl = "public, max-age=3600, must-revalidate"

type capabilitiesCatalog struct {
	body []byte
	etag string
}

func connectorsCapabilities(backend backend.Backend) http.HandlerFunc {
	// Catalog is constant per process: compute body + ETag once per handler
	// instance and serve 304s for matching If-None-Match.
	cache := sync.OnceValues(func() (capabilitiesCatalog, error) {
		catalog := backend.ConnectorsCapabilities()
		body, err := json.Marshal(api.BaseResponse[map[string][]models.Capability]{
			Data: &catalog,
		})
		if err != nil {
			return capabilitiesCatalog{}, err
		}
		sum := sha256.Sum256(body)
		return capabilitiesCatalog{body: body, etag: `"` + hex.EncodeToString(sum[:]) + `"`}, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer().Start(r.Context(), "v3_connectorsCapabilities")
		defer span.End()

		cat, err := cache()
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}

		w.Header().Set("ETag", cat.etag)
		w.Header().Set("Cache-Control", capabilitiesCacheControl)

		if r.Header.Get("If-None-Match") == cat.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(cat.body); err != nil {
			otel.RecordError(span, err)
		}
	}
}

func connectorsCapabilitiesGet(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsCapabilitiesGet")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		id, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		caps, err := backend.ConnectorsCapabilitiesGet(ctx, id)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Ok(w, caps)
	}
}
