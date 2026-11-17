package connectorconfig

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadable(t *testing.T) {
	for _, raw := range []string{
		`{"name":"fixture","pollingPeriod":"30m","apiKey":"fake","credentials":{"unknown":"fake"},"provider":"fake"}`,
		`{"name":{"nested":"fake"},"pollingPeriod":["fake"]}`,
		`null`, `[]`, `{"name":null}`, `invalid`,
	} {
		t.Run(raw, func(t *testing.T) {
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(Readable(json.RawMessage(raw)), &fields))
			for key, value := range fields {
				require.Contains(t, []string{"name", "pollingPeriod"}, key)
				var text string
				require.NoError(t, json.Unmarshal(value, &text))
			}
		})
	}
	require.JSONEq(t, `{"name":"fixture","pollingPeriod":"30m"}`, string(Readable(json.RawMessage(`{"name":"fixture","pollingPeriod":"30m","unknown":"fake"}`))))
}

func TestMergePreservesOmittedProviderFields(t *testing.T) {
	stored := json.RawMessage(`{"name":"old","apiKey":"fake-old","credentials":{"secret":"fake-nested","region":"eu"},"account":9007199254740993}`)
	for _, tc := range []struct{ name, patch, want string }{
		{"metadata", `{"name":"new"}`, `{"name":"new","apiKey":"fake-old","credentials":{"secret":"fake-nested","region":"eu"},"account":9007199254740993}`},
		{"nested", `{"credentials":{"region":"us"}}`, `{"name":"old","apiKey":"fake-old","credentials":{"secret":"fake-nested","region":"us"},"account":9007199254740993}`},
		{"replacement", `{"apiKey":"fake-new"}`, `{"name":"old","apiKey":"fake-new","credentials":{"secret":"fake-nested","region":"eu"},"account":9007199254740993}`},
		{"empty", `{"apiKey":""}`, `{"name":"old","apiKey":"","credentials":{"secret":"fake-nested","region":"eu"},"account":9007199254740993}`},
		{"null", `{"apiKey":null}`, `{"name":"old","apiKey":null,"credentials":{"secret":"fake-nested","region":"eu"},"account":9007199254740993}`},
		{"nested null", `{"credentials":{"secret":null}}`, `{"name":"old","apiKey":"fake-old","credentials":{"secret":null,"region":"eu"},"account":9007199254740993}`},
		{"empty object", `{}`, string(stored)},
		{"array replacement", `{"credentials":["fake-new"]}`, `{"name":"old","apiKey":"fake-old","credentials":["fake-new"],"account":9007199254740993}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Merge(stored, json.RawMessage(tc.patch))
			require.NoError(t, err)
			// Decode without converting JSON numbers to float64.
			var got, want any
			decoder := json.NewDecoder(bytes.NewReader(out))
			decoder.UseNumber()
			require.NoError(t, decoder.Decode(&got))
			decoder = json.NewDecoder(bytes.NewBufferString(tc.want))
			decoder.UseNumber()
			require.NoError(t, decoder.Decode(&want))
			require.True(t, reflect.DeepEqual(want, got), "merged configuration differs")
			again, err := Merge(out, json.RawMessage(tc.patch))
			require.NoError(t, err)
			require.Equal(t, out, again)
		})
	}
}

func TestMergeRejectsInvalidUpdates(t *testing.T) {
	for _, patch := range []string{``, `null`, `[]`, `"fake"`, `{`} {
		_, err := Merge(json.RawMessage(`{"apiKey":"fake"}`), json.RawMessage(patch))
		require.Error(t, err)
		require.NotContains(t, err.Error(), "apiKey")
	}
}
