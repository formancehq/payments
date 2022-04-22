package payment

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs/sharedapi"
	"github.com/numary/go-libs/sharedlogging"
	"net/http"
	"strconv"
	"strings"
)

func Bool(r *http.Request, key string) (bool, bool) {
	vv := r.URL.Query().Get(key)
	if vv == "" {
		return false, false
	}
	vv = strings.ToUpper(vv)
	return vv == "YES" || vv == "TRUE" || vv == "1", true
}

func BoolWithDefault(r *http.Request, key string, def bool) bool {
	v, ok := Bool(r, key)
	if !ok {
		return def
	}
	return v
}

func Integer(r *http.Request, key string) (int64, bool, error) {
	if value := r.URL.Query().Get(key); value != "" {
		ret, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, false, err
		}
		return ret, true, nil
	}
	return 0, false, nil
}

func IntegerWithDefault(r *http.Request, key string, def int64) (int64, error) {
	value, ok, err := Integer(r, key)
	if err != nil {
		return 0, err
	}
	if ok {
		return value, nil
	}
	return def, nil
}

const (
	maxPerPage = 100
)

func handleServerError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	sharedlogging.GetLogger(r.Context()).Error(err)
	err = json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
		ErrorCode: "INTERNAL",
	})
	if err != nil {
		panic(err)
	}
}

func handleClientError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusBadRequest)
	err = json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
		ErrorCode:    "INTERNAL",
		ErrorMessage: err.Error(),
	})
	if err != nil {
		panic(err)
	}
}

func ListPaymentsHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var err error
		parameters := ListQueryParameters{}
		parameters.Skip, err = IntegerWithDefault(r, "skip", 0)
		if err != nil {
			handleClientError(w, r, err)
			return
		}
		parameters.Limit, err = IntegerWithDefault(r, "limit", maxPerPage)
		if err != nil {
			handleClientError(w, r, err)
			return
		}
		if parameters.Limit > maxPerPage {
			parameters.Limit = maxPerPage
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

		cursor, err := s.ListPayments(r.Context(), parameters)
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
		err = json.NewEncoder(w).Encode(sharedapi.BaseResponse{
			Data: results,
		})
		if err != nil {
			handleServerError(w, r, err)
			return
		}
	}
}

func SavePaymentHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		payment := Payment{}
		err := json.NewDecoder(r.Body).Decode(&payment)
		if err != nil {
			handleClientError(w, r, err)
			return
		}

		err = s.SavePayment(r.Context(), payment)
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func NewMux(service Service) *mux.Router {
	router := mux.NewRouter()
	router.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			h.ServeHTTP(w, r)
		})
	})
	router.Path("/").Methods(http.MethodGet).Handler(ListPaymentsHandler(service))
	router.Path("/").Methods(http.MethodPut).Handler(SavePaymentHandler(service))

	return router
}
