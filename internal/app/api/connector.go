package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/numary/go-libs/sharedapi"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/internal/pkg/integration"
	"github.com/numary/payments/internal/pkg/payments"
	"go.mongodb.org/mongo-driver/mongo"
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

func readConnectorsHandler(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cursor, err := db.Collection(payments.ConnectorsCollection).
			Find(r.Context(), map[string]any{})
		if err != nil {
			handleError(w, r, err)

			return
		}
		defer cursor.Close(r.Context())

		res := make([]payments.ConnectorBaseInfo, 0)
		if err = cursor.All(r.Context(), &res); err != nil {
			handleError(w, r, err)

			return
		}

		err = json.NewEncoder(w).Encode(
			sharedapi.BaseResponse[[]payments.ConnectorBaseInfo]{
				Data: &res,
			})
		if err != nil {
			handleServerError(w, r, err)

			return
		}
	}
}

func readConfig[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config, err := connectorManager.ReadConfig(r.Context())
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

func listTasks[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks, err := connectorManager.ListTasksStates(r.Context())
		if err != nil {
			handleError(w, r, err)

			return
		}

		err = json.NewEncoder(w).Encode(tasks)
		if err != nil {
			panic(err)
		}
	}
}

func readTask[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var descriptor Descriptor

		payments.DescriptorFromID(mux.Vars(r)["taskID"], &descriptor)

		tasks, err := connectorManager.ReadTaskState(r.Context(), descriptor)
		if err != nil {
			handleError(w, r, err)

			return
		}

		err = json.NewEncoder(w).Encode(tasks)
		if err != nil {
			panic(err)
		}
	}
}

func uninstall[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := connectorManager.Uninstall(r.Context())
		if err != nil {
			handleError(w, r, err)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func install[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		installed, err := connectorManager.IsInstalled(context.Background())
		if err != nil {
			handleError(w, r, err)

			return
		}

		if installed {
			handleError(w, r, integration.ErrAlreadyInstalled)

			return
		}

		var config Config
		if r.ContentLength > 0 {
			err = json.NewDecoder(r.Body).Decode(&config)
			if err != nil {
				handleError(w, r, err)

				return
			}
		}

		err = connectorManager.Install(r.Context(), config)
		if err != nil {
			handleError(w, r, err)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func reset[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		installed, err := connectorManager.IsInstalled(context.Background())
		if err != nil {
			handleError(w, r, err)

			return
		}

		if !installed {
			handleError(w, r, errors.New("connector not installed"))

			return
		}

		err = connectorManager.Reset(r.Context())
		if err != nil {
			handleError(w, r, err)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}