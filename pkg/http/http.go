package http

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs/sharedapi"
	"github.com/numary/go-libs/sharedauth"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
)

const (
	maxPerPage = 100
)

func handleServerError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	sharedlogging.GetLogger(r.Context()).Error(err)
	err = json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
		ErrorCode:    "INTERNAL",
		ErrorMessage: err.Error(),
	})
	if err != nil {
		panic(err)
	}
}

func handleClientError(w http.ResponseWriter, r *http.Request, err error) {
	w.WriteHeader(http.StatusBadRequest)
	sharedlogging.GetLogger(r.Context()).Error(err)
	err = json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
		ErrorCode:    "INTERNAL",
		ErrorMessage: err.Error(),
	})
	if err != nil {
		panic(err)
	}
}

func ListPaymentsHandler(db *mongo.Database) http.HandlerFunc {
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
						handleClientError(w, r, errors.New("sort order not well specified, got "+parts[1]))
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
		skip, err := IntegerWithDefault(r, "skip", 0)
		if err != nil {
			handleClientError(w, r, err)
			return
		}
		if skip != 0 {
			pipeline = append(pipeline, map[string]any{
				"$skip": skip,
			})
		}
		limit, err := IntegerWithDefault(r, "limit", maxPerPage)
		if err != nil {
			handleClientError(w, r, err)
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
		pipeline = append(pipeline, map[string]any{
			"$addFields": map[string]any{
				"actualPayment": map[string]any{
					"$first": "$items",
				},
			},
		})
		pipeline = append(pipeline, map[string]any{
			"$replaceRoot": map[string]any{
				"newRoot": "$actualPayment",
			},
		})

		cursor, err := db.Collection(payment.PaymentsCollection).Aggregate(r.Context(), pipeline)
		if err != nil {
			handleServerError(w, r, err)
			return
		}
		defer cursor.Close(r.Context())

		ret := make([]payment.Payment, 0)
		err = cursor.All(r.Context(), &ret)
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		err = json.NewEncoder(w).Encode(sharedapi.BaseResponse{
			Data: ret,
		})
		if err != nil {
			handleServerError(w, r, err)
			return
		}
	}
}

func ReadPaymentHandler(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		paymentId := mux.Vars(r)["paymentId"]

		identifier, err := payment.IdentifierFromString(paymentId)
		if err != nil {
			handleClientError(w, r, err)
			return
		}

		ret := db.Collection(payment.PaymentsCollection).FindOne(r.Context(), identifier)
		if ret.Err() != nil {
			if ret.Err() == mongo.ErrNoDocuments {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			handleServerError(w, r, ret.Err())
			return
		}
		type Object struct {
			Items []payment.Payment `bson:"items"`
		}
		ob := &Object{}
		err = ret.Decode(ob)
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		err = json.NewEncoder(w).Encode(sharedapi.BaseResponse{
			Data: ob.Items[0],
		})
		if err != nil {
			handleServerError(w, r, err)
			return
		}
	}
}

func wrapHandler(useScopes bool, h http.Handler, scopes ...string) http.Handler {
	if !useScopes {
		return h
	}
	return sharedauth.NeedOneOfScopes(scopes...)(h)
}

func PaymentsRouter(
	db *mongo.Database,
	useScopes bool) *mux.Router {
	router := mux.NewRouter()
	router.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			h.ServeHTTP(w, r)
		})
	})
	router.Path("/payments").Methods(http.MethodGet).Handler(wrapHandler(useScopes, ListPaymentsHandler(db), ScopeReadPayments, ScopeWritePayments))
	router.Path("/payments/{paymentId}").Methods(http.MethodGet).Handler(wrapHandler(useScopes, ReadPaymentHandler(db), ScopeReadPayments, ScopeWritePayments))

	return router
}

func ConnectorRouter[T bridge.ConnectorConfigObject, S bridge.ConnectorState](
	name string,
	useScopes bool,
	manager *bridge.ConnectorManager[T, S],
) *mux.Router {
	r := mux.NewRouter()
	r.Path("/" + name).Methods(http.MethodPut).Handler(
		wrapHandler(useScopes, EnableConnector(manager), ScopeWriteConnectors),
	)
	r.Path("/" + name).Methods(http.MethodDelete).Handler(
		wrapHandler(useScopes, DisableConnector(manager), ScopeWriteConnectors),
	)
	r.Path("/" + name + "/config").Methods(http.MethodGet).Handler(
		wrapHandler(useScopes, ReadConnectorConfig(manager), ScopeReadConnectors),
	)
	r.Path("/" + name + "/state").Methods(http.MethodGet).Handler(
		wrapHandler(useScopes, ReadConnectorState(manager), ScopeReadConnectors),
	)
	return r
}
