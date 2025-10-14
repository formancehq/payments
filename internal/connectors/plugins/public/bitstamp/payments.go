package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	bsclient "github.com/formancehq/payments/internal/connectors/plugins/public/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

// Persisted pagination state for Bitstamp
type paymentsState struct {
	LastOffset       int       `json:"lastOffset"`
	LastCreationDate time.Time `json:"lastCreationDate"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	// ---- load previous state ----
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	// Bitstamp "user_transactions" is account-global, so we don't require FromPayload.
	newState := paymentsState{
		LastOffset:       oldState.LastOffset,
		LastCreationDate: oldState.LastCreationDate,
	}

	var (
		out      []models.PSPPayment
		needMore bool
		hasMore  bool
	)

	offset := oldState.LastOffset
	limit := req.PageSize
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 { // API upper bound
		limit = 1000
	}

	for {
		params := bsclient.TransactionsParams{
			Offset: offset,
			Limit:  limit,
			Sort:   "desc",
		}
		// Optional optimization: keep a moving window using the last seen creation time.
		if !oldState.LastCreationDate.IsZero() {
			params.SinceTimestamp = oldState.LastCreationDate.Unix()
		}

		// Your GetTransactions returns []client.Transaction, error
		rows, err := p.client.GetTransactions(ctx, params)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		out, err = fillPaymentsBitstamp(rows, out)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(out, rows, req.PageSize)
		if !needMore || !hasMore {
			break
		}

		offset += limit
	}

	// Update state from the *last* emitted payment (descending order)
	if len(out) > 0 {
		last := out[len(out)-1]
		// If we saw the same newest timestamp, advance offset; else reset to 0 for next pass.
		if oldState.LastCreationDate.Equal(last.CreatedAt) {
			newState.LastOffset = offset + limit
		} else {
			newState.LastOffset = 0
		}
		newState.LastCreationDate = last.CreatedAt
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: out,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

// ---- Transaction → PSPPayment ----------------------------------------------

func fillPaymentsBitstamp(
	rows []bsclient.Transaction,
	out []models.PSPPayment,
) ([]models.PSPPayment, error) {
	for _, tx := range rows {
		pmt, err := transactionToPayment(tx)
		if err != nil {
			return nil, err
		}
		if pmt != nil {
			out = append(out, *pmt)
		}
	}
	return out, nil
}

func transactionToPayment(tx bsclient.Transaction) (*models.PSPPayment, error) {
	// Raw
	raw, err := json.Marshal(&tx)
	if err != nil {
		return nil, err
	}

	// Parse time (Bitstamp formats with microseconds)
	createdAt, err := parseBitstampTime(tx.Datetime)
	if err != nil {
		return nil, fmt.Errorf("parse datetime: %w", err)
	}

	// Collect non-zero legs to infer payment type and pick primary leg.
	legs := map[string]string{
		"EUR":  string(tx.EUR),
		"USD":  string(tx.USD),
		"USDC": string(tx.USDC),
		"BTC":  string(tx.BTC),
	}

	type leg struct {
		Code string
		Val  string
		Neg  bool
	}
	nonZero := make([]leg, 0, 2)
	for code, s := range legs {
		if isZeroNumStr(s) {
			continue
		}
		ns := strings.TrimSpace(s)
		nonZero = append(nonZero, leg{
			Code: code,
			Val:  ns,
			Neg:  strings.HasPrefix(ns, "-"),
		})
	}

	if len(nonZero) == 0 {
		// nothing meaningful
		return nil, nil
	}

	// Default heuristic:
	// - trade (two legs): TRANSFER, prefer negative leg as primary
	// - single leg: PAYIN if positive, PAYOUT if negative
	var pType models.PaymentType = models.PAYMENT_TYPE_OTHER
	primary := nonZero[0]

	if len(nonZero) > 1 {
		// For transfers, prefer the negative leg as primary (money leaving)
		// If no negative leg, use the first leg
		for _, l := range nonZero {
			if l.Neg {
				primary = l
				break
			}
		}
		pType = models.PAYMENT_TYPE_TRANSFER
	} else {
		if primary.Neg {
			pType = models.PAYMENT_TYPE_PAYOUT
		} else {
			pType = models.PAYMENT_TYPE_PAYIN
		}
	}

	// Amount in minor units from primary leg (keep sign)
	decimals, ok := supportedCurrenciesWithDecimal[primary.Code]
	if !ok {
		return nil, fmt.Errorf("unsupported currency %s", primary.Code)
	}
	amt, err := decimalToMinor(strings.TrimPrefix(strings.TrimSpace(primary.Val), "+"), int(decimals))
	if err != nil {
		return nil, fmt.Errorf("parse amount %s %s: %w", primary.Code, primary.Val, err)
	}

	payment := models.PSPPayment{
		Reference: string(tx.ID),
		CreatedAt: createdAt,
		Type:      pType,
		Amount:    amt,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, primary.Code),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED, // user_transactions are settled history
		Raw:       raw,
	}

	return &payment, nil
}

// ---- Helpers ----------------------------------------------------------------

func isZeroNumStr(s string) bool {
	ns := strings.TrimSpace(s)
	if ns == "" {
		return true
	}
	r := new(big.Rat)
	if _, ok := r.SetString(ns); !ok {
		// if unparsable, treat as non-zero to avoid dropping records silently
		return false
	}
	return r.Sign() == 0
}

// decimalToMinor converts a decimal string like "-5.81077" into minor units big.Int
// given the number of decimals for the currency (e.g., 2 for EUR, 6 for USDC, 8 for BTC).
func decimalToMinor(s string, decimals int) (*big.Int, error) {
	sign := 1
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = strings.TrimPrefix(s, "-")
	}

	intPart, fracPart := s, ""
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		intPart, fracPart = s[:dot], s[dot+1:]
	}

	// normalize fractional part to 'decimals' digits
	switch {
	case len(fracPart) > decimals:
		fracPart = fracPart[:decimals]
	case len(fracPart) < decimals:
		fracPart += strings.Repeat("0", decimals-len(fracPart))
	}

	// strip leading zeros from int part
	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}

	numStr := intPart + fracPart
	if allZero(numStr) {
		return big.NewInt(0), nil
	}

	bi := new(big.Int)
	if _, ok := bi.SetString(numStr, 10); !ok {
		return nil, fmt.Errorf("invalid number %q", numStr)
	}
	if sign < 0 {
		bi.Neg(bi)
	}
	return bi, nil
}

func allZero(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return false
		}
	}
	return true
}

// parseBitstampTime parses Bitstamp datetime format: "2025-09-25 14:42:59.894846"
func parseBitstampTime(datetime string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05.000000", datetime)
}
