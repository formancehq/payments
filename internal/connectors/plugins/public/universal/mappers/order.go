package mappers

import (
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

// OrderToPSPOrder parses every numeric field eagerly so a malformed amount
// fails the row rather than silently producing a zero — silent zeros would
// create misleading adjustments in the engine's order history.
func OrderToPSPOrder(o client.Order) (models.PSPOrder, error) {
	parse := func(label, raw string) (*big.Int, error) {
		v, err := ParseAmount(raw)
		if err != nil {
			return nil, fmt.Errorf("order %s: %w", label, err)
		}
		return v, nil
	}

	type field struct {
		name   string
		raw    string
		target **big.Int
	}
	var baseOrdered, baseFilled, limit, stop, quote, fee, avg *big.Int
	for _, f := range []field{
		{"baseQuantityOrdered", o.BaseQuantityOrdered, &baseOrdered},
		{"baseQuantityFilled", o.BaseQuantityFilled, &baseFilled},
		{"limitPrice", o.LimitPrice, &limit},
		{"stopPrice", o.StopPrice, &stop},
		{"quoteAmount", o.QuoteAmount, &quote},
		{"fee", o.Fee, &fee},
		{"averageFillPrice", o.AverageFillPrice, &avg},
	} {
		v, err := parse(f.name, f.raw)
		if err != nil {
			return models.PSPOrder{}, err
		}
		*f.target = v
	}

	r, err := Raw(o)
	if err != nil {
		return models.PSPOrder{}, err
	}
	return models.PSPOrder{
		Reference:                   o.Reference,
		ClientOrderID:               o.ClientOrderID,
		CreatedAt:                   DefaultTime(o.CreatedAt, o.UpdatedAt),
		Direction:                   OrderDirection(o.Direction),
		SourceAsset:                 o.SourceAsset,
		DestinationAsset:            o.DestinationAsset,
		Type:                        OrderType(o.Type),
		Status:                      OrderStatus(o.Status),
		BaseQuantityOrdered:         baseOrdered,
		BaseQuantityFilled:          baseFilled,
		LimitPrice:                  limit,
		StopPrice:                   stop,
		TimeInForce:                 TimeInForce(o.TimeInForce),
		ExpiresAt:                   o.ExpiresAt,
		QuoteAmount:                 quote,
		QuoteAsset:                  o.QuoteAsset,
		Fee:                         fee,
		FeeAsset:                    o.FeeAsset,
		AverageFillPrice:            avg,
		PriceAsset:                  o.PriceAsset,
		SourceAccountReference:      o.SourceAccountReference,
		DestinationAccountReference: o.DestinationAccountReference,
		Metadata:                    o.Metadata,
		Raw:                         r,
	}, nil
}

func OrderDirection(s string) models.OrderDirection {
	var d models.OrderDirection
	if err := d.Scan(s); err != nil {
		return models.ORDER_DIRECTION_UNKNOWN
	}
	return d
}

func OrderType(s string) models.OrderType {
	var t models.OrderType
	if err := t.Scan(s); err != nil {
		return models.ORDER_TYPE_UNKNOWN
	}
	return t
}

func OrderStatus(s string) models.OrderStatus {
	var st models.OrderStatus
	if err := st.Scan(s); err != nil {
		return models.ORDER_STATUS_UNKNOWN
	}
	return st
}

func TimeInForce(s string) models.TimeInForce {
	if s == "" {
		return models.TIME_IN_FORCE_UNKNOWN
	}
	var t models.TimeInForce
	if err := t.Scan(s); err != nil {
		return models.TIME_IN_FORCE_UNKNOWN
	}
	return t
}
