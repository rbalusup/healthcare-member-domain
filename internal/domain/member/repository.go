package member

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the persistence contract for Member aggregates.
// Implementations live in internal/infrastructure/postgres.
type Repository interface {
	// Create persists a new member. Returns ErrMemberAlreadyExists if email is taken.
	Create(ctx context.Context, member *Member) error

	// GetByID retrieves a member by their unique ID. Returns ErrMemberNotFound if absent.
	GetByID(ctx context.Context, id uuid.UUID) (*Member, error)

	// GetByEmail retrieves a member by email address. Returns ErrMemberNotFound if absent.
	GetByEmail(ctx context.Context, email string) (*Member, error)

	// Update persists changes to an existing member. Returns ErrMemberNotFound if absent.
	Update(ctx context.Context, member *Member) error

	// Delete removes a member record. Returns ErrMemberNotFound if absent.
	Delete(ctx context.Context, id uuid.UUID) error

	// List returns a paginated, filtered list of members and the total count.
	List(ctx context.Context, filter Filter, pagination Pagination) ([]*Member, int64, error)
}
