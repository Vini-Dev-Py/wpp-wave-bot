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
func (r *Repository) Save(ctx context.Context, companyID, msgID, sender, receiver, msgType, content, payload, status string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO messages (company_id, msg_id, sender, receiver, type, content, payload, status)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		companyID, msgID, sender, receiver, msgType, content, payload, status,
	)
	return err
}

// UpdateStatus updates the status of a message
func (r *Repository) UpdateStatus(ctx context.Context, companyID, msgID, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE messages SET status=$1, updated_at=now() WHERE company_id=$2 AND msg_id=$3`,
		status, companyID, msgID,
	)
	return err
}
