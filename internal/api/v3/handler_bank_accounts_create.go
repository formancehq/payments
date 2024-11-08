package v3

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type bankAccountsCreateRequest struct {
	Name string `json:"name"`

	AccountNumber *string `json:"accountNumber"`
	IBAN          *string `json:"iban"`
	SwiftBicCode  *string `json:"swiftBicCode"`
	Country       *string `json:"country"`

	Metadata map[string]string `json:"metadata"`
}

func (r *bankAccountsCreateRequest) Validate() error {
	if r.AccountNumber == nil && r.IBAN == nil {
		return errors.New("either accountNumber or iban must be provided")
	}

	if r.Name == "" {
		return errors.New("name must be provided")
	}

	return nil
}

func bankAccountsCreate(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_bankAccountsCreate")
		defer span.End()

		var req bankAccountsCreateRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromBankAccountCreateRequest(span, req)

		if err := req.Validate(); err != nil {
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

func populateSpanFromBankAccountCreateRequest(span trace.Span, req bankAccountsCreateRequest) {
	span.SetAttributes(attribute.String("name", req.Name))

	// Do not record sensitive information

	if req.Country != nil {
		span.SetAttributes(attribute.String("country", *req.Country))
	}

	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
