package checkout

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validateTransferPayoutRequests(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) generateIdempotencyKey(values ...string) string {
	joined := strings.Join(values, "-")
	hash := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(hash[:])
}