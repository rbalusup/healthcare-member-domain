package testutil

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	appMember "github.com/rbalusup/healthcare-member-domain/internal/application/member"
	"github.com/rbalusup/healthcare-member-domain/internal/domain/member"
)

// MockRepository is a testify mock implementation of member.Repository.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, mem *member.Member) error {
	args := m.Called(ctx, mem)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*member.Member, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*member.Member), args.Error(1)
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) (*member.Member, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*member.Member), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, mem *member.Member) error {
	args := m.Called(ctx, mem)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, filter member.Filter, pagination member.Pagination) ([]*member.Member, int64, error) {
	args := m.Called(ctx, filter, pagination)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*member.Member), args.Get(1).(int64), args.Error(2)
}

// MockEventPublisher is a testify mock implementation of appMember.EventPublisher.
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishMemberCreated(ctx context.Context, mem *member.Member) error {
	args := m.Called(ctx, mem)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishMemberUpdated(ctx context.Context, mem *member.Member) error {
	args := m.Called(ctx, mem)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishMemberEnrolled(ctx context.Context, mem *member.Member) error {
	args := m.Called(ctx, mem)
	return args.Error(0)
}

func (m *MockEventPublisher) PublishMemberStatusChanged(ctx context.Context, mem *member.Member, previous member.MemberStatus) error {
	args := m.Called(ctx, mem, previous)
	return args.Error(0)
}

// Ensure interfaces are satisfied at compile time.
var _ member.Repository = (*MockRepository)(nil)
var _ appMember.EventPublisher = (*MockEventPublisher)(nil)
