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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreateAccountRequest struct {
	Reference    string            `json:"reference"`
	ConnectorID  string            `json:"connectorID"`
	CreatedAt    time.Time         `json:"createdAt"`
	DefaultAsset string            `json:"defaultAsset"`
	AccountName  string            `json:"accountName"`
	Type         string            `json:"type"`
	Metadata     map[string]string `json:"metadata"`
}

func (r *CreateAccountRequest) validate() error {
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
		ctx, span := otel.Tracer().Start(r.Context(), "v2_accountsCreate")
		defer span.End()

		var req CreateAccountRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromAccountCreateRequest(span, req)

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

		// Compatibility with old API
		res := accountResponse{
			ID:          account.ID.String(),
			Reference:   account.Reference,
			CreatedAt:   account.CreatedAt,
			ConnectorID: account.ConnectorID.String(),
			Provider:    account.ConnectorID.Provider,
			Type:        string(account.Type),
			Metadata:    account.Metadata,
			Raw:         account.Raw,
		}

		if account.DefaultAsset != nil {
			res.DefaultCurrency = *account.DefaultAsset
			res.DefaultAsset = *account.DefaultAsset
		}

		if account.Name != nil {
			res.AccountName = *account.Name
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[accountResponse]{
			Data: &res,
		})
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
	}
}

func populateSpanFromAccountCreateRequest(span trace.Span, req CreateAccountRequest) {
	span.SetAttributes(attribute.String("reference", req.Reference))
	span.SetAttributes(attribute.String("connectorID", req.ConnectorID))
	span.SetAttributes(attribute.String("createdAt", req.CreatedAt.String()))
	span.SetAttributes(attribute.String("defaultAsset", req.DefaultAsset))
	span.SetAttributes(attribute.String("accountName", req.AccountName))
	span.SetAttributes(attribute.String("type", req.Type))
	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
