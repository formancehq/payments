package storage

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type capability struct {
	bun.BaseModel `bun:"table:capabilities"`

	// Mandatory fields
	ConnectorID models.ConnectorID `bun:"connector_id,pk,type:character varying,notnull"`
	Capability  models.Capability  `bun:"capability,pk,type:text,notnull"`
}

func (s *store) CapabilitiesUpsert(ctx context.Context, connectorID models.ConnectorID, capabilities []models.Capability) error {
	toInsert := make([]capability, 0, len(capabilities))
	for _, c := range capabilities {
		toInsert = append(toInsert, capability{
			ConnectorID: connectorID,
			Capability:  c,
		})
	}

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (connector_id, capability) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("upsert capabilities", err)
	}

	return nil
}

func (s *store) CapabilitiesGet(ctx context.Context, connectorID models.ConnectorID) ([]models.Capability, error) {
	var capabilities []capability
	err := s.db.NewSelect().
		Model(&capabilities).
		Where("connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, e("get capabilities", err)
	}

	res := make([]models.Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		res = append(res, capability.Capability)
	}

	return res, nil
}
