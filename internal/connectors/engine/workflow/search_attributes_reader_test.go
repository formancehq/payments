package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

func TestPaymentScheduleIDReaderLegacyAndTypedPayloads(t *testing.T) {
	for _, value := range []string{"", "é栈", "quote\"\\\n"} {
		for _, typed := range []bool{false, true} {
			payload, err := converter.GetDefaultDataConverter().ToPayload(value)
			require.NoError(t, err)
			if typed {
				payload.Metadata["type"] = []byte("Keyword")
			}
			// Deserialize the persisted legacy Info shape, preserving the reader's
			// contract without adding deprecated field initializers to the source.
			data, err := json.Marshal(map[string]interface{}{"SearchAttributes": &commonpb.SearchAttributes{IndexedFields: map[string]*commonpb.Payload{SearchAttributeScheduleID: payload}}})
			require.NoError(t, err)
			var info workflow.Info
			require.NoError(t, json.Unmarshal(data, &info))
			got, err := getPaymentScheduleID(&info)
			require.NoError(t, err)
			require.Equal(t, value, got)
		}
	}
	for _, raw := range []string{`{}`, `{"SearchAttributes":{}}`, `{"SearchAttributes":{"indexed_fields":{"PaymentScheduleID":null}}}`} {
		var info workflow.Info
		require.NoError(t, json.Unmarshal([]byte(raw), &info))
		_, err := getPaymentScheduleID(&info)
		require.ErrorIs(t, err, errNotFromSchedule)
	}
}
