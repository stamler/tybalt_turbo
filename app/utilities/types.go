package utilities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JsonStringSlice []string

// Scan implements the sql.Scanner interface for JsonStringSlice
func (s *JsonStringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported Scan, storing driver.Value type into type *JsonStringSlice")
	}

	// Unmarshal the JSON bytes into the slice
	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface for JsonStringSlice
func (s JsonStringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}
