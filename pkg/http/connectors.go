package http

import (
	"encoding/json"
	"github.com/numary/payments/pkg/bridge"
	"net/http"
)

func ReadConnectorConfig[T bridge.ConnectorConfigObject, S bridge.ConnectorState](cm *bridge.ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var config T
		err := cm.ReadConfig(r.Context(), &config)
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		err = json.NewEncoder(w).Encode(config)
		if err != nil {
			panic(err)
		}
	}
}

func ReadConnectorState[T bridge.ConnectorConfigObject, S bridge.ConnectorState](cm *bridge.ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var state S
		err := cm.ReadState(r.Context(), &state)
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		err = json.NewEncoder(w).Encode(state)
		if err != nil {
			panic(err)
		}
	}
}

func DisableConnector[T bridge.ConnectorConfigObject, S bridge.ConnectorState](cm *bridge.ConnectorManager[T, S]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := cm.Disable(r.Context())
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		err = cm.Stop(r.Context())
		if err != nil {
			handleServerError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func EnableConnector[T bridge.ConnectorConfigObject, S bridge.ConnectorState](cm *bridge.ConnectorManager[T, S]) http.HandlerFunc {
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
				handleServerError(w, r, err)
				return
			}
		} else {
			err := cm.ReadConfig(r.Context(), &config)
			if err != nil {
				handleServerError(w, r, err)
				return
			}
		}

		err := cm.Enable(r.Context())
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		err = cm.Stop(r.Context())
		if err != nil {
			handleServerError(w, r, err)
			return
		}

		err = cm.StartWithConfig(r.Context(), config)
		if err != nil {
			handleServerError(w, r, err)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
