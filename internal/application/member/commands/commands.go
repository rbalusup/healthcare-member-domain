package commands

import (
	"time"

	"github.com/google/uuid"
	"github.com/rbalusup/healthcare-member-domain/internal/domain/member"
)

// CreateMemberCommand carries input for creating a new member.
type CreateMemberCommand struct {
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	DateOfBirth    time.Time
	Gender         member.Gender
	Street         string
	City           string
	State          string
	ZipCode        string
	Country        string
	PlanID         string
	EnrollmentDate time.Time
}

// UpdateMemberCommand carries input for updating an existing member's profile.
type UpdateMemberCommand struct {
	MemberID  uuid.UUID
	FirstName string
	LastName  string
	Phone     string
	Street    string
	City      string
	State     string
	ZipCode   string
	Country   string
}

// EnrollMemberCommand carries input for enrolling a member in a plan.
type EnrollMemberCommand struct {
	MemberID       uuid.UUID
	PlanID         string
	EnrollmentDate time.Time
}

// TerminateMemberCommand carries input for terminating a member's enrollment.
type TerminateMemberCommand struct {
	MemberID        uuid.UUID
	TerminationDate time.Time
}

// UpdateRiskProfileCommand carries input for updating a member's risk assessment.
type UpdateRiskProfileCommand struct {
	MemberID  uuid.UUID
	RiskScore float64
	CareLevel member.CareLevel
}

// DeleteMemberCommand carries input for deleting a member record.
type DeleteMemberCommand struct {
	MemberID uuid.UUID
}
