package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	internalTime "github.com/formancehq/go-libs/v3/time"
	internalEvents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/uptrace/bun"
)

type trade struct {
	bun.BaseModel `bun:"table:trades"`

	// Auto-increment
	SortID int64 `bun:"sort_id,autoincrement"`

	// Mandatory fields
	ID               models.TradeID             `bun:"id,pk,type:character varying,notnull"`
	ConnectorID      models.ConnectorID         `bun:"connector_id,type:character varying,notnull"`
	Reference        string                     `bun:"reference,type:text,notnull"`
	CreatedAt        internalTime.Time          `bun:"created_at,type:timestamp without time zone,notnull"`
	UpdatedAt        internalTime.Time          `bun:"updated_at,type:timestamp without time zone,notnull"`
	InstrumentType   models.TradeInstrumentType `bun:"instrument_type,type:text,notnull"`
	ExecutionModel   models.TradeExecutionModel `bun:"execution_model,type:text,notnull"`
	MarketSymbol     string                     `bun:"market_symbol,type:text,notnull"`
	MarketBaseAsset  string                     `bun:"market_base_asset,type:text,notnull"`
	MarketQuoteAsset string                     `bun:"market_quote_asset,type:text,notnull"`
	Side             models.TradeSide           `bun:"side,type:text,notnull"`
	Status           models.TradeStatus         `bun:"status,type:text,notnull"`
	Requested        json.RawMessage            `bun:"requested,type:jsonb,notnull"`
	Executed         json.RawMessage            `bun:"executed,type:jsonb,notnull"`
	Fills            json.RawMessage            `bun:"fills,type:jsonb,notnull"`
	Legs             json.RawMessage            `bun:"legs,type:jsonb,notnull"`
	Raw              json.RawMessage            `bun:"raw,type:json,notnull"`

	// Optional fields
	PortfolioAccountID *models.AccountID        `bun:"portfolio_account_id,type:character varying,nullzero"`
	OrderType          *models.TradeOrderType   `bun:"order_type,type:text,nullzero"`
	TimeInForce        *models.TradeTimeInForce `bun:"time_in_force,type:text,nullzero"`

	// Optional with defaults
	Fees     json.RawMessage   `bun:"fees,type:jsonb,nullzero,notnull,default:'[]'"`
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func fromTradeModels(t models.Trade) (*trade, error) {
	requestedJSON, err := json.Marshal(t.Requested)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal requested: %w", err)
	}

	executedJSON, err := json.Marshal(t.Executed)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal executed: %w", err)
	}

	feesJSON, err := json.Marshal(t.Fees)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fees: %w", err)
	}

	fillsJSON, err := json.Marshal(t.Fills)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fills: %w", err)
	}

	legsJSON, err := json.Marshal(t.Legs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal legs: %w", err)
	}

	return &trade{
		ID:                 t.ID,
		ConnectorID:        t.ConnectorID,
		Reference:          t.Reference,
		CreatedAt:          internalTime.New(t.CreatedAt),
		UpdatedAt:          internalTime.New(t.UpdatedAt),
		PortfolioAccountID: t.PortfolioAccountID,
		InstrumentType:     t.InstrumentType,
		ExecutionModel:     t.ExecutionModel,
		MarketSymbol:       t.Market.Symbol,
		MarketBaseAsset:    t.Market.BaseAsset,
		MarketQuoteAsset:   t.Market.QuoteAsset,
		Side:               t.Side,
		OrderType:          t.OrderType,
		TimeInForce:        t.TimeInForce,
		Status:             t.Status,
		Requested:          requestedJSON,
		Executed:           executedJSON,
		Fees:               feesJSON,
		Fills:              fillsJSON,
		Legs:               legsJSON,
		Metadata:           t.Metadata,
		Raw:                t.Raw,
	}, nil
}

func (t *trade) toTradeModels() (*models.Trade, error) {
	var requested models.TradeRequested
	if err := json.Unmarshal(t.Requested, &requested); err != nil {
		return nil, fmt.Errorf("failed to unmarshal requested: %w", err)
	}

	var executed models.TradeExecuted
	if err := json.Unmarshal(t.Executed, &executed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal executed: %w", err)
	}

	var fees []models.TradeFee
	if err := json.Unmarshal(t.Fees, &fees); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fees: %w", err)
	}

	var fills []models.TradeFill
	if err := json.Unmarshal(t.Fills, &fills); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fills: %w", err)
	}

	var legs []models.TradeLeg
	if err := json.Unmarshal(t.Legs, &legs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal legs: %w", err)
	}

	return &models.Trade{
		ID:                 t.ID,
		ConnectorID:        t.ConnectorID,
		Reference:          t.Reference,
		CreatedAt:          t.CreatedAt.Time,
		UpdatedAt:          t.UpdatedAt.Time,
		PortfolioAccountID: t.PortfolioAccountID,
		InstrumentType:     t.InstrumentType,
		ExecutionModel:     t.ExecutionModel,
		Market: models.TradeMarket{
			Symbol:     t.MarketSymbol,
			BaseAsset:  t.MarketBaseAsset,
			QuoteAsset: t.MarketQuoteAsset,
		},
		Side:        t.Side,
		OrderType:   t.OrderType,
		TimeInForce: t.TimeInForce,
		Status:      t.Status,
		Requested:   requested,
		Executed:    executed,
		Fees:        fees,
		Fills:       fills,
		Legs:        legs,
		Metadata:    t.Metadata,
		Raw:         t.Raw,
	}, nil
}

