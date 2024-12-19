package v2

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

// NOTE: in order to maintain previous version compatibility, we need to keep the
// same response structure as the previous version of the API
type bankAccountRelatedAccountsResponse struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	AccountID   string    `json:"accountID"`
	ConnectorID string    `json:"connectorID"`
	Provider    string    `json:"provider"`
}

type BankAccountResponse struct {
	ID              string                                `json:"id"`
	Name            string                                `json:"name"`
	CreatedAt       time.Time                             `json:"createdAt"`
	Country         string                                `json:"country"`
	Iban            string                                `json:"iban,omitempty"`
	AccountNumber   string                                `json:"accountNumber,omitempty"`
	SwiftBicCode    string                                `json:"swiftBicCode,omitempty"`
	Metadata        map[string]string                     `json:"metadata,omitempty"`
	RelatedAccounts []*bankAccountRelatedAccountsResponse `json:"relatedAccounts,omitempty"`
}

type BankAccountsCreateRequest struct {
	Name string `json:"name"`

	AccountNumber *string `json:"accountNumber"`
	IBAN          *string `json:"iban"`
	SwiftBicCode  *string `json:"swiftBicCode"`
	Country       *string `json:"country"`
	ConnectorID   *string `json:"connectorID"`

	Metadata map[string]string `json:"metadata"`
}

func (r *BankAccountsCreateRequest) Validate() error {
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
		ctx, span := otel.Tracer().Start(r.Context(), "v2_bankAccountsCreate")
		defer span.End()

		var req BankAccountsCreateRequest
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

		var connectorID *models.ConnectorID
		if req.ConnectorID != nil {
			c, err := models.ConnectorIDFromString(*req.ConnectorID)
			if err != nil {
				otel.RecordError(span, err)
				api.BadRequest(w, ErrValidation, err)
				return
			}
			connectorID = &c
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

		if connectorID != nil {
			_, err = backend.BankAccountsForwardToConnector(ctx, bankAccount.ID, *connectorID, true)
			if err != nil {
				otel.RecordError(span, err)
				handleServiceErrors(w, r, err)
				return
			}

			bankAccount, err = backend.BankAccountsGet(ctx, bankAccount.ID)
			if err != nil {
				otel.RecordError(span, err)
				handleServiceErrors(w, r, err)
				return
			}
		}

		if err := bankAccount.Offuscate(); err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}

		data := &BankAccountResponse{
			ID:        bankAccount.ID.String(),
			Name:      bankAccount.Name,
			CreatedAt: bankAccount.CreatedAt,
			Metadata:  bankAccount.Metadata,
		}

		if bankAccount.IBAN != nil {
			data.Iban = *bankAccount.IBAN
		}

		if bankAccount.AccountNumber != nil {
			data.AccountNumber = *bankAccount.AccountNumber
		}

		if bankAccount.SwiftBicCode != nil {
			data.SwiftBicCode = *bankAccount.SwiftBicCode
		}

		if bankAccount.Country != nil {
			data.Country = *bankAccount.Country
		}

		for _, relatedAccount := range bankAccount.RelatedAccounts {
			data.RelatedAccounts = append(data.RelatedAccounts, &bankAccountRelatedAccountsResponse{
				ID:          "",
				CreatedAt:   relatedAccount.CreatedAt,
				AccountID:   relatedAccount.AccountID.String(),
				ConnectorID: relatedAccount.AccountID.ConnectorID.String(),
				Provider:    relatedAccount.AccountID.ConnectorID.Provider,
			})
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[BankAccountResponse]{
			Data: data,
		})
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
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
