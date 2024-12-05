package migrations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type v3eventSent struct {
	bun.BaseModel `bun:"table:events_sent"`

	ID          models.EventID      `bun:"id,pk,type:character varying,notnull"`
	ConnectorID *models.ConnectorID `bun:"connector_id,type:character varying"`
	SentAt      time.Time           `bun:"sent_at,type:timestamp without time zone,notnull"`
}

func isTableExisting(ctx context.Context, db bun.IDB, schema, table string) (bool, error) {
	var count int
	err := db.NewRaw(fmt.Sprintf(`SELECT count(*)
		FROM information_schema.tables 
		WHERE table_schema ='%s' AND table_name ='%s'
	`, schema, table)).Scan(ctx, &count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

type v2Duration struct {
	time.Duration `json:"duration"`
}

func (d *v2Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

func (d *v2Duration) UnmarshalJSON(b []byte) error {
	var rawValue any

	if err := json.Unmarshal(b, &rawValue); err != nil {
		return fmt.Errorf("custom Duration UnmarshalJSON: %w", err)
	}

	switch value := rawValue.(type) {
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("custom Duration UnmarshalJSON: time.ParseDuration: %w", err)
		}

		return nil
	case map[string]interface{}:
		switch val := value["duration"].(type) {
		case float64:
			d.Duration = time.Duration(int64(val))

			return nil
		default:
			return fmt.Errorf("custom Duration UnmarshalJSON from map: invalid type: value:%v, type:%T", val, val)
		}
	default:
		return fmt.Errorf("custom Duration UnmarshalJSON: invalid type: value:%v, type:%T", value, value)
	}
}

func paginateWithOffset[FILTERS any, RETURN any](ctx context.Context, db bun.IDB,
	q *bunpaginate.OffsetPaginatedQuery[FILTERS], builders ...func(query *bun.SelectQuery) *bun.SelectQuery) (*bunpaginate.Cursor[RETURN], error) {
	query := db.NewSelect()
	return bunpaginate.UsingOffset[FILTERS, RETURN](ctx, query, *q, builders...)
}
