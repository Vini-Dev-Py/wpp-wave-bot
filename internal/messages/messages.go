package messages

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides helpers to persist messages.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new Repository instance.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Save inserts a message record.
func (r *Repository) Save(ctx context.Context, companyID, msgID, sender, receiver, msgType, content string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO messages (company_id, msg_id, sender, receiver, type, content)
         VALUES ($1, $2, $3, $4, $5, $6)`,
		companyID, msgID, sender, receiver, msgType, content,
	)
	return err
}
