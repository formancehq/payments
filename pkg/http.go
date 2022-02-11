package payment

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

func handleServerError(w http.ResponseWriter, r *http.Request, err error) {
	panic(err)
}

func handleClientError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func ListPaymentsHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payments, err := s.ListPayments(r.Context(), mux.Vars(r)["organizationId"])
		if err != nil {
			handleServerError(w, r, err)
			return
		}
		err = json.NewEncoder(w).Encode(payments)
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

		err = s.UpdatePayment(r.Context(), mux.Vars(r)["organizationId"], mux.Vars(r)["paymentId"], data)
		if err != nil {
			handleServerError(w, r, err)
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
