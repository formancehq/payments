package v3

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
)

type createAccountRequest struct {
	Reference    string            `json:"reference"`
	ConnectorID  string            `json:"connectorID"`
	CreatedAt    time.Time         `json:"createdAt"`
	DefaultAsset string            `json:"defaultAsset"`
	AccountName  string            `json:"accountName"`
	Type         string            `json:"type"`
	Metadata     map[string]string `json:"metadata"`
}

func (r *createAccountRequest) validate() error {
	if r.Reference == "" {
		return errors.New("reference is required")
	}

	if r.ConnectorID == "" {
		return errors.New("connectorID is required")
	}

	if r.CreatedAt.IsZero() || r.CreatedAt.After(time.Now()) {
		return errors.New("createdAt is empty or in the future")
	}

	if r.AccountName == "" {
		return errors.New("accountName is required")
	}

	if r.Type == "" {
		return errors.New("type is required")
	}

	_, err := models.ConnectorIDFromString(r.ConnectorID)
	if err != nil {
		return errors.New("connectorID is invalid")
	}

	switch r.Type {
	case string(models.ACCOUNT_TYPE_EXTERNAL):
	case string(models.ACCOUNT_TYPE_INTERNAL):
	default:
		return errors.New("type is invalid")
	}

	return nil
}

func accountsCreate(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_accountsCreate")
		defer span.End()

		var req createAccountRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		if err := req.validate(); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectorID := models.MustConnectorIDFromString(req.ConnectorID)
		raw, err := json.Marshal(req)
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}

		account := models.Account{
			ID: models.AccountID{
				Reference:   req.Reference,
				ConnectorID: connectorID,
			},
			ConnectorID:  connectorID,
			Reference:    req.Reference,
			CreatedAt:    req.CreatedAt,
			Type:         models.AccountType(req.Type),
			Name:         &req.AccountName,
			DefaultAsset: &req.DefaultAsset,
			Metadata:     req.Metadata,
			Raw:          raw,
		}

		err = backend.AccountsCreate(ctx, account)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, account)
	}
}