package member

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/rbalusup/healthcare-member-domain/internal/application/member/commands"
	"github.com/rbalusup/healthcare-member-domain/internal/application/member/queries"
	"github.com/rbalusup/healthcare-member-domain/internal/domain/member"
	"github.com/rbalusup/healthcare-member-domain/internal/infrastructure/metrics"
)

// EventPublisher publishes domain events to the message bus (Kafka).
type EventPublisher interface {
	PublishMemberCreated(ctx context.Context, m *member.Member) error
	PublishMemberUpdated(ctx context.Context, m *member.Member) error
	PublishMemberEnrolled(ctx context.Context, m *member.Member) error
	PublishMemberStatusChanged(ctx context.Context, m *member.Member, previous member.MemberStatus) error
}

// Service orchestrates Member use cases, coordinating the domain layer,
// persistence, and event publishing.
type Service struct {
	repo      member.Repository
	publisher EventPublisher
	logger    *zap.Logger
	metrics   *metrics.Metrics
}

// NewService constructs an application Service with its dependencies.
func NewService(
	repo member.Repository,
	publisher EventPublisher,
	logger *zap.Logger,
	m *metrics.Metrics,
) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
		metrics:   m,
	}
}

// CreateMember handles the CreateMemberCommand use case.
func (s *Service) CreateMember(ctx context.Context, cmd commands.CreateMemberCommand) (*queries.MemberResult, error) {
	// Check for duplicate email
	existing, err := s.repo.GetByEmail(ctx, cmd.Email)
	if err != nil && err != member.ErrMemberNotFound {
		return nil, fmt.Errorf("checking existing member: %w", err)
	}
	if existing != nil {
		return nil, member.ErrMemberAlreadyExists
	}

	m, err := member.NewMember(
		cmd.FirstName,
		cmd.LastName,
		cmd.Email,
		cmd.Phone,
		cmd.DateOfBirth,
		cmd.Gender,
		member.Address{
			Street:  cmd.Street,
			City:    cmd.City,
			State:   cmd.State,
			ZipCode: cmd.ZipCode,
			Country: cmd.Country,
		},
		cmd.PlanID,
		cmd.EnrollmentDate,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", member.ErrInvalidMemberData, err)
	}

	if err := s.repo.Create(ctx, m); err != nil {
		s.metrics.MembersCreatedErrorsTotal.Inc()
		return nil, fmt.Errorf("persisting member: %w", err)
	}

	s.metrics.MembersCreatedTotal.Inc()

	if err := s.publisher.PublishMemberCreated(ctx, m); err != nil {
		// Log but don't fail — event delivery is best-effort at this layer.
		s.logger.Warn("failed to publish member.created event",
			zap.String("member_id", m.MemberID.String()),
			zap.Error(err),
		)
	}

	s.logger.Info("member created",
		zap.String("member_id", m.MemberID.String()),
		zap.String("email", m.Contact.Email),
	)

	return toMemberResult(m), nil
}

// GetMember handles the GetMemberQuery use case.
func (s *Service) GetMember(ctx context.Context, qry queries.GetMemberQuery) (*queries.MemberResult, error) {
	m, err := s.repo.GetByID(ctx, qry.MemberID)
	if err != nil {
		return nil, err
	}
	return toMemberResult(m), nil
}

// ListMembers handles the ListMembersQuery use case.
func (s *Service) ListMembers(ctx context.Context, qry queries.ListMembersQuery) (*queries.ListMembersResult, error) {
	members, total, err := s.repo.List(ctx, qry.Filter, qry.Pagination)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}

	results := make([]*queries.MemberResult, 0, len(members))
	for _, m := range members {
		results = append(results, toMemberResult(m))
	}

	return &queries.ListMembersResult{
		Members:    results,
		TotalCount: total,
		Page:       qry.Pagination.Page,
		PageSize:   qry.Pagination.PageSize,
	}, nil
}

