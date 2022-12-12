package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/formancehq/payments/internal/app/models"
	"github.com/formancehq/payments/internal/app/storage"

	"github.com/formancehq/payments/internal/app/payments"

	"github.com/formancehq/go-libs/sharedapi"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

const (
	maxPerPage = 100
)

type listPaymentsRepository interface {
	ListPayments(ctx context.Context, sort storage.Sorter, pagination storage.Paginator) ([]*models.Payment, error)
}

func listPaymentsHandler(repo listPaymentsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var sorter storage.Sorter

		if sortParams := r.URL.Query()["sort"]; sortParams != nil {
			for _, s := range sortParams {
				parts := strings.SplitN(s, ":", 2)

				var order storage.SortOrder

				if len(parts) > 1 {
					switch parts[1] {
					case "asc", "ASC":
						order = storage.SortOrderAsc
					case "dsc", "desc", "DSC", "DESC":
						order = storage.SortOrderDesc
					default:
						handleValidationError(w, r, errors.New("sort order not well specified, got "+parts[1]))

						return
					}
				}

				column := parts[0]

				sorter.Add(column, order)
			}
		}

		skip, err := unsignedIntegerWithDefault(r, "skip", 0)
		if err != nil {
			handleValidationError(w, r, err)

			return
		}

		limit, err := unsignedIntegerWithDefault(r, "limit", maxPerPage)
		if err != nil {
			handleValidationError(w, r, err)

			return
		}

		if limit > maxPerPage {
			limit = maxPerPage
		}

		ret, err := repo.ListPayments(r.Context(), sorter, storage.Paginate(skip, limit))
		if err != nil {
			handleServerError(w, r, err)

			return
		}

		data := make([]*payments.Payment, len(ret))

		for i := range ret {
			data[i] = &payments.Payment{
				Identifier: payments.Identifier{
					Referenced: payments.Referenced{
						Reference: ret[i].Reference,
						Type:      ret[i].Type.String(),
					},
					Provider: ret[i].Connector.Provider.String(),
				},
				Data: payments.Data{
					Status:        payments.Status(ret[i].Status),
					InitialAmount: ret[i].Amount,
					Scheme:        payments.Scheme(ret[i].Scheme),
					Asset:         ret[i].Asset.String(),
					CreatedAt:     ret[i].CreatedAt,
					Raw:           ret[i].RawData,
				},
			}

			for adjustmentIdx := range ret[i].Adjustments {
				data[i].Adjustments = append(data[i].Adjustments,
					payments.Adjustment{
						Status:   payments.Status(ret[i].Adjustments[adjustmentIdx].Status),
						Amount:   ret[i].Adjustments[adjustmentIdx].Amount,
						Date:     ret[i].Adjustments[adjustmentIdx].CreatedAt,
						Raw:      ret[i].Adjustments[adjustmentIdx].RawData,
						Absolute: ret[i].Adjustments[adjustmentIdx].Absolute,
					})
			}
		}

		err = json.NewEncoder(w).Encode(sharedapi.BaseResponse[[]*payments.Payment]{
			Data: &data,
		})
		if err != nil {
			handleServerError(w, r, err)

			return
		}
	}
}

type readPaymentRepository interface {
	GetPayment(ctx context.Context, reference string) (*models.Payment, error)
}

func readPaymentHandler(repo readPaymentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		paymentID := mux.Vars(r)["paymentID"]

		identifier, err := payments.IdentifierFromString(paymentID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		payment, err := repo.GetPayment(r.Context(), identifier.Reference)
		if err != nil {
			handleServerError(w, r, err)

			return
		}

		data := payments.Payment{
			Identifier: payments.Identifier{
				Referenced: payments.Referenced{
					Reference: payment.Reference,
					Type:      payment.Type.String(),
				},
				Provider: payment.Connector.Provider.String(),
			},
			Data: payments.Data{
				Status:        payments.Status(payment.Status),
				InitialAmount: payment.Amount,
				Scheme:        payments.Scheme(payment.Scheme),
				Asset:         payment.Asset.String(),
				CreatedAt:     payment.CreatedAt,
				Raw:           payment.RawData,
			},
		}

		for i := range payment.Adjustments {
			data.Adjustments = append(data.Adjustments,
				payments.Adjustment{
					Status:   payments.Status(payment.Adjustments[i].Status),
					Amount:   payment.Adjustments[i].Amount,
					Date:     payment.Adjustments[i].CreatedAt,
					Raw:      payment.Adjustments[i].RawData,
					Absolute: payment.Adjustments[i].Absolute,
				})
		}

		err = json.NewEncoder(w).Encode(sharedapi.BaseResponse[payments.Payment]{
			Data: &data,
		})
		if err != nil {
			handleServerError(w, r, err)

			return
		}
	}
}

func integer(r *http.Request, key string) (int64, bool, error) {
	if value := r.URL.Query().Get(key); value != "" {
		ret, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, false, err
		}

		return ret, true, nil
	}

	return 0, false, nil
}

func unsignedIntegerWithDefault(r *http.Request, key string, def uint64) (uint64, error) {
	value, ok, err := integer(r, key)
	if err != nil {
		return 0, err
	}

	if !ok || value < 0 {
		return def, nil
	}

	return uint64(value), nil
}
