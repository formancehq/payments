package mappers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// ConversionMapResult — Conversion != nil → emit; Skip → ignore;
// DerivativesRow → orchestrator Errorf-skip (spot-only).
type ConversionMapResult struct {
	Conversion     *models.PSPConversion
	Skip           bool
	DerivativesRow bool
}

// UserTransactionToPSPConversion detects type-36 (instant buy/sell)
// rows with exactly one negative + one positive non-zero
// known-currency amount. See MAPPINGS §4.5.
func UserTransactionToPSPConversion(currencies map[string]int, tx client.UserTransaction) (ConversionMapResult, error) {
	if tx.HasDerivativesMarker() {
		return ConversionMapResult{Skip: true, DerivativesRow: true}, nil
	}
	if tx.Type != TxTypeBuySell {
		return ConversionMapResult{Skip: true}, nil
	}

	source, destination, ok, err := ResolveTwoAssetConversion(currencies, tx.CurrencyAmounts)
	if err != nil {
		return ConversionMapResult{}, fmt.Errorf("resolve conversion legs for tx %d: %w", tx.ID, err)
	}
	if !ok {
		return ConversionMapResult{Skip: true}, nil
	}

	createdAt, err := ParseBitstampTime(tx.Datetime)
	if err != nil {
		return ConversionMapResult{}, fmt.Errorf("conversion tx %d: %w", tx.ID, err)
	}

	raw, err := json.Marshal(tx)
	if err != nil {
		return ConversionMapResult{}, fmt.Errorf("marshal raw for tx %d: %w", tx.ID, err)
	}

	pairKey, pairRate := pickPairRate(tx.PairRates, source.Symbol, destination.Symbol)
	currencyPair := pairKey
	if currencyPair == "" {
		// Synthesise stable currency_pair when Bitstamp omits the rate key.
		currencyPair = strings.ToLower(source.Symbol) + "_" + strings.ToLower(destination.Symbol)
	}

	conv := &models.PSPConversion{
		Reference:                   strconv.FormatInt(tx.ID, 10),
		CreatedAt:                   createdAt,
		SourceAsset:                 source.Asset,
		DestinationAsset:            destination.Asset,
		SourceAmount:                source.Amount,
		DestinationAmount:           destination.Amount,
		Status:                      models.CONVERSION_STATUS_COMPLETED,
		SourceAccountReference:      strPtr(source.Symbol),
		DestinationAccountReference: strPtr(destination.Symbol),
		Metadata:                    ConversionMetadata(tx, currencyPair, pairRate),
		Raw:                         raw,
	}

	// Fee is in the quote currency. Leave Fee/FeeAsset nil when absent
	// rather than fabricating a zero amount in the wrong asset.
	if !IsZeroAmount(tx.Fee) {
		fee, ferr := ParseDecimalAmount(AbsAmount(tx.Fee), destination.Precision)
		if ferr != nil {
			return ConversionMapResult{}, fmt.Errorf("conversion tx %d fee: %w", tx.ID, ferr)
		}
		conv.Fee = fee
		feeAsset := destination.Asset
		conv.FeeAsset = &feeAsset
	}

	return ConversionMapResult{Conversion: conv}, nil
}

// pickPairRate tries both <src>_<dst> and <dst>_<src> — Bitstamp
// keys the rate on the market pair, not the trade direction.
func pickPairRate(pairs map[string]string, source, destination string) (key, rate string) {
	src := strings.ToLower(source)
	dst := strings.ToLower(destination)
	for _, candidate := range []string{src + "_" + dst, dst + "_" + src} {
		if v, ok := pairs[candidate]; ok && v != "" {
			return candidate, v
		}
	}
	return "", ""
}
