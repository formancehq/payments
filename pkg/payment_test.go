package payments

import (
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
)

func TestPayment(t *testing.T) {
	now := time.Now()
	p := Payment{
		Identifier: Identifier{
			Provider: "testing",
			Referenced: Referenced{
				Reference: uuid.New(),
				Type:      TypePayIn,
			},
		},
		Data: Data{
			Status:        "success",
			InitialAmount: 100,
			Scheme:        SchemeSepa,
			Asset:         "USD/2",
			CreatedAt:     now,
		},
		Adjustments: []Adjustment{
			{
				Status: "success",
				Amount: 10,
				Date:   now.Add(time.Minute),
			},
			{
				Status: "success",
				Amount: 100,
				Date:   now,
			},
		},
	}
	cp := p.Computed()
	require.EqualValues(t, 110, cp.Amount)
}
