package contacts

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides helpers to persist contacts.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Repository instance.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Upsert inserts or updates a contact.
func (r *Repository) Upsert(ctx context.Context, companyID, jid, name, phone, avatarURL string) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO contacts (jid, company_id, name, phone, avatar_url, last_seen)
        VALUES ($1, $2, $3, $4, $5, now())
        ON CONFLICT (jid) DO UPDATE
            SET name = COALESCE(EXCLUDED.name, contacts.name),
                phone = COALESCE(EXCLUDED.phone, contacts.phone),
                avatar_url = COALESCE(EXCLUDED.avatar_url, contacts.avatar_url),
                last_seen = now(),
                updated_at = now()
    `, jid, companyID, name, phone, avatarURL)
	return err
}
