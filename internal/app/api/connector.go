package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/formancehq/payments/internal/pkg/integration"
	"github.com/formancehq/payments/internal/pkg/payments"

	"github.com/gorilla/mux"
	"github.com/numary/go-libs/sharedapi"
	"github.com/numary/go-libs/sharedlogging"
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

func readConfig[Config payments.ConnectorConfigObject,
	Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor],
) http.HandlerFunc {
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

func listTasks[Config payments.ConnectorConfigObject,
	Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor],
) http.HandlerFunc {
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

func readTask[Config payments.ConnectorConfigObject,
	Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor],
) http.HandlerFunc {
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

func findAll[Config payments.ConnectorConfigObject,
	Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor],
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := connectorManager.FindAll(context.Background())
		if err != nil {
			handleError(w, r, err)

			return
		}

		if err = json.NewEncoder(w).Encode(res); err != nil {
			panic(err)
		}
	}
}

func uninstall[Config payments.ConnectorConfigObject,
	Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor],
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := connectorManager.Uninstall(r.Context())
		if err != nil {
			handleError(w, r, err)

			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func install[Config payments.ConnectorConfigObject,
	Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor],
) http.HandlerFunc {
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

func reset[Config payments.ConnectorConfigObject,
	Descriptor payments.TaskDescriptor](connectorManager *integration.ConnectorManager[Config, Descriptor],
) http.HandlerFunc {
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
