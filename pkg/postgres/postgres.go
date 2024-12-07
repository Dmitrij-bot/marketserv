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

	connectionURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.DBHost,
		config.DBPort,
		config.DBUser,
		config.DBPassword,
		config.DBName,
		config.SSLMode,
	)

	db, err := sqlx.Open("postgres", connectionURL)
	if err != nil {
		fmt.Printf("1")
	}

	return &DB{
		cfg: config,
		DB:  db,
	}
}

func (d *DB) Start(ctx context.Context) error {

	if err := d.DB.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (d *DB) Stop(ctx context.Context) error {
	return d.DB.Close()
}
