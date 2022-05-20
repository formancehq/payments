package bridge

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs/sharedapi"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/auth"
	"net/http"
)

// TODO: Properly handle errors
func handleError(w http.ResponseWriter, r *http.Request, err error) {
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

func ReadConnectorConfig[T ConnectorConfigObject, S ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var config T
		err := cm.ReadConfig(r.Context(), &config)
		if err != nil {
			handleError(w, r, err)
			return
		}

		err = json.NewEncoder(w).Encode(config)
		if err != nil {
			panic(err)
		}
	}
}

func ReadConnectorState[T ConnectorConfigObject, S ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var state S
		err := cm.ReadState(r.Context(), &state)
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

func ResetConnector[T ConnectorConfigObject, S ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		err := cm.Reset(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func DisableConnector[T ConnectorConfigObject, S ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
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

func EnableConnector[T ConnectorConfigObject, S ConnectorState](cm *ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var config T
		if r.ContentLength > 0 {
			if r.ContentLength > 0 {
				err := json.NewDecoder(r.Body).Decode(&config)
				if err != nil {
					panic(err)
				}
			}

			err := cm.Configure(r.Context(), config)
			if err != nil {
				handleError(w, r, err)
				return
			}
		} else {
			err := cm.ReadConfig(r.Context(), &config)
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

		err = cm.StartWithConfig(r.Context(), config)
		if err != nil {
			handleError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func ConnectorRouter[T ConnectorConfigObject, S ConnectorState](
	name string,
	useScopes bool,
	manager *ConnectorManager[T, S],
) *mux.Router {
	r := mux.NewRouter()
	r.Path("/" + name).Methods(http.MethodPut).Handler(
		bridge.WrapHandler(useScopes, EnableConnector(manager), ScopeWriteConnectors),
	)
	r.Path("/" + name).Methods(http.MethodDelete).Handler(
		bridge.WrapHandler(useScopes, DisableConnector(manager), ScopeWriteConnectors),
	)
	r.Path("/" + name + "/config").Methods(http.MethodGet).Handler(
		bridge.WrapHandler(useScopes, ReadConnectorConfig(manager), ScopeReadConnectors),
	)
	r.Path("/" + name + "/state").Methods(http.MethodGet).Handler(
		bridge.WrapHandler(useScopes, ReadConnectorState(manager), ScopeReadConnectors),
	)
	r.Path("/" + name + "/reset").Methods(http.MethodPut).Handler(
		bridge.WrapHandler(useScopes, ResetConnector(manager), ScopeWriteConnectors),
	)
	return r
}
