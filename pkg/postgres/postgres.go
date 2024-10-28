// postgres/postgres.go
package postgres

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct {
	*sqlx.DB
	cfg Config
}

func NewDB(config Config) *DB {
	return &DB{cfg: config}
}

func (d *DB) Start(ctx context.Context) error {
	connectionURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.cfg.DBHost,
		d.cfg.DBPort,
		d.cfg.DBUser,
		d.cfg.DBPassword,
		d.cfg.DBName,
		d.cfg.SSLMode,
	)

	db, err := sqlx.Open("postgres", connectionURL)
	if err != nil {
		return err
	}

	if err := db.PingContext(ctx); err != nil {
		return err
	}

	d.DB = db
	return nil
}

func (d *DB) Stop(ctx context.Context) error {
	return d.DB.Close()
}
