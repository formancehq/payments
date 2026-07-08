package mappers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

// accountRef returns the account reference for a raw Kraken code (the
// per-variant account's Reference), or nil for a blank code. References
// are optional on payments/orders/conversions, so a blank code maps to
// no link rather than an error.
func accountRef(rawCode string) *string {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		return nil
	}
	return &code
}

// ConversionLeg captures one side of a paired conversion row.
type ConversionLeg struct {
	LedgerID string
	Entry    client.LedgerEntry
}

// PairConversionLegs takes two ledger rows sharing one refid and
// returns (source, destination) — source carries the negative-amount
// leg (asset spent), destination the positive-amount leg (asset
// received). Same-sign pairs return ok=false; the orchestrator logs
// and skips.
func PairConversionLegs(a, b ConversionLeg) (source, destination ConversionLeg, ok bool) {
	aNeg, bNeg := IsNegative(a.Entry.Amount), IsNegative(b.Entry.Amount)
	if aNeg == bNeg {
		return ConversionLeg{}, ConversionLeg{}, false
	}
	if aNeg {
		return a, b, true
	}
	return b, a, true
}

// ConversionPairToPSPConversion materialises a PSPConversion from
// the two paired ledger rows. Caller passes the legs already
// ordered (source = negative, destination = positive) via
// PairConversionLegs. Unknown assets surface as a hard error so the
// pair can be retried later rather than silently dropped.
//
// Source/destination account references are each leg's own raw Kraken
// code (the per-variant account Reference); the normalised symbol is
// kept in the asset fields and metadata.
func ConversionPairToPSPConversion(currencies map[string]int, source, destination ConversionLeg) (*models.PSPConversion, error) {
	srcSym := NormalizeAsset(source.Entry.Asset)
	dstSym := NormalizeAsset(destination.Entry.Asset)
	srcPrec, ok := currencies[srcSym]
	if !ok {
		return nil, fmt.Errorf("unknown source asset %q", source.Entry.Asset)
	}
	dstPrec, ok := currencies[dstSym]
	if !ok {
		return nil, fmt.Errorf("unknown destination asset %q", destination.Entry.Asset)
	}

	srcAmt, err := ParseDecimalAmount(AbsAmount(source.Entry.Amount), srcPrec)
	if err != nil {
		return nil, fmt.Errorf("conversion %s source amount: %w", source.Entry.Refid, err)
	}
	dstAmt, err := ParseDecimalAmount(destination.Entry.Amount, dstPrec)
	if err != nil {
		return nil, fmt.Errorf("conversion %s destination amount: %w", destination.Entry.Refid, err)
	}

	raw, err := json.Marshal(struct {
		Source      ConversionLeg `json:"source"`
		Destination ConversionLeg `json:"destination"`
	}{Source: source, Destination: destination})
	if err != nil {
		return nil, fmt.Errorf("conversion %s marshal: %w", source.Entry.Refid, err)
	}

	createdAt := FloatEpochToTime(source.Entry.Time)
	if destination.Entry.Time > source.Entry.Time {
		createdAt = FloatEpochToTime(destination.Entry.Time)
	}

	conv := &models.PSPConversion{
		Reference:                   source.Entry.Refid,
		CreatedAt:                   createdAt,
		SourceAsset:                 FormatAsset(currencies, srcSym),
		DestinationAsset:            FormatAsset(currencies, dstSym),
		SourceAmount:                srcAmt,
		DestinationAmount:           dstAmt,
		Status:                      models.CONVERSION_STATUS_COMPLETED,
		SourceAccountReference:      accountRef(source.Entry.Asset),
		DestinationAccountReference: accountRef(destination.Entry.Asset),
		Metadata: map[string]string{
			MetadataPrefix + "source_ledger_id":         source.LedgerID,
			MetadataPrefix + "destination_ledger_id":    destination.LedgerID,
			MetadataPrefix + "kraken_type":              source.Entry.Type,
			MetadataPrefix + "refid":                    source.Entry.Refid,
			MetadataPrefix + "kraken_source_asset":      source.Entry.Asset,
			MetadataPrefix + "kraken_destination_asset": destination.Entry.Asset,
		},
		Raw: raw,
	}

	// Source-leg fees are in the source asset and don't sum cleanly
	// with the destination side, so they're kept verbatim in metadata;
	// destination-leg fees become the conversion's Fee.
	if !IsZeroAmount(source.Entry.Fee) {
		conv.Metadata[MetadataPrefix+"source_fee"] = source.Entry.Fee
	}
	if !IsZeroAmount(destination.Entry.Fee) {
		if fee, err := ParseDecimalAmount(AbsAmount(destination.Entry.Fee), dstPrec); err == nil && fee.Sign() > 0 {
			conv.Fee = fee
			dstAsset := FormatAsset(currencies, dstSym)
			conv.FeeAsset = &dstAsset
		}
	}
	return conv, nil
}
