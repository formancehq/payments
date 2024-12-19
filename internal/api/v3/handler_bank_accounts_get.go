package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

func bankAccountsGet(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_bankAccountsGet")
		defer span.End()

		span.SetAttributes(attribute.String("bankAccountID", bankAccountID(r)))
		id, err := uuid.Parse(bankAccountID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		bankAccount, err := backend.BankAccountsGet(ctx, id)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		if err := bankAccount.Offuscate(); err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}

		api.Ok(w, bankAccount)
	}
}
