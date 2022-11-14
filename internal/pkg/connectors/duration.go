package connectors

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type Duration struct {
	time.Duration
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return errors.Wrap(err, "custom Duration UnmarshalJSON: json.Unmarshal")
	}
	switch value := v.(type) {
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return errors.Wrap(err, "custom Duration UnmarshalJSON: time.ParseDuration")
		}

		return nil
	default:
		return fmt.Errorf("custom Duration UnmarshalJSON: invalid type: value:%v, type:%T", value, value)
	}
}