func (s *store) TradesUpsert(ctx context.Context, trades []models.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	toInsert := make([]*trade, 0, len(trades))
	for _, t := range trades {
		trade, err := fromTradeModels(t)
		if err != nil {
			return err
		}
		toInsert = append(toInsert, trade)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return e("failed to create transaction", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	var insertedTrades []trade
	err = tx.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Set("status = EXCLUDED.status").
		Set("executed = EXCLUDED.executed").
		Set("fills = EXCLUDED.fills").
		Set("fees = EXCLUDED.fees").
		Set("legs = EXCLUDED.legs").
		Set("metadata = EXCLUDED.metadata").
		Set("raw = EXCLUDED.raw").
		Returning("*").
		Scan(ctx, &insertedTrades)

	if err != nil {
		return e("failed to upsert trades", err)
	}

	var outboxEvents []models.OutboxEvent
	for _, t := range insertedTrades {
		tradeModel, err := t.toTradeModels()
		if err != nil {
			return e("failed to convert trade to model", err)
		}

		payload := internalEvents.TradeMessagePayload{
			ID:             tradeModel.ID.String(),
			ConnectorID:    tradeModel.ConnectorID.String(),
			Provider:       models.ToV3Provider(tradeModel.ConnectorID.Provider),
			Reference:      tradeModel.Reference,
			CreatedAt:      tradeModel.CreatedAt,
			UpdatedAt:      tradeModel.UpdatedAt,
			InstrumentType: tradeModel.InstrumentType.String(),
			ExecutionModel: tradeModel.ExecutionModel.String(),
			Market: internalEvents.TradeMarketPayload{
				Symbol:     tradeModel.Market.Symbol,
				BaseAsset:  tradeModel.Market.BaseAsset,
				QuoteAsset: tradeModel.Market.QuoteAsset,
			},
			Side:     tradeModel.Side.String(),
			Status:   tradeModel.Status.String(),
			Metadata: tradeModel.Metadata,
			RawData:  tradeModel.Raw,
		}

		if tradeModel.PortfolioAccountID != nil {
			payload.PortfolioAccountID = tradeModel.PortfolioAccountID.String()
		}

		if tradeModel.OrderType != nil {
			payload.OrderType = tradeModel.OrderType.String()
		}

		if tradeModel.TimeInForce != nil {
			payload.TimeInForce = tradeModel.TimeInForce.String()
		}

		// Marshal complex objects
		requestedJSON, _ := json.Marshal(tradeModel.Requested)
		payload.Requested = requestedJSON

		executedJSON, _ := json.Marshal(tradeModel.Executed)
		payload.Executed = executedJSON

		feesJSON, _ := json.Marshal(tradeModel.Fees)
		payload.Fees = feesJSON

		fillsJSON, _ := json.Marshal(tradeModel.Fills)
		payload.Fills = fillsJSON

		legsJSON, _ := json.Marshal(tradeModel.Legs)
		payload.Legs = legsJSON

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return e("failed to marshal trade event payload", err)
		}

		outboxEvents = append(outboxEvents, models.OutboxEvent{
			ID: models.EventID{
				EventIdempotencyKey: tradeModel.IdempotencyKey(),
				ConnectorID:         &tradeModel.ConnectorID,
			},
			EventType:   events.EventTypeSavedTrade,
			EntityID:    tradeModel.ID.String(),
			Payload:     payloadBytes,
			CreatedAt:   internalTime.Now().UTC().Time,
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &tradeModel.ConnectorID,
		})
	}

	if len(outboxEvents) > 0 {
		if err = s.OutboxEventsInsert(ctx, tx, outboxEvents); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return e("failed to commit transaction", err)
	}

	return nil
}

func (s *store) TradesUpdateMetadata(ctx context.Context, id models.TradeID, metadata map[string]string) error {
	_, err := s.db.NewUpdate().
		Model((*trade)(nil)).
		Set("metadata = ?", metadata).
		Where("id = ?", id).
		Exec(ctx)

	return e("failed to update trade metadata", err)
}

func (s *store) TradesGet(ctx context.Context, id models.TradeID) (*models.Trade, error) {
	var t trade

	err := s.db.NewSelect().
		Model(&t).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get trade", err)
	}

	return t.toTradeModels()
}

type TradeQuery struct{}

type ListTradesQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[TradeQuery]]

func NewListTradesQuery(opts bunpaginate.PaginatedQueryOptions[TradeQuery]) ListTradesQuery {
	return ListTradesQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) tradesQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch key {
		case "id",
			"reference",
			"connector_id",
			"status",
			"side",
			"instrument_type",
			"execution_model",
			"order_type",
			"time_in_force",
			"market_symbol",
			"market_base_asset",
			"market_quote_asset":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		case "created_at", "updated_at":
			mapped, ok := query.DefaultComparisonOperatorsMapping[operator]
			if !ok {
				return "", nil, fmt.Errorf("unsupported operator '%s' for column '%s': %w", operator, key, ErrValidation)
			}
			return fmt.Sprintf("%s %s ?", key, mapped), []any{value}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))
}

func (s *store) TradesList(ctx context.Context, q ListTradesQuery) (*bunpaginate.Cursor[models.Trade], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.tradesQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[TradeQuery], trade](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[TradeQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}
			query = query.Order("created_at DESC", "sort_id DESC")
			return query
		},
	)
	if err != nil {
		return nil, e("failed to list trades", err)
	}

	modelTrades := make([]models.Trade, 0, len(cursor.Data))
	for _, t := range cursor.Data {
		mt, err := t.toTradeModels()
		if err != nil {
			return nil, err
		}
		modelTrades = append(modelTrades, *mt)
	}

	return &bunpaginate.Cursor[models.Trade]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     modelTrades,
	}, nil
}

func (s *store) TradesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*trade)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return e("failed to delete trades from connector", err)
}

func (s *store) TradesDelete(ctx context.Context, id models.TradeID) error {
	_, err := s.db.NewDelete().
		Model((*trade)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return e("failed to delete trade", err)
}
