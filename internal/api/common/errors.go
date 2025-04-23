package common

import (
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/logging"
)

func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	logging.FromContext(r.Context()).Error(err)
	api.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrorInternal, errors.New("Internal error. Consult logs/traces to have more details."))
}
