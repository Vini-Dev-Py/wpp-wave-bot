package seeders

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Seed can be used to seed initial data
func Seed(pool *pgxpool.Pool) error {
	// No-op for now
	return pool.Ping(context.Background())
}
