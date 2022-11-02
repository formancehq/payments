package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/numary/payments/internal/pkg/payments"

	"github.com/gorilla/mux"
	"github.com/numary/go-libs/sharedapi"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	maxPerPage = 100
)

func handleServerError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	sharedlogging.GetLogger(r.Context()).Error(err)
	// TODO: Opentracing
	err = json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
		ErrorCode:    "INTERNAL",
		ErrorMessage: err.Error(),
	})
	if err != nil {
		panic(err)
	}
}

func handleValidationError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusBadRequest)
	sharedlogging.GetLogger(r.Context()).Error(err)
	// TODO: Opentracing
	err = json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
		ErrorCode:    "VALIDATION",
		ErrorMessage: err.Error(),
	})
	if err != nil {
		panic(err)
	}
}

func listPaymentsHandler(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pipeline := make([]map[string]any, 0)

		if sortParams := r.URL.Query()["sort"]; sortParams != nil {
			sort := bson.M{}

			for _, s := range sortParams {
				parts := strings.SplitN(s, ":", 2)
				desc := false

				if len(parts) > 1 {
					switch parts[1] {
					case "asc", "ASC":
					case "dsc", "desc", "DSC", "DESC":
						desc = true
					default:
						handleValidationError(w, r, errors.New("sort order not well specified, got "+parts[1]))

						return
					}
				}

				key := parts[0]
				if key == "id" {
					key = "_id"
				}

				sort[key] = func() int {
					if desc {
						return -1
					}

					return 1
				}()
			}

			pipeline = append(pipeline, map[string]any{"$sort": sort})
		}

		skip, err := integerWithDefault(r, "skip", 0)
		if err != nil {
			handleValidationError(w, r, err)

			return
		}

		if skip != 0 {
			pipeline = append(pipeline, map[string]any{
				"$skip": skip,
			})
		}

		limit, err := integerWithDefault(r, "limit", maxPerPage)
		if err != nil {
			handleValidationError(w, r, err)

			return
		}

		if limit > maxPerPage {
			limit = maxPerPage
		}

		if limit != 0 {
			pipeline = append(pipeline, map[string]any{
				"$limit": limit,
			})
		}

		cursor, err := db.Collection(payments.Collection).Aggregate(r.Context(), pipeline)
		if err != nil {
			handleServerError(w, r, err)

			return
		}

		defer cursor.Close(r.Context())

		ret := make([]payments.Payment, 0)

		err = cursor.All(r.Context(), &ret)
		if err != nil {
			handleServerError(w, r, err)

			return
		}

		err = json.NewEncoder(w).Encode(sharedapi.BaseResponse[[]payments.Payment]{
			Data: &ret,
		})
		if err != nil {
			handleServerError(w, r, err)

			return
		}
	}
}

func readPaymentHandler(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paymentID := mux.Vars(r)["paymentID"]

		identifier, err := payments.IdentifierFromString(paymentID)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		ret := db.Collection(payments.Collection).FindOne(r.Context(), identifier)
		if ret.Err() != nil {
			if errors.Is(ret.Err(), mongo.ErrNoDocuments) {
				w.WriteHeader(http.StatusNotFound)

				return
			}

			handleServerError(w, r, ret.Err())

			return
		}

		payment := &payments.Payment{}

		err = ret.Decode(payment)
		if err != nil {
			handleServerError(w, r, err)

			return
		}

		err = json.NewEncoder(w).Encode(sharedapi.BaseResponse[payments.Payment]{
			Data: payment,
		})
		if err != nil {
			handleServerError(w, r, err)

			return
		}
	}
}

func paymentsRouter(db *mongo.Database) *mux.Router {
	router := mux.NewRouter()
	router.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			h.ServeHTTP(w, r)
		})
	})
	router.Path("/payments").Methods(http.MethodGet).Handler(listPaymentsHandler(db))
	router.Path("/payments/{paymentID}").Methods(http.MethodGet).Handler(readPaymentHandler(db))

	return router
}