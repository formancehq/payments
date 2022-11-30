package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/formancehq/payments/internal/pkg/integration"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/pkg/errors"

	stripeConnector "github.com/formancehq/payments/internal/pkg/connectors/stripe"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/transfer"
)

type stripeTransferRequest struct {
	Amount      int64             `json:"amount"`
	Asset       string            `json:"asset"`
	Destination string            `json:"destination"`
	Metadata    map[string]string `json:"metadata"`

	currency string
}

func (req *stripeTransferRequest) validate() error {
	if req.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	if req.Asset == "" {
		return errors.New("asset is required")
	}

	if req.Asset != "USD/2" && req.Asset != "EUR/2" {
		return errors.New("asset must be USD/2 or EUR/2")
	}

	req.currency = req.Asset[:3]

	if req.Destination == "" {
		return errors.New("destination is required")
	}

	return nil
}

func handleStripeTransfers(db *mongo.Database) http.HandlerFunc {
	connectorStore := integration.NewMongoDBConnectorStore(db)
	var cfg stripeConnector.Config

	err := connectorStore.ReadConfig(context.Background(), stripeConnector.Name, &cfg)
	if err != nil {
		panic(err)
	}

	stripe.Key = cfg.APIKey

	return func(w http.ResponseWriter, r *http.Request) {
		var transferRequest stripeTransferRequest

		err = json.NewDecoder(r.Body).Decode(&transferRequest)
		if err != nil {
			handleError(w, r, err)

			return
		}

		err = transferRequest.validate()
		if err != nil {
			handleError(w, r, err)

			return
		}

		params := &stripe.TransferParams{
			Amount:      stripe.Int64(transferRequest.Amount),
			Currency:    stripe.String(transferRequest.Currency),
			Destination: stripe.String(transferRequest.Destination),
		}

		for k, v := range transferRequest.Metadata {
			params.AddMetadata(k, v)
		}

		transferResponse, err := transfer.New(params)
		if err != nil {
			handleServerError(w, r, err)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(transferResponse)
		if err != nil {
			handleServerError(w, r, err)

			return
		}
	}
}
