package mappers

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDecimalAmount(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		precision int
		want      *big.Int
		wantErr   bool
	}{
		{
			name:      "empty string",
			value:     "",
			precision: 2,
			wantErr:   true,
		},
		{
			name:      "integer no decimal",
			value:     "100",
			precision: 2,
			want:      big.NewInt(10000),
		},
		{
			name:      "exact precision",
			value:     "1.23",
			precision: 2,
			want:      big.NewInt(123),
		},
		{
			name:      "fewer decimals than precision",
			value:     "1.2",
			precision: 2,
			want:      big.NewInt(120),
		},
		{
			name:      "trailing zeros beyond precision truncated",
			value:     "1.23000",
			precision: 2,
			want:      big.NewInt(123),
		},
		{
			name:      "single trailing zero beyond precision",
			value:     "0.500",
			precision: 2,
			want:      big.NewInt(50),
		},
		{
			name:      "non-zero digits beyond precision errors",
			value:     "1.235",
			precision: 2,
			wantErr:   true,
		},
		{
			name:      "precision zero with decimal errors",
			value:     "1.0",
			precision: 0,
			wantErr:   true,
		},
		{
			name:      "zero amount",
			value:     "0.00",
			precision: 2,
			want:      big.NewInt(0),
		},
		{
			name:      "large amount trailing zeros",
			value:     "12345.678900",
			precision: 4,
			want:      big.NewInt(123456789),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDecimalAmount(tc.value, tc.precision)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
