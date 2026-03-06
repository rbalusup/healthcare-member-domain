package member_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/healthcare/member-service/internal/domain/member"
)

func validMemberArgs() (string, string, string, string, time.Time, member.Gender, member.Address, string, time.Time) {
	return "Jane", "Doe", "jane.doe@example.com", "555-0100",
		time.Date(1985, 6, 15, 0, 0, 0, 0, time.UTC),
		member.GenderFemale,
		member.Address{Street: "123 Main St", City: "Springfield", State: "IL", ZipCode: "62701", Country: "US"},
		"PPO-2024-001",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestNewMember_Valid(t *testing.T) {
	first, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	m, err := member.NewMember(first, last, email, phone, dob, gender, addr, planID, enrollDate)

	require.NoError(t, err)
	assert.Equal(t, "Jane", m.FirstName)
	assert.Equal(t, "Doe", m.LastName)
	assert.Equal(t, "jane.doe@example.com", m.Contact.Email)
	assert.Equal(t, "PPO-2024-001", m.Enrollment.PlanID)
	assert.Equal(t, member.MemberStatusActive, m.Status)
	assert.Equal(t, member.CareLevelStandard, m.Risk.CareLevel)
	assert.NotEqual(t, uuid.Nil, m.MemberID)
	assert.False(t, m.CreatedAt.IsZero())
}

func TestNewMember_InvalidEmail(t *testing.T) {
	first, last, _, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	_, err := member.NewMember(first, last, "", phone, dob, gender, addr, planID, enrollDate)
	assert.ErrorContains(t, err, "email")
}

func TestNewMember_InvalidPlanID(t *testing.T) {
	first, last, email, phone, dob, gender, addr, _, enrollDate := validMemberArgs()
	_, err := member.NewMember(first, last, email, phone, dob, gender, addr, "", enrollDate)
	assert.ErrorContains(t, err, "plan")
}

func TestNewMember_InvalidFirstName(t *testing.T) {
	_, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	_, err := member.NewMember("", last, email, phone, dob, gender, addr, planID, enrollDate)
	assert.ErrorContains(t, err, "first name")
}

func TestMember_Enroll(t *testing.T) {
	first, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	m, _ := member.NewMember(first, last, email, phone, dob, gender, addr, planID, enrollDate)

	// Terminate first so we can verify Enroll clears it.
	m.Terminate(time.Now())
	assert.Equal(t, member.MemberStatusInactive, m.Status)

	newEnroll := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	err := m.Enroll("HMO-2025-001", newEnroll)
	require.NoError(t, err)
	assert.Equal(t, "HMO-2025-001", m.Enrollment.PlanID)
	assert.Equal(t, member.MemberStatusActive, m.Status)
	assert.Nil(t, m.Enrollment.TerminationDate)
}

func TestMember_Enroll_EmptyPlanID(t *testing.T) {
	first, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	m, _ := member.NewMember(first, last, email, phone, dob, gender, addr, planID, enrollDate)

	err := m.Enroll("", time.Now())
	assert.Error(t, err)
}

func TestMember_Terminate(t *testing.T) {
	first, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	m, _ := member.NewMember(first, last, email, phone, dob, gender, addr, planID, enrollDate)

	termDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	m.Terminate(termDate)

	assert.Equal(t, member.MemberStatusInactive, m.Status)
	require.NotNil(t, m.Enrollment.TerminationDate)
	assert.Equal(t, termDate, *m.Enrollment.TerminationDate)
}

func TestMember_UpdateRiskProfile_Valid(t *testing.T) {
	first, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	m, _ := member.NewMember(first, last, email, phone, dob, gender, addr, planID, enrollDate)

	err := m.UpdateRiskProfile(75.5, member.CareLevelEnhanced)
	require.NoError(t, err)
	assert.InDelta(t, 75.5, m.Risk.RiskScore, 0.001)
	assert.Equal(t, member.CareLevelEnhanced, m.Risk.CareLevel)
}

func TestMember_UpdateRiskProfile_OutOfRange(t *testing.T) {
	first, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	m, _ := member.NewMember(first, last, email, phone, dob, gender, addr, planID, enrollDate)

	err := m.UpdateRiskProfile(101.0, member.CareLevelComplex)
	assert.Error(t, err)
	// Score should remain unchanged
	assert.InDelta(t, 0.0, m.Risk.RiskScore, 0.001)
}

func TestMember_IsActive(t *testing.T) {
	first, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	m, _ := member.NewMember(first, last, email, phone, dob, gender, addr, planID, enrollDate)

	assert.True(t, m.IsActive())
	m.Terminate(time.Now())
	assert.False(t, m.IsActive())
}

func TestMember_FullName(t *testing.T) {
	first, last, email, phone, dob, gender, addr, planID, enrollDate := validMemberArgs()
	m, _ := member.NewMember(first, last, email, phone, dob, gender, addr, planID, enrollDate)
	assert.Equal(t, "Jane Doe", m.FullName())
}
