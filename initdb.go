package pgparty

import (
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib" // Postgresql driver
	"github.com/jmoiron/sqlx"
)

const (
	DriverPostgreSQL = "pgx"
)

type DatabaseDSN struct {
	Postgres        string        `json:"postgres" yaml:"postgres"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
}

func InitDB(c DatabaseDSN) (*sqlx.DB, error) {
	var driverName, connectionString string

	switch {
	case strings.TrimSpace(c.Postgres) != "":
		driverName = DriverPostgreSQL
		connectionString = c.Postgres
	default:
		return nil, fmt.Errorf("DB not initialized (driver is not specified)")
	}

	db, err := sqlx.Connect(driverName, connectionString)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %w", driverName, err)
	}
	if c.MaxIdleConns > 0 {
		db.SetMaxIdleConns(c.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(25)
	}
	if c.MaxOpenConns > 0 {
		db.SetMaxOpenConns(c.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(25)
	}
	if c.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(c.ConnMaxLifetime)
	} else {
		db.SetConnMaxLifetime(5 * time.Minute)
	}

	go regularPing(db)

	return db, nil
}

func regularPing(db *sqlx.DB) {
	for {
		if err := db.Ping(); err != nil {
			log.Printf("can't ping db driver: %s", err)
		}
		time.Sleep(time.Minute)
	}
}
