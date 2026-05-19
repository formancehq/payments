package mappers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// ConversionMapResult tells the orchestrator how to handle the row.
//   - Conversion != nil — emit it.
//   - Skip = true       — silently skip (non-type-36 row, or a row
//                         that does not reduce to a single negative +
//                         single positive non-zero known currency).
//   - DerivativesRow    — Warn-and-skip; spot-only stance.
type ConversionMapResult struct {
	Conversion     *models.PSPConversion
	Skip           bool
	DerivativesRow bool
}

// UserTransactionToPSPConversion is the conversions equivalent of
// UserTransactionToPSPPayment: same stream, same since_id cursor,
// different detection rule and target shape.
//
// Detection (MAPPINGS.md §3.5):
//
//   - tx.Type == "36" (instant buy/sell)
//   - exactly one negative + one positive non-zero known-currency
//     amount (the base/quote swap)
//
// The dynamic <src>_<dst> pair-rate key is surfaced in metadata via
// MetadataKeyRate; the orchestrator does not need to re-parse Raw to
// retrieve the rate.
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
		// Synthesise a stable currency_pair string when Bitstamp did
		// not return a <src>_<dst> rate key. Keeps metadata uniform.
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

	// Bitstamp charges instant buy/sell fees in the quote (destination)
	// currency. If the row carries no fee, leave Fee/FeeAsset zero/nil
	// rather than fabricating a zero amount denominated in the wrong
	// asset.
	if !IsZeroAmount(tx.Fee) {
		fee, ferr := ParseAmount(AbsAmount(tx.Fee), destination.Precision)
		if ferr != nil {
			return ConversionMapResult{}, fmt.Errorf("conversion tx %d fee: %w", tx.ID, ferr)
		}
		conv.Fee = fee
		feeAsset := destination.Asset
		conv.FeeAsset = &feeAsset
	}

	return ConversionMapResult{Conversion: conv}, nil
}

// pickPairRate finds the <src>_<dst> or <dst>_<src> rate key, returning
// the canonical key string ("src_dst" lowercase) and the rate string.
// Either order is accepted because Bitstamp emits the rate keyed on
// the market pair, not the trade direction.
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
