package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreateAccountRequest struct {
	Reference    string            `json:"reference" validate:"required,gte=3,lte=1000"`
	ConnectorID  string            `json:"connectorID" validate:"required,connectorID"`
	CreatedAt    time.Time         `json:"createdAt" validate:"required,lte=now"`
	DefaultAsset string            `json:"defaultAsset" validate:"omitempty,asset"`
	AccountName  string            `json:"accountName" validate:"required,lte=1000"`
	Type         string            `json:"type" validate:"required,accountType"`
	Metadata     map[string]string `json:"metadata" validate:""`
}

func accountsCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
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

		if _, err := validator.Validate(req); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectorID := models.MustConnectorIDFromString(req.ConnectorID)
		raw, err := json.Marshal(req)
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
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

		_, err = backend.AccountsCreate(ctx, account)
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
			Provider:    toV2Provider(account.ConnectorID.Provider),
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
			common.InternalServerError(w, r, err)
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
