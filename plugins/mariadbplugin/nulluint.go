package mariadbplugin

import (
	"database/sql/driver"
	"fmt"
)

//NullUint64 is intended to support null values from mysql driver
type NullUint64 struct {
	Uint64 uint64
	Valid  bool
}

// Scan implements the Scanner interface.
func (n *NullUint64) Scan(value interface{}) error {
	if value == nil {
		n.Uint64, n.Valid = 0, false
		return nil
	}

	switch v := value.(type) {
	case uint64:
		n.Uint64, n.Valid = v, true
		return nil
	case int64:
		n.Uint64, n.Valid = uint64(v), true
		return nil
	}

	n.Valid = false
	return fmt.Errorf("Can't convert %T to uint64", value)
}

// Value implements the driver Valuer interface.
func (n NullUint64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return int64(n.Uint64), nil
}
