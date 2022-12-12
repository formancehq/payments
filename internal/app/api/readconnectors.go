package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/formancehq/payments/internal/app/models"

	"github.com/formancehq/go-libs/sharedapi"
	"github.com/formancehq/payments/internal/app/payments"
)

type readConnectorsRepository interface {
	ListConnectors(ctx context.Context) ([]*models.Connector, error)
}

func readConnectorsHandler(repo readConnectorsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := repo.ListConnectors(r.Context())
		if err != nil {
			handleError(w, r, err)

			return
		}

		data := make([]*payments.ConnectorBaseInfo, len(res))

		for i := range res {
			data[i] = &payments.ConnectorBaseInfo{
				Provider: res[i].Provider,
				Disabled: res[i].Enabled,
			}
		}

		err = json.NewEncoder(w).Encode(
			sharedapi.BaseResponse[[]*payments.ConnectorBaseInfo]{
				Data: &data,
			})
		if err != nil {
			panic(err)
		}
	}
}
