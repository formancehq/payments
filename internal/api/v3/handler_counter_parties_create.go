package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type BankAccountInformationRequest struct {
	// To populate if bank account was already created on Formance
	BankAccountID *string `json:"bankAccountId" validate:"omitempty,uuid"`

	// Otherwise, populate the following fields
	AccountNumber *string `json:"accountNumber" validate:"required_if=BankAccountID nil IBAN nil,omitempty,alphanum"`
	IBAN          *string `json:"iban" validate:"required_if=BankAccountID nil AccountNumber nil,omitempty,alphanum,gte=15,lte=31"`
	SwiftBicCode  *string `json:"swiftBicCode" validate:"omitempty,alphanum,gte=8,lte=11"`
}

type AddressRequest struct {
	StreetName   string `json:"streetName" validate:"omitempty"`
	StreetNumber string `json:"streetNumber" validate:"omitempty"`
	City         string `json:"city" validate:"omitempty"`
	PostalCode   string `json:"postalCode" validate:"omitempty"`
	Country      string `json:"country" validate:"omitempty,country_code"`
}

type ContactDetailsRequest struct {
	Email *string `json:"email" validate:"omitempty"`
	Phone *string `json:"phone" validate:"omitempty"`
}

type CounterPartiesCreateRequest struct {
	Name string `json:"name" validate:"required,lte=1000"`

	BankAccountInformation *BankAccountInformationRequest `json:"bankAccountInformation,omitempty" validate:"omitempty"`
	ContactDetails         *ContactDetailsRequest         `json:"contactDetails,omitempty" validate:"omitempty"`
	Address                *AddressRequest                `json:"address,omitempty" validate:"omitempty"`

	Metadata map[string]string `json:"metadata" validate:""`
}

func counterPartiesCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_counterPartiesCreate")
		defer span.End()

		var req CounterPartiesCreateRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromCounterPartiesCreateRequest(span, req)

		if _, err := validator.Validate(req); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		now := time.Now().UTC()
		counterParty := models.CounterParty{
			ID:        uuid.New(),
			Name:      req.Name,
			CreatedAt: now,
			Metadata:  req.Metadata,
		}

		var bankAccount *models.BankAccount
		switch {
		case req.BankAccountInformation != nil && req.BankAccountInformation.BankAccountID != nil:
			counterParty.BankAccountID = pointer.For(uuid.MustParse(*req.BankAccountInformation.BankAccountID))
		case req.BankAccountInformation != nil && req.BankAccountInformation.BankAccountID == nil:
			bankAccount = &models.BankAccount{
				ID:            uuid.New(),
				CreatedAt:     now,
				Name:          req.Name,
				AccountNumber: req.BankAccountInformation.AccountNumber,
				IBAN:          req.BankAccountInformation.IBAN,
				SwiftBicCode:  req.BankAccountInformation.SwiftBicCode,
			}

			if req.Address != nil {
				bankAccount.Country = &req.Address.Country
			}

			counterParty.BankAccountID = pointer.For(bankAccount.ID)
		}

		if req.Address != nil {
			counterParty.Address = &models.Address{
				StreetName:   req.Address.StreetName,
				StreetNumber: req.Address.StreetNumber,
				City:         req.Address.City,
				PostalCode:   req.Address.PostalCode,
				Country:      req.Address.Country,
			}
		}

		if req.ContactDetails != nil {
			counterParty.ContactDetails = &models.ContactDetails{
				Email: req.ContactDetails.Email,
				Phone: req.ContactDetails.Phone,
			}
		}

		err = backend.CounterPartiesCreate(ctx, counterParty, bankAccount)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, counterParty.ID.String())
	}
}

func populateSpanFromCounterPartiesCreateRequest(span trace.Span, req CounterPartiesCreateRequest) {
	span.SetAttributes(attribute.String("name", req.Name))

	// Do not record sensitive information

	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}

	if req.BankAccountInformation != nil {
		if req.BankAccountInformation.BankAccountID != nil {
			span.SetAttributes(attribute.String("bankAccountId", *req.BankAccountInformation.BankAccountID))
		}

		if req.BankAccountInformation.AccountNumber != nil {
			span.SetAttributes(attribute.String("accountNumber", *req.BankAccountInformation.AccountNumber))
		}

		if req.BankAccountInformation.IBAN != nil {
			span.SetAttributes(attribute.String("iban", *req.BankAccountInformation.IBAN))
		}

		if req.BankAccountInformation.SwiftBicCode != nil {
			span.SetAttributes(attribute.String("swiftBicCode", *req.BankAccountInformation.SwiftBicCode))
		}
	}

	if req.Address != nil {
		span.SetAttributes(attribute.String("streetName", req.Address.StreetName))
		span.SetAttributes(attribute.String("streetNumber", req.Address.StreetNumber))
		span.SetAttributes(attribute.String("city", req.Address.City))
		span.SetAttributes(attribute.String("postalCode", req.Address.PostalCode))
		span.SetAttributes(attribute.String("country", req.Address.Country))
	}

	if req.ContactDetails != nil {
		if req.ContactDetails.Email != nil {
			span.SetAttributes(attribute.String("email", *req.ContactDetails.Email))
		}

		if req.ContactDetails.Phone != nil {
			span.SetAttributes(attribute.String("phone", *req.ContactDetails.Phone))
		}
	}
}
