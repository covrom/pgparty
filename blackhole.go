package pgparty

import (
	"database/sql/driver"
)

type BlackHole struct {}

func (b *BlackHole) Scan(value interface{}) error {
	return nil
}

func (b BlackHole) Value() (driver.Value, error) {
	return nil, nil
}
