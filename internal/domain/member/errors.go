package member

import "errors"

// Sentinel errors for the Member domain.
// Handlers map these to appropriate gRPC/HTTP status codes.
var (
	// ErrMemberNotFound is returned when a member lookup finds no matching record.
	ErrMemberNotFound = errors.New("member not found")

	// ErrMemberAlreadyExists is returned when creating a member with a duplicate email.
	ErrMemberAlreadyExists = errors.New("member already exists")

	// ErrInvalidMemberData is returned when member input fails validation.
	ErrInvalidMemberData = errors.New("invalid member data")

	// ErrMemberEnrollmentFailed is returned when enrollment processing fails.
	ErrMemberEnrollmentFailed = errors.New("member enrollment failed")

	// ErrMemberNotActive is returned when an operation requires an active member.
	ErrMemberNotActive = errors.New("member is not active")

	// ErrInvalidRiskScore is returned when a risk score is outside the valid range.
	ErrInvalidRiskScore = errors.New("risk score must be between 0 and 100")
)
