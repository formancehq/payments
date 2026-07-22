package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression test: the v1 get-by-ID payload (GET /v1/accounts/{id}) carries
// the owning profile as "profile", while the v2 list (GET /v2/accounts)
// carries it as "profileId". The struct tag only matched v2, so
// GetRecipientAccount decoded Profile as 0 and the balance enrichment of
// transfers in GetTransfers was silently disabled. Both shapes must decode.
func TestRecipientAccountUnmarshalProfileShapes(t *testing.T) {
	t.Parallel()

	t.Run("v1 get-by-ID shape (profile)", func(t *testing.T) {
		t.Parallel()

		var ra RecipientAccount
		require.NoError(t, json.Unmarshal([]byte(`{
			"id": 702499330,
			"profile": 30565298,
			"currency": "EUR",
			"accountHolderName": "Formance Contract Test"
		}`), &ra))
		require.Equal(t, uint64(702499330), ra.ID)
		require.Equal(t, uint64(30565298), ra.Profile)
		require.Equal(t, "EUR", ra.Currency)
	})

	t.Run("v2 list shape (profileId)", func(t *testing.T) {
		t.Parallel()

		var ra RecipientAccount
		require.NoError(t, json.Unmarshal([]byte(`{
			"id": 702499330,
			"profileId": 30565298,
			"currency": "EUR",
			"name": {"fullName": "Formance Contract Test"}
		}`), &ra))
		require.Equal(t, uint64(702499330), ra.ID)
		require.Equal(t, uint64(30565298), ra.Profile)
		require.Equal(t, "Formance Contract Test", ra.Name.FullName)
	})
}
