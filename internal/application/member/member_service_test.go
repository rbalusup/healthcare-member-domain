package member_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	appMember "github.com/rbalusup/healthcare-member-domain/internal/application/member"
	"github.com/rbalusup/healthcare-member-domain/internal/application/member/commands"
	"github.com/rbalusup/healthcare-member-domain/internal/application/member/queries"
	"github.com/rbalusup/healthcare-member-domain/internal/domain/member"
	"github.com/rbalusup/healthcare-member-domain/internal/infrastructure/metrics"
	"github.com/rbalusup/healthcare-member-domain/internal/testutil"
)

func newTestService(repo *testutil.MockRepository, pub *testutil.MockEventPublisher) *appMember.Service {
	return appMember.NewService(repo, pub, zap.NewNop(), metrics.NewForTest())
}

func newTestMember(t *testing.T) *member.Member {
	t.Helper()
	m, err := member.NewMember(
		"Jane", "Doe", "jane.doe@example.com", "555-0100",
		time.Date(1985, 6, 15, 0, 0, 0, 0, time.UTC),
		member.GenderFemale,
		member.Address{Street: "123 Main St", City: "Springfield", State: "IL", ZipCode: "62701", Country: "US"},
		"PPO-2024-001",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	return m
}

func TestCreateMember_Success(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	svc := newTestService(repo, pub)

	repo.On("GetByEmail", mock.Anything, "jane.doe@example.com").Return(nil, member.ErrMemberNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*member.Member")).Return(nil)
	pub.On("PublishMemberCreated", mock.Anything, mock.AnythingOfType("*member.Member")).Return(nil)

	cmd := commands.CreateMemberCommand{
		FirstName:      "Jane",
		LastName:       "Doe",
		Email:          "jane.doe@example.com",
		Phone:          "555-0100",
		DateOfBirth:    time.Date(1985, 6, 15, 0, 0, 0, 0, time.UTC),
		Gender:         member.GenderFemale,
		PlanID:         "PPO-2024-001",
		EnrollmentDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	result, err := svc.CreateMember(context.Background(), cmd)
	require.NoError(t, err)
	assert.Equal(t, "Jane", result.FirstName)
	assert.Equal(t, "jane.doe@example.com", result.Email)

	repo.AssertExpectations(t)
	pub.AssertExpectations(t)
}

func TestCreateMember_DuplicateEmail(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	svc := newTestService(repo, pub)

	existing := newTestMember(t)
	repo.On("GetByEmail", mock.Anything, "jane.doe@example.com").Return(existing, nil)

	cmd := commands.CreateMemberCommand{
		Email:          "jane.doe@example.com",
		FirstName:      "Jane",
		LastName:       "Doe",
		PlanID:         "PPO-2024-001",
		DateOfBirth:    time.Now().AddDate(-30, 0, 0),
		EnrollmentDate: time.Now(),
	}

	_, err := svc.CreateMember(context.Background(), cmd)
	assert.ErrorIs(t, err, member.ErrMemberAlreadyExists)
	repo.AssertExpectations(t)
}

func TestGetMember_NotFound(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	svc := newTestService(repo, pub)

	id := uuid.New()
	repo.On("GetByID", mock.Anything, id).Return(nil, member.ErrMemberNotFound)

	_, err := svc.GetMember(context.Background(), queries.GetMemberQuery{MemberID: id})
	assert.ErrorIs(t, err, member.ErrMemberNotFound)
}

func TestListMembers_Pagination(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	svc := newTestService(repo, pub)

	m1 := newTestMember(t)
	members := []*member.Member{m1}

	filter := member.Filter{SearchTerm: "jane"}
	pagination := member.Pagination{Page: 2, PageSize: 10}

	repo.On("List", mock.Anything, filter, pagination).Return(members, int64(1), nil)

	result, err := svc.ListMembers(context.Background(), queries.ListMembersQuery{
		Filter:     filter,
		Pagination: pagination,
	})
	require.NoError(t, err)
	assert.Len(t, result.Members, 1)
	assert.Equal(t, int64(1), result.TotalCount)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 10, result.PageSize)
}

func TestEnrollMember_Success(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	svc := newTestService(repo, pub)

	m := newTestMember(t)
	repo.On("GetByID", mock.Anything, m.MemberID).Return(m, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*member.Member")).Return(nil)
	pub.On("PublishMemberEnrolled", mock.Anything, mock.AnythingOfType("*member.Member")).Return(nil)

	enrollDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cmd := commands.EnrollMemberCommand{
		MemberID:       m.MemberID,
		PlanID:         "HMO-2025-001",
		EnrollmentDate: enrollDate,
	}

	result, err := svc.EnrollMember(context.Background(), cmd)
	require.NoError(t, err)
	assert.Equal(t, "HMO-2025-001", result.PlanID)

	repo.AssertExpectations(t)
	pub.AssertExpectations(t)
}

func TestUpdateRiskProfile_InvalidScore(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	svc := newTestService(repo, pub)

	m := newTestMember(t)
	repo.On("GetByID", mock.Anything, m.MemberID).Return(m, nil)

	cmd := commands.UpdateRiskProfileCommand{
		MemberID:  m.MemberID,
		RiskScore: 150.0, // invalid: > 100
		CareLevel: member.CareLevelComplex,
	}

	_, err := svc.UpdateRiskProfile(context.Background(), cmd)
	assert.ErrorIs(t, err, member.ErrInvalidMemberData)
}
