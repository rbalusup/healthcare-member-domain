package member

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MemberStatus represents the enrollment/activity status of a member.
type MemberStatus string

const (
	MemberStatusActive    MemberStatus = "ACTIVE"
	MemberStatusInactive  MemberStatus = "INACTIVE"
	MemberStatusSuspended MemberStatus = "SUSPENDED"
)

// CareLevel represents the intensity of care required for a member.
type CareLevel string

const (
	CareLevelStandard CareLevel = "STANDARD"
	CareLevelEnhanced CareLevel = "ENHANCED"
	CareLevelComplex  CareLevel = "COMPLEX"
)

// Gender represents a member's gender identity.
type Gender string

const (
	GenderMale        Gender = "MALE"
	GenderFemale      Gender = "FEMALE"
	GenderNonBinary   Gender = "NON_BINARY"
	GenderUnspecified Gender = "UNSPECIFIED"
)

// Address is a value object representing a physical mailing address.
type Address struct {
	Street  string
	City    string
	State   string
	ZipCode string
	Country string
}

// ContactInfo holds a member's communication details.
type ContactInfo struct {
	Email   string
	Phone   string
	Address Address
}

// EnrollmentInfo holds a member's insurance/plan enrollment details.
type EnrollmentInfo struct {
	PlanID         string
	EnrollmentDate time.Time
	TerminationDate *time.Time
}

// RiskProfile holds clinical risk stratification data for a member.
type RiskProfile struct {
	RiskScore  float64
	CareLevel  CareLevel
	UpdatedAt  time.Time
}

// Member is the core aggregate root for the Member domain.
// It represents a patient/member receiving care through the platform.
type Member struct {
	MemberID    uuid.UUID
	FirstName   string
	LastName    string
	DateOfBirth time.Time
	Gender      Gender
	Contact     ContactInfo
	Enrollment  EnrollmentInfo
	Risk        RiskProfile
	Status      MemberStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewMember constructs a validated Member entity.
func NewMember(
	firstName, lastName, email, phone string,
	dateOfBirth time.Time,
	gender Gender,
	address Address,
	planID string,
	enrollmentDate time.Time,
) (*Member, error) {
	if strings.TrimSpace(firstName) == "" {
		return nil, errors.New("first name is required")
	}
	if strings.TrimSpace(lastName) == "" {
		return nil, errors.New("last name is required")
	}
	if strings.TrimSpace(email) == "" {
		return nil, errors.New("email is required")
	}
	if strings.TrimSpace(planID) == "" {
		return nil, errors.New("plan ID is required")
	}
	if dateOfBirth.IsZero() {
		return nil, errors.New("date of birth is required")
	}
	if enrollmentDate.IsZero() {
		return nil, errors.New("enrollment date is required")
	}

	now := time.Now().UTC()
	return &Member{
		MemberID:  uuid.New(),
		FirstName: strings.TrimSpace(firstName),
		LastName:  strings.TrimSpace(lastName),
		DateOfBirth: dateOfBirth,
		Gender:    gender,
		Contact: ContactInfo{
			Email: strings.ToLower(strings.TrimSpace(email)),
			Phone: strings.TrimSpace(phone),
			Address: address,
		},
		Enrollment: EnrollmentInfo{
			PlanID:         planID,
			EnrollmentDate: enrollmentDate,
		},
		Risk: RiskProfile{
			RiskScore: 0.0,
			CareLevel: CareLevelStandard,
			UpdatedAt: now,
		},
		Status:    MemberStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// FullName returns the member's display name.
func (m *Member) FullName() string {
	return m.FirstName + " " + m.LastName
}

// IsActive returns true if the member is currently active.
func (m *Member) IsActive() bool {
	return m.Status == MemberStatusActive
}

// UpdateRiskProfile updates the member's clinical risk score and care level.
func (m *Member) UpdateRiskProfile(score float64, level CareLevel) error {
	if score < 0 || score > 100 {
		return errors.New("risk score must be between 0 and 100")
	}
	m.Risk.RiskScore = score
	m.Risk.CareLevel = level
	m.Risk.UpdatedAt = time.Now().UTC()
	m.UpdatedAt = time.Now().UTC()
	return nil
}

// Enroll updates the member's enrollment information and activates the member.
func (m *Member) Enroll(planID string, enrollmentDate time.Time) error {
	if strings.TrimSpace(planID) == "" {
		return errors.New("plan ID is required for enrollment")
	}
	m.Enrollment.PlanID = planID
	m.Enrollment.EnrollmentDate = enrollmentDate
	m.Enrollment.TerminationDate = nil
	m.Status = MemberStatusActive
	m.UpdatedAt = time.Now().UTC()
	return nil
}

// Terminate marks a member's enrollment as terminated.
func (m *Member) Terminate(terminationDate time.Time) {
	m.Enrollment.TerminationDate = &terminationDate
	m.Status = MemberStatusInactive
	m.UpdatedAt = time.Now().UTC()
}

// Filter holds the criteria for querying members.
type Filter struct {
	Status    *MemberStatus
	CareLevel *CareLevel
	PlanID    string
	SearchTerm string // matches first/last name or email
}

// Pagination controls page-based result sets.
type Pagination struct {
	Page     int
	PageSize int
}

// DefaultPagination returns sensible defaults for list queries.
func DefaultPagination() Pagination {
	return Pagination{Page: 1, PageSize: 20}
}
