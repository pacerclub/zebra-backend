package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// InitDB initializes the database connection pool
func InitDB() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/zebra?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("error parsing database URL: %v", err)
	}

	// Set reasonable pool limits
	config.MaxConns = 10
	config.MinConns = 2

	Pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %v", err)
	}

	// Test the connection
	if err := Pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %v", err)
	}

	return nil
}

// GetDB returns the database pool
func GetDB() *pgxpool.Pool {
	return Pool
}

// CloseDB closes the database connection pool
func CloseDB() {
	if Pool != nil {
		Pool.Close()
	}
}
