package member

import (
	"context"

	"github.com/google/uuid"
)

// DomainService handles complex business logic that spans multiple Member
// aggregates or requires external domain knowledge (e.g., risk stratification rules).
type DomainService interface {
	// CalculateRiskScore computes a clinical risk score for the member using
	// clinical data sourced from other microservices (claims, vitals, diagnoses).
	CalculateRiskScore(ctx context.Context, memberID uuid.UUID) (float64, CareLevel, error)

	// ValidateEnrollmentEligibility checks whether a member is eligible
	// to enroll in the given plan based on plan rules and member demographics.
	ValidateEnrollmentEligibility(ctx context.Context, member *Member, planID string) error

	// NotifyStatusChange triggers downstream notifications (care team, scheduler, etc.)
	// when a member's status transitions.
	NotifyStatusChange(ctx context.Context, member *Member, previousStatus MemberStatus) error
}
