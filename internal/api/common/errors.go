package common

import (
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/go-libs/v5/pkg/observe/log"
)

func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	logging.FromContext(r.Context()).Error(err)
	api.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrorInternal, errors.New("internal error, consult logs/traces for details"))
}
