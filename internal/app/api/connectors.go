package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/numary/payments/internal/pkg/integration"
	"github.com/numary/payments/internal/pkg/payments"

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

func ReadConfig[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](cm *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config, err := cm.ReadConfig(r.Context())
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

func ListTasks[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](cm *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks, err := cm.ListTasksStates(r.Context())
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

func ReadTask[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](cm *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var descriptor Descriptor
		payments.DescriptorFromID(mux.Vars(r)["taskId"], &descriptor)

		tasks, err := cm.ReadTaskState(r.Context(), descriptor)
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

func Uninstall[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](cm *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := cm.Uninstall(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func Install[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](cm *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		installed, err := cm.IsInstalled(context.Background())
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
			err := json.NewDecoder(r.Body).Decode(&config)
			if err != nil {
				handleError(w, r, err)
				return
			}
		}

		err = cm.Install(r.Context(), config)
		if err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func Reset[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](cm *integration.ConnectorManager[Config, Descriptor]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		installed, err := cm.IsInstalled(context.Background())
		if err != nil {
			handleError(w, r, err)
			return
		}
		if !installed {
			handleError(w, r, errors.New("connector not installed"))
			return
		}

		err = cm.Reset(r.Context())
		if err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
