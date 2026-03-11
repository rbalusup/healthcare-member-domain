package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	memberv1 "github.com/rbalusup/healthcare-member-domain/gen/go/member/v1"
	appMember "github.com/rbalusup/healthcare-member-domain/internal/application/member"
	"github.com/rbalusup/healthcare-member-domain/internal/domain/member"
	grpcHandler "github.com/rbalusup/healthcare-member-domain/internal/handler/grpc"
	"github.com/rbalusup/healthcare-member-domain/internal/infrastructure/metrics"
	"github.com/rbalusup/healthcare-member-domain/internal/testutil"
)

func newHandlerWithMocks(repo *testutil.MockRepository, pub *testutil.MockEventPublisher) *grpcHandler.MemberHandler {
	svc := appMember.NewService(repo, pub, zap.NewNop(), metrics.NewForTest())
	return grpcHandler.NewMemberHandler(svc, zap.NewNop())
}

func TestGetMember_InvalidUUID(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	h := newHandlerWithMocks(repo, pub)

	_, err := h.GetMember(context.Background(), &memberv1.GetMemberRequest{MemberId: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetMember_NotFound(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	h := newHandlerWithMocks(repo, pub)

	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, member.ErrMemberNotFound)

	_, err := h.GetMember(context.Background(), &memberv1.GetMemberRequest{
		MemberId: "00000000-0000-0000-0000-000000000001",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCreateMember_MapsRequestToCommand(t *testing.T) {
	repo := &testutil.MockRepository{}
	pub := &testutil.MockEventPublisher{}
	h := newHandlerWithMocks(repo, pub)

	repo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, member.ErrMemberNotFound)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(m *member.Member) bool {
		return m.Contact.Email == "test@example.com" &&
			m.FirstName == "Test" &&
			m.Enrollment.PlanID == "PPO-001"
	})).Return(nil)
	pub.On("PublishMemberCreated", mock.Anything, mock.Anything).Return(nil)

	req := &memberv1.CreateMemberRequest{
		FirstName:      "Test",
		LastName:       "User",
		Email:          "test@example.com",
		Phone:          "555-0200",
		DateOfBirth:    timestamppb.New(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
		Gender:         memberv1.Gender_GENDER_MALE,
		PlanId:         "PPO-001",
		EnrollmentDate: timestamppb.New(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	result, err := h.CreateMember(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Test", result.FirstName)
	assert.Equal(t, "test@example.com", result.Email)

	repo.AssertExpectations(t)
	pub.AssertExpectations(t)
}
