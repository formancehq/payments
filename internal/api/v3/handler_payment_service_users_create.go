package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ContactDetailsRequest struct {
	Email       *string `json:"email,omitempty" validate:"omitempty,email"`
	PhoneNumber *string `json:"phoneNumber,omitempty" validate:"omitempty,phoneNumber"`
}

type AddressRequest struct {
	StreetName   *string `json:"streetName,omitempty"`
	StreetNumber *string `json:"streetNumber,omitempty" validate:"omitempty,alphanum"`
	City         *string `json:"city,omitempty"`
	Region       *string `json:"region,omitempty"`
	PostalCode   *string `json:"postalCode,omitempty"`
	Country      *string `json:"country,omitempty" validate:"omitempty,country_code"`
}

type PaymentServiceUsersCreateRequest struct {
	Name string `json:"name" validate:"required,lte=1000"`

	ContactDetails *ContactDetailsRequest `json:"contactDetails,omitempty"`
	Address        *AddressRequest        `json:"address,omitempty"`
	BankAccountIDs []string               `json:"bankAccountIDs,omitempty" validate:"omitempty,dive,uuid"`
	Metadata       map[string]string      `json:"metadata,omitempty" validate:""`
}

func paymentServiceUsersCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentServiceUsersCreate")
		defer span.End()

		var req PaymentServiceUsersCreateRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromPaymentServiceUserCreateRequest(span, req)

		if _, err := validator.Validate(req); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		bankAccountIDs := make([]uuid.UUID, len(req.BankAccountIDs))
		for i, id := range req.BankAccountIDs {
			bankAccountIDs[i], err = uuid.Parse(id)
			if err != nil {
				otel.RecordError(span, err)
				api.BadRequest(w, ErrValidation, err)
				return
			}
		}

		paymentServiceUser := models.PaymentServiceUser{
			ID:        uuid.New(),
			Name:      req.Name,
			CreatedAt: time.Now().UTC(),
			ContactDetails: func() *models.ContactDetails {
				if req.ContactDetails == nil {
					return nil
				}

				return &models.ContactDetails{
					Email:       req.ContactDetails.Email,
					PhoneNumber: req.ContactDetails.PhoneNumber,
				}
			}(),
			Address: func() *models.Address {
				if req.Address == nil {
					return nil
				}

				return &models.Address{
					StreetName:   req.Address.StreetName,
					StreetNumber: req.Address.StreetNumber,
					City:         req.Address.City,
					Region:       req.Address.Region,
					PostalCode:   req.Address.PostalCode,
					Country:      req.Address.Country,
				}
			}(),
			BankAccountIDs: bankAccountIDs,
			Metadata:       req.Metadata,
		}

		err = backend.PaymentServiceUsersCreate(ctx, paymentServiceUser)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, paymentServiceUser.ID.String())
	}
}

func populateSpanFromPaymentServiceUserCreateRequest(span trace.Span, req PaymentServiceUsersCreateRequest) {
	span.SetAttributes(attribute.String("name", req.Name))

	// Do not record other information as they are sensitive information

	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
