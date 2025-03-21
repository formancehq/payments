package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
)

func bankAccountsList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_bankAccountsList")
		defer span.End()

		query, err := bunpaginate.Extract[storage.ListBankAccountsQuery](r, func() (*storage.ListBankAccountsQuery, error) {
			options, err := getPagination(span, r, storage.BankAccountQuery{})
			if err != nil {
				return nil, err
			}
			return pointer.For(storage.NewListBankAccountsQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		cursor, err := backend.BankAccountsList(ctx, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		for i := range cursor.Data {
			if err := cursor.Data[i].Obfuscate(); err != nil {
				otel.RecordError(span, err)
				common.InternalServerError(w, r, err)
				return
			}
		}

		api.RenderCursor(w, *cursor)
	}
}
