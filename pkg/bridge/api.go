package bridge

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs/sharedapi"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg"
	. "github.com/numary/payments/pkg/http"
	"net/http"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
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

func ReadConnectorConfig[T payments.ConnectorConfigObject, S payments.ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		config, _, err := cm.ReadConfig(r.Context())
		if err != nil {
			if err == ErrNotFound {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			handleError(w, r, err)
			return
		}

		err = json.NewEncoder(w).Encode(config)
		if err != nil {
			panic(err)
		}
	}
}

func ReadConnectorState[T payments.ConnectorConfigObject, S payments.ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		state, err := cm.ReadState(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}

		err = json.NewEncoder(w).Encode(state)
		if err != nil {
			panic(err)
		}
	}
}

func ResetConnector[T payments.ConnectorConfigObject, S payments.ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		err := cm.Reset(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func DisableConnector[T payments.ConnectorConfigObject, S payments.ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := cm.Disable(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}

		err = cm.Stop(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func EnableConnector[T payments.ConnectorConfigObject, S payments.ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var config *T
		if r.ContentLength > 0 {
			config = new(T)
			err := json.NewDecoder(r.Body).Decode(config)
			if err != nil {
				handleError(w, r, err)
				return
			}
			err = cm.Configure(r.Context(), *config)
			if err != nil {
				handleError(w, r, err)
				return
			}
		} else {
			var err error
			config, _, err = cm.ReadConfig(r.Context())
			if err != nil {
				handleError(w, r, err)
				return
			}
		}

		err := cm.Enable(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}

		err = cm.Stop(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}

		err = cm.StartWithConfig(r.Context(), *config)
		if err != nil {
			handleError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func ConnectorRouter[T payments.ConnectorConfigObject, S payments.ConnectorState](
	name string,
	useScopes bool,
	manager *ConnectorManager[T, S],
) *mux.Router {
	r := mux.NewRouter()
	r.Path("/" + name).Methods(http.MethodPut).Handler(
		WrapHandler(useScopes, EnableConnector(manager), ScopeWriteConnectors),
	)
	r.Path("/" + name).Methods(http.MethodDelete).Handler(
		WrapHandler(useScopes, DisableConnector(manager), ScopeWriteConnectors),
	)
	r.Path("/" + name + "/config").Methods(http.MethodGet).Handler(
		WrapHandler(useScopes, ReadConnectorConfig(manager), ScopeReadConnectors, ScopeWriteConnectors),
	)
	r.Path("/" + name + "/state").Methods(http.MethodGet).Handler(
		WrapHandler(useScopes, ReadConnectorState(manager), ScopeReadConnectors, ScopeWriteConnectors),
	)
	r.Path("/" + name + "/reset").Methods(http.MethodPut).Handler(
		WrapHandler(useScopes, ResetConnector(manager), ScopeWriteConnectors),
	)
	return r
}
