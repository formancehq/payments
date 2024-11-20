package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type BankAccountsUpdateMetadataRequest struct {
	Metadata map[string]string `json:"metadata"`
}

func (u *BankAccountsUpdateMetadataRequest) Validate() error {
	if len(u.Metadata) == 0 {
		return errors.New("metadata must be provided")
	}

	return nil
}

func bankAccountsUpdateMetadata(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_bankAccountsUpdateMetadata")
		defer span.End()

		id, err := uuid.Parse(bankAccountID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		var req BankAccountsUpdateMetadataRequest
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromUpdateMetadataRequest(span, req.Metadata)

		err = req.Validate()
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		err = backend.BankAccountsUpdateMetadata(ctx, id, req.Metadata)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.NoContent(w)
	}
}

func populateSpanFromUpdateMetadataRequest(span trace.Span, metadata map[string]string) {
	for k, v := range metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
