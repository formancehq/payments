package mappers

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

func OtherToPSPOther(o client.Other) (models.PSPOther, error) {
	if err := requireRef("other", o.ID); err != nil {
		return models.PSPOther{}, err
	}
	data, err := json.Marshal(o.Data)
	if err != nil {
		return models.PSPOther{}, err
	}
	return models.PSPOther{ID: o.ID, Other: data}, nil
}
