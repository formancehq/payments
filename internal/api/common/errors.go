package common

import (
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
)

func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	api.InternalServerError(w, r, errors.New("Internal error. Consult logs/traces to have more details."))
}
