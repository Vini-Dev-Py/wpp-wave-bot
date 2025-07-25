package groups

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles group persistence.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Upsert inserts or updates a group record.
func (r *Repository) Upsert(ctx context.Context, companyID, jid, name string) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO groups (jid, company_id, name)
        VALUES ($1, $2, $3)
        ON CONFLICT (jid) DO UPDATE SET name = EXCLUDED.name, updated_at = now()
    `, jid, companyID, name)
	return err
}
