package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreateTradeMarketRequest struct {
	Symbol     string `json:"symbol" validate:"required"`
	BaseAsset  string `json:"baseAsset" validate:"required,asset"`
	QuoteAsset string `json:"quoteAsset" validate:"required,asset"`
}

type CreateTradeRequestedRequest struct {
	Quantity            *string `json:"quantity"`
	LimitPrice          *string `json:"limitPrice"`
	NotionalQuoteAmount *string `json:"notionalQuoteAmount"`
	ClientOrderID       *string `json:"clientOrderID"`
}

type CreateTradeExecutedRequest struct {
	Quantity     *string    `json:"quantity"`
	QuoteAmount  *string    `json:"quoteAmount"`
	AveragePrice *string    `json:"averagePrice"`
	CompletedAt  *time.Time `json:"completedAt"`
}

type CreateTradeFeeRequest struct {
	Asset     string  `json:"asset" validate:"required,asset"`
	Amount    string  `json:"amount" validate:"required"`
	Kind      *string `json:"kind" validate:"omitempty,tradeFeeKind"`
	AppliedOn *string `json:"appliedOn" validate:"omitempty,tradeFeeAppliedOn"`
	Rate      *string `json:"rate"`
}

type CreateTradeFillRequest struct {
	TradeReference string                  `json:"tradeReference" validate:"required"`
	Timestamp      time.Time               `json:"timestamp" validate:"required"`
	Price          string                  `json:"price" validate:"required"`
	Quantity       string                  `json:"quantity" validate:"required"`
	QuoteAmount    string                  `json:"quoteAmount" validate:"required"`
	Liquidity      *string                 `json:"liquidity" validate:"omitempty,tradeLiquidity"`
	Fees           []CreateTradeFeeRequest `json:"fees"`
	Raw            json.RawMessage         `json:"raw"`
}

type CreateTradeRequest struct {
	Reference          string                       `json:"reference" validate:"required,gte=3,lte=1000"`
	ConnectorID        string                       `json:"connectorID" validate:"required,connectorID"`
	CreatedAt          time.Time                    `json:"createdAt" validate:"required,lte=now"`
	UpdatedAt          *time.Time                   `json:"updatedAt" validate:"omitempty,lte=now"`
	PortfolioAccountID *string                      `json:"portfolioAccountID" validate:"omitempty,accountID"`
	InstrumentType     string                       `json:"instrumentType" validate:"required,tradeInstrumentType"`
	ExecutionModel     string                       `json:"executionModel" validate:"required,tradeExecutionModel"`
	Market             CreateTradeMarketRequest     `json:"market" validate:"required"`
	Side               string                       `json:"side" validate:"required,tradeSide"`
	OrderType          *string                      `json:"orderType" validate:"omitempty,tradeOrderType"`
	TimeInForce        *string                      `json:"timeInForce" validate:"omitempty,tradeTimeInForce"`
	Status             string                       `json:"status" validate:"required,tradeStatus"`
	Requested          CreateTradeRequestedRequest  `json:"requested"`
	Executed           CreateTradeExecutedRequest   `json:"executed" validate:"required"`
	Fees               []CreateTradeFeeRequest      `json:"fees"`
	Fills              []CreateTradeFillRequest     `json:"fills" validate:"min=1,dive"`
	Metadata           map[string]string            `json:"metadata"`
	Raw                json.RawMessage              `json:"raw"`
}

func tradesCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_tradesCreate")
		defer span.End()

		var req CreateTradeRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromTradeCreateRequest(span, req)

		if _, err := validator.Validate(req); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectorID := models.MustConnectorIDFromString(req.ConnectorID)
		tradeID := models.TradeID_FromReference(req.Reference, connectorID)

		// Convert requested
		requested := models.TradeRequested{
			Quantity:            req.Requested.Quantity,
			LimitPrice:          req.Requested.LimitPrice,
			NotionalQuoteAmount: req.Requested.NotionalQuoteAmount,
			ClientOrderID:       req.Requested.ClientOrderID,
		}

		// Convert executed
		executed := models.TradeExecuted{
			Quantity:     req.Executed.Quantity,
			QuoteAmount:  req.Executed.QuoteAmount,
			AveragePrice: req.Executed.AveragePrice,
			CompletedAt:  req.Executed.CompletedAt,
		}

		// Convert fees
		fees := make([]models.TradeFee, 0, len(req.Fees))
		for _, f := range req.Fees {
			fee := models.TradeFee{
				Asset:  f.Asset,
				Amount: f.Amount,
				Rate:   f.Rate,
			}
			if f.Kind != nil {
				feeKind := models.MustTradeFeeKindFromString(*f.Kind)
				fee.Kind = &feeKind
			}
			if f.AppliedOn != nil {
				appliedOn := models.MustTradeFeeAppliedOnFromString(*f.AppliedOn)
				fee.AppliedOn = &appliedOn
			}
			fees = append(fees, fee)
		}

		// Convert fills
		fills := make([]models.TradeFill, 0, len(req.Fills))
		for _, fill := range req.Fills {
			fillFees := make([]models.TradeFee, 0, len(fill.Fees))
			for _, f := range fill.Fees {
				fee := models.TradeFee{
					Asset:  f.Asset,
					Amount: f.Amount,
					Rate:   f.Rate,
				}
				if f.Kind != nil {
					feeKind := models.MustTradeFeeKindFromString(*f.Kind)
					fee.Kind = &feeKind
				}
				if f.AppliedOn != nil {
					appliedOn := models.MustTradeFeeAppliedOnFromString(*f.AppliedOn)
					fee.AppliedOn = &appliedOn
				}
				fillFees = append(fillFees, fee)
			}

			tradeFill := models.TradeFill{
				TradeReference: fill.TradeReference,
				Timestamp:      fill.Timestamp.UTC(),
				Price:          fill.Price,
				Quantity:       fill.Quantity,
				QuoteAmount:    fill.QuoteAmount,
				Fees:           fillFees,
				Raw:            fill.Raw,
			}
			if fill.Liquidity != nil {
				liquidity := models.MustTradeLiquidityFromString(*fill.Liquidity)
				tradeFill.Liquidity = &liquidity
			}
			fills = append(fills, tradeFill)
		}

		updatedAt := req.CreatedAt
		if req.UpdatedAt != nil {
			updatedAt = *req.UpdatedAt
		}

		trade := models.Trade{
			ID:          tradeID,
			ConnectorID: connectorID,
			Reference:   req.Reference,
			CreatedAt:   req.CreatedAt.UTC(),
			UpdatedAt:   updatedAt.UTC(),
			PortfolioAccountID: func() *models.AccountID {
				if req.PortfolioAccountID == nil {
					return nil
				}
				return pointer.For(models.MustAccountIDFromString(*req.PortfolioAccountID))
			}(),
			InstrumentType: models.MustTradeInstrumentTypeFromString(req.InstrumentType),
			ExecutionModel: models.MustTradeExecutionModelFromString(req.ExecutionModel),
			Market: models.TradeMarket{
				Symbol:     req.Market.Symbol,
				BaseAsset:  req.Market.BaseAsset,
				QuoteAsset: req.Market.QuoteAsset,
			},
			Side:   models.MustTradeSideFromString(req.Side),
			Status: models.MustTradeStatusFromString(req.Status),
			OrderType: func() *models.TradeOrderType {
				if req.OrderType == nil {
					return nil
				}
				ot := models.MustTradeOrderTypeFromString(*req.OrderType)
				return &ot
			}(),
			TimeInForce: func() *models.TradeTimeInForce {
				if req.TimeInForce == nil {
					return nil
				}
				tif := models.MustTradeTimeInForceFromString(*req.TimeInForce)
				return &tif
			}(),
			Requested: requested,
			Executed:  executed,
			Fees:      fees,
			Fills:     fills,
			Legs:      []models.TradeLeg{}, // Will be populated after payments are created
			Metadata:  req.Metadata,
			Raw:       req.Raw,
		}

		// Validate trade math
		check := models.ValidateTradeMath(trade)
		if !check.OK {
			errMsg := fmt.Sprintf("Trade validation failed: %v", check.Errors)
			err := fmt.Errorf("%s", errMsg)
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		err = backend.TradesCreate(ctx, trade)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, trade)
	}
}

func populateSpanFromTradeCreateRequest(span trace.Span, req CreateTradeRequest) {
	span.SetAttributes(attribute.String("reference", req.Reference))
	span.SetAttributes(attribute.String("connectorID", req.ConnectorID))
	span.SetAttributes(attribute.String("createdAt", req.CreatedAt.String()))
	span.SetAttributes(attribute.String("instrumentType", req.InstrumentType))
	span.SetAttributes(attribute.String("executionModel", req.ExecutionModel))
	span.SetAttributes(attribute.String("market.symbol", req.Market.Symbol))
	span.SetAttributes(attribute.String("side", req.Side))
	span.SetAttributes(attribute.String("status", req.Status))
	if req.PortfolioAccountID != nil {
		span.SetAttributes(attribute.String("portfolioAccountID", *req.PortfolioAccountID))
	}
	span.SetAttributes(attribute.Int("fillsCount", len(req.Fills)))
	span.SetAttributes(attribute.Int("feesCount", len(req.Fees)))
}

