package payment

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
)

const (
	maxPerPage = 100
)

func handleServerError(w http.ResponseWriter, r *http.Request, err error) {
	panic(err)
}

func handleClientError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func ListPaymentsHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var err error
		parameters := ListQueryParameters{}
		if skip := r.URL.Query().Get("skip"); skip != "" {
			parameters.Skip, err = strconv.ParseInt(skip, 10, 64)
			if err != nil {
				handleClientError(w, r, err)
				return
			}
		}
		if limit := r.URL.Query().Get("limit"); limit != "" {
			parameters.Limit, err = strconv.ParseInt(limit, 10, 64)
			if err != nil {
				handleClientError(w, r, err)
				return
			}
			if parameters.Limit > maxPerPage {
				parameters.Limit = maxPerPage
			}
		}
		if sorts := r.URL.Query()["sort"]; sorts != nil {
			for _, s := range sorts {
				parts := strings.SplitN(s, ":", 2)
				desc := false
				if len(parts) > 1 {
					switch parts[1] {
					case "asc", "ASC":
					case "dsc", "desc", "DSC", "DESC":
						desc = true
					default:
						handleClientError(w, r, errors.New("sort order not well specified, got "+parts[1]))
						return
					}
				}
				key := parts[0]
				if key == "id" {
					key = "_id"
				}
				parameters.Sort = append(parameters.Sort, Sort{
					Key:  key,
					Desc: desc,
				})
			}
		}

		cursor, err := s.ListPayments(r.Context(), mux.Vars(r)["organizationId"], parameters)
		if err != nil {
			handleServerError(w, r, err)
			return
		}
		defer cursor.Close(r.Context())

		results := make([]*Payment, 0)
		err = cursor.All(r.Context(), &results)
		if err != nil {
			handleServerError(w, r, err)
			return
		}
		err = json.NewEncoder(w).Encode(results)
		if err != nil {
			handleServerError(w, r, err)
			return
		}
	}
}

func CreatePaymentHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		data := Data{}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			handleClientError(w, r, err)
			return
		}

		p, err := s.CreatePayment(r.Context(), mux.Vars(r)["organizationId"], data)
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		w.Header().Set("Location", "./"+p.ID)
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(p)
		if err != nil {
			handleServerError(w, r, err)
			return
		}
	}
}

func UpdatePaymentHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := Data{}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			handleClientError(w, r, err)
			return
		}

		modified, err := s.UpdatePayment(r.Context(), mux.Vars(r)["organizationId"], mux.Vars(r)["paymentId"], data)
		if err != nil {
			handleServerError(w, r, err)
			return
		}
		if !modified {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func NewMux(service Service) *mux.Router {
	router := mux.NewRouter()
	organizationRouter := router.PathPrefix("/organizations/{organizationId}").Subrouter()
	organizationRouter.Path("/payments").Methods(http.MethodGet).Handler(ListPaymentsHandler(service))
	organizationRouter.Path("/payments").Methods(http.MethodPost).Handler(CreatePaymentHandler(service))
	paymentsRouter := organizationRouter.PathPrefix("/payments").Subrouter()
	paymentsRouter.Path("/{paymentId}").Methods(http.MethodPut).Handler(UpdatePaymentHandler(service))

	return router
}
