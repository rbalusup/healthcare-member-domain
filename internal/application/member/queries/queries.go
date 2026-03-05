package queries

import (
	"github.com/google/uuid"
	"github.com/healthcare/member-service/internal/domain/member"
)

// GetMemberQuery retrieves a single member by ID.
type GetMemberQuery struct {
	MemberID uuid.UUID
}

// GetMemberByEmailQuery retrieves a single member by email address.
type GetMemberByEmailQuery struct {
	Email string
}

// ListMembersQuery retrieves a paginated, filtered list of members.
type ListMembersQuery struct {
	Filter     member.Filter
	Pagination member.Pagination
}

// GetMemberRiskProfileQuery retrieves a member's current risk profile.
type GetMemberRiskProfileQuery struct {
	MemberID uuid.UUID
}

// MemberResult is the read model returned by member queries.
// It is intentionally separate from the domain entity to allow
// independent evolution of the read and write models.
type MemberResult struct {
	MemberID       string
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	DateOfBirth    string // ISO 8601
	Gender         string
	Street         string
	City           string
	State          string
	ZipCode        string
	Country        string
	PlanID         string
	EnrollmentDate string // ISO 8601
	Status         string
	RiskScore      float64
	CareLevel      string
	CreatedAt      string // ISO 8601
	UpdatedAt      string // ISO 8601
}

// ListMembersResult wraps a paginated member list.
type ListMembersResult struct {
	Members    []*MemberResult
	TotalCount int64
	Page       int
	PageSize   int
}

// RiskProfileResult is the read model for a member's risk assessment.
type RiskProfileResult struct {
	MemberID  string
	RiskScore float64
	CareLevel string
	UpdatedAt string // ISO 8601
}
