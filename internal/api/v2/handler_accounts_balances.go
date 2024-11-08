package v2

import (
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type balancesResponse struct {
	AccountID     string    `json:"accountId"`
	CreatedAt     time.Time `json:"createdAt"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	Asset         string    `json:"asset"`
	Balance       *big.Int  `json:"balance"`
}

func accountsBalances(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_accountsBalances")
		defer span.End()

		balanceQuery, err := populateBalanceQueryFromRequest(span, r)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		query, err := bunpaginate.Extract[storage.ListBalancesQuery](r, func() (*storage.ListBalancesQuery, error) {
			options, err := getPagination(span, r, balanceQuery)
			if err != nil {
				return nil, err
			}
			return pointer.For(storage.NewListBalancesQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		cursor, err := backend.BalancesList(ctx, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		ret := cursor.Data
		data := make([]*balancesResponse, len(ret))

		for i := range ret {
			data[i] = &balancesResponse{
				AccountID:     ret[i].AccountID.String(),
				CreatedAt:     ret[i].CreatedAt,
				Asset:         ret[i].Asset,
				Balance:       ret[i].Balance,
				LastUpdatedAt: ret[i].LastUpdatedAt,
			}
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[*balancesResponse]{
			Cursor: &bunpaginate.Cursor[*balancesResponse]{
				PageSize: cursor.PageSize,
				HasMore:  cursor.HasMore,
				Previous: cursor.Previous,
				Next:     cursor.Next,
				Data:     data,
			},
		})
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
	}
}

func populateBalanceQueryFromRequest(span trace.Span, r *http.Request) (storage.BalanceQuery, error) {
	var balanceQuery storage.BalanceQuery

	balanceQuery = balanceQuery.WithAsset(r.URL.Query().Get("asset"))
	span.SetAttributes(attribute.String("asset", balanceQuery.Asset))

	span.SetAttributes(attribute.String("accountID", accountID(r)))
	accountID, err := models.AccountIDFromString(accountID(r))
	if err != nil {
		return balanceQuery, err
	}
	balanceQuery = balanceQuery.WithAccountID(&accountID)

	var startTimeParsed, endTimeParsed time.Time

	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from != "" {
		startTimeParsed, err = time.Parse(time.RFC3339Nano, from)
		if err != nil {
			return balanceQuery, err
		}
	}
	if to != "" {
		endTimeParsed, err = time.Parse(time.RFC3339Nano, to)
		if err != nil {
			return balanceQuery, err
		}
	}

	switch {
	case startTimeParsed.IsZero() && endTimeParsed.IsZero():
		balanceQuery = balanceQuery.
			WithTo(time.Now())
	case !startTimeParsed.IsZero() && endTimeParsed.IsZero():
		balanceQuery = balanceQuery.
			WithFrom(startTimeParsed).
			WithTo(time.Now())
	case startTimeParsed.IsZero() && !endTimeParsed.IsZero():
		balanceQuery = balanceQuery.
			WithTo(endTimeParsed)
	default:
		balanceQuery = balanceQuery.
			WithFrom(startTimeParsed).
			WithTo(endTimeParsed)
	}

	span.SetAttributes(attribute.String("from", balanceQuery.From.Format(time.RFC3339Nano)))
	span.SetAttributes(attribute.String("to", balanceQuery.To.Format(time.RFC3339Nano)))

	return balanceQuery, nil
}