// UpdateMember handles the UpdateMemberCommand use case.
func (s *Service) UpdateMember(ctx context.Context, cmd commands.UpdateMemberCommand) (*queries.MemberResult, error) {
	m, err := s.repo.GetByID(ctx, cmd.MemberID)
	if err != nil {
		return nil, err
	}

	m.FirstName = cmd.FirstName
	m.LastName = cmd.LastName
	m.Contact.Phone = cmd.Phone
	m.Contact.Address = member.Address{
		Street:  cmd.Street,
		City:    cmd.City,
		State:   cmd.State,
		ZipCode: cmd.ZipCode,
		Country: cmd.Country,
	}

	if err := s.repo.Update(ctx, m); err != nil {
		s.metrics.MembersUpdatedErrorsTotal.Inc()
		return nil, fmt.Errorf("updating member: %w", err)
	}

	s.metrics.MembersUpdatedTotal.Inc()

	if err := s.publisher.PublishMemberUpdated(ctx, m); err != nil {
		s.logger.Warn("failed to publish member.updated event",
			zap.String("member_id", m.MemberID.String()),
			zap.Error(err),
		)
	}

	return toMemberResult(m), nil
}

// EnrollMember handles the EnrollMemberCommand use case.
func (s *Service) EnrollMember(ctx context.Context, cmd commands.EnrollMemberCommand) (*queries.MemberResult, error) {
	m, err := s.repo.GetByID(ctx, cmd.MemberID)
	if err != nil {
		return nil, err
	}

	if err := m.Enroll(cmd.PlanID, cmd.EnrollmentDate); err != nil {
		return nil, fmt.Errorf("%w: %s", member.ErrMemberEnrollmentFailed, err)
	}

	if err := s.repo.Update(ctx, m); err != nil {
		return nil, fmt.Errorf("persisting enrollment: %w", err)
	}

	s.metrics.MembersEnrolledTotal.Inc()

	if err := s.publisher.PublishMemberEnrolled(ctx, m); err != nil {
		s.logger.Warn("failed to publish member.enrolled event",
			zap.String("member_id", m.MemberID.String()),
			zap.Error(err),
		)
	}

	return toMemberResult(m), nil
}

// UpdateRiskProfile handles the UpdateRiskProfileCommand use case.
func (s *Service) UpdateRiskProfile(ctx context.Context, cmd commands.UpdateRiskProfileCommand) (*queries.RiskProfileResult, error) {
	m, err := s.repo.GetByID(ctx, cmd.MemberID)
	if err != nil {
		return nil, err
	}

	if err := m.UpdateRiskProfile(cmd.RiskScore, cmd.CareLevel); err != nil {
		return nil, fmt.Errorf("%w: %s", member.ErrInvalidMemberData, err)
	}

	if err := s.repo.Update(ctx, m); err != nil {
		return nil, fmt.Errorf("persisting risk profile: %w", err)
	}

	return &queries.RiskProfileResult{
		MemberID:  m.MemberID.String(),
		RiskScore: m.Risk.RiskScore,
		CareLevel: string(m.Risk.CareLevel),
		UpdatedAt: m.Risk.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// toMemberResult maps a domain Member to the application read model.
func toMemberResult(m *member.Member) *queries.MemberResult {
	result := &queries.MemberResult{
		MemberID:       m.MemberID.String(),
		FirstName:      m.FirstName,
		LastName:       m.LastName,
		Email:          m.Contact.Email,
		Phone:          m.Contact.Phone,
		DateOfBirth:    m.DateOfBirth.Format("2006-01-02"),
		Gender:         string(m.Gender),
		Street:         m.Contact.Address.Street,
		City:           m.Contact.Address.City,
		State:          m.Contact.Address.State,
		ZipCode:        m.Contact.Address.ZipCode,
		Country:        m.Contact.Address.Country,
		PlanID:         m.Enrollment.PlanID,
		EnrollmentDate: m.Enrollment.EnrollmentDate.Format("2006-01-02"),
		Status:         string(m.Status),
		RiskScore:      m.Risk.RiskScore,
		CareLevel:      string(m.Risk.CareLevel),
		CreatedAt:      m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	return result
}
