package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type BankAccountsCreateRequest struct {
	Name string `json:"name" validate:"required,lte=1000"`

	AccountNumber *string `json:"accountNumber" validate:"required_if=IBAN nil"`
	IBAN          *string `json:"iban" validate:"required_if=AccountNumber nil"`
	SwiftBicCode  *string `json:"swiftBicCode" validate:""`
	Country       *string `json:"country" validate:"omitempty,country_code"`

	Metadata map[string]string `json:"metadata" validate:""`
}

func bankAccountsCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_bankAccountsCreate")
		defer span.End()

		var req BankAccountsCreateRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromBankAccountCreateRequest(span, req)

		if _, err := validator.Validate(req); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		bankAccount := &models.BankAccount{
			ID:            uuid.New(),
			CreatedAt:     time.Now().UTC(),
			Name:          req.Name,
			AccountNumber: req.AccountNumber,
			IBAN:          req.IBAN,
			SwiftBicCode:  req.SwiftBicCode,
			Country:       req.Country,
			Metadata:      req.Metadata,
		}

		err = backend.BankAccountsCreate(ctx, *bankAccount)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, bankAccount.ID.String())
	}
}

func populateSpanFromBankAccountCreateRequest(span trace.Span, req BankAccountsCreateRequest) {
	span.SetAttributes(attribute.String("name", req.Name))

	// Do not record sensitive information

	if req.Country != nil {
		span.SetAttributes(attribute.String("country", *req.Country))
	}

	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
