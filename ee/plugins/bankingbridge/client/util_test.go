package client_test

import (
	"net/url"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bankingbridge/client"
)

func TestRawQuery(t *testing.T) {
	tests := []struct {
		name       string
		pageSize   int
		cursor     string
		importedAt string
		expected   string
	}{
		{
			name:       "With cursor",
			pageSize:   10,
			cursor:     "abc123",
			importedAt: "",
			expected:   "cursor=abc123&pageSize=10",
		},
		{
			name:       "With cursor and importedAt",
			pageSize:   10,
			cursor:     "abc123",
			importedAt: "2023-01-01T00:00:00Z",
			expected:   "cursor=abc123&pageSize=10",
		},
		{
			name:       "Without cursor, with importedAt",
			pageSize:   20,
			cursor:     "",
			importedAt: "2023-01-01T00:00:00Z",
			expected:   "pageSize=20&query=%7B%22%24gt%22%3A%7B%22importedAt%22%3A%222023-01-01T00%3A00%3A00Z%22%7D%7D",
		},
		{
			name:       "Without cursor and importedAt",
			pageSize:   5,
			cursor:     "",
			importedAt: "",
			expected:   "pageSize=5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := url.Values{}
			result := client.RawQuery(v, tt.pageSize, tt.cursor, tt.importedAt)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
