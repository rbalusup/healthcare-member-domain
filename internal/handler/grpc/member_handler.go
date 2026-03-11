package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	memberv1 "github.com/rbalusup/healthcare-member-domain/gen/go/member/v1"
	appMember "github.com/rbalusup/healthcare-member-domain/internal/application/member"
	"github.com/rbalusup/healthcare-member-domain/internal/application/member/commands"
	"github.com/rbalusup/healthcare-member-domain/internal/application/member/queries"
	"github.com/rbalusup/healthcare-member-domain/internal/domain/member"
)

// MemberHandler implements the gRPC MemberServiceServer interface.
// It translates gRPC requests to application commands/queries and maps
// domain errors to appropriate gRPC status codes.
type MemberHandler struct {
	memberv1.UnimplementedMemberServiceServer
	svc    *appMember.Service
	logger *zap.Logger
}

// NewMemberHandler constructs a MemberHandler.
func NewMemberHandler(svc *appMember.Service, logger *zap.Logger) *MemberHandler {
	return &MemberHandler{svc: svc, logger: logger}
}

// CreateMember handles the CreateMember RPC.
func (h *MemberHandler) CreateMember(ctx context.Context, req *memberv1.CreateMemberRequest) (*memberv1.Member, error) {
	enrollDate := req.EnrollmentDate.AsTime()
	dob := req.DateOfBirth.AsTime()

	cmd := commands.CreateMemberCommand{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		Phone:          req.Phone,
		DateOfBirth:    dob,
		Gender:         protoGenderToDomain(req.Gender),
		Street:         req.Address.GetStreet(),
		City:           req.Address.GetCity(),
		State:          req.Address.GetState(),
		ZipCode:        req.Address.GetZipCode(),
		Country:        req.Address.GetCountry(),
		PlanID:         req.PlanId,
		EnrollmentDate: enrollDate,
	}

	result, err := h.svc.CreateMember(ctx, cmd)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return toProtoMember(result), nil
}

// GetMember handles the GetMember RPC.
func (h *MemberHandler) GetMember(ctx context.Context, req *memberv1.GetMemberRequest) (*memberv1.Member, error) {
	id, err := parseUUID(req.MemberId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid member_id: %v", err)
	}

	result, err := h.svc.GetMember(ctx, queries.GetMemberQuery{MemberID: id})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return toProtoMember(result), nil
}

// UpdateMember handles the UpdateMember RPC.
func (h *MemberHandler) UpdateMember(ctx context.Context, req *memberv1.UpdateMemberRequest) (*memberv1.Member, error) {
	id, err := parseUUID(req.MemberId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid member_id: %v", err)
	}

	cmd := commands.UpdateMemberCommand{
		MemberID:  id,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Street:    req.Address.GetStreet(),
		City:      req.Address.GetCity(),
		State:     req.Address.GetState(),
		ZipCode:   req.Address.GetZipCode(),
		Country:   req.Address.GetCountry(),
	}

	result, err := h.svc.UpdateMember(ctx, cmd)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return toProtoMember(result), nil
}

// ListMembers handles the ListMembers RPC.
func (h *MemberHandler) ListMembers(ctx context.Context, req *memberv1.ListMembersRequest) (*memberv1.ListMembersResponse, error) {
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}

	qry := queries.ListMembersQuery{
		Filter:     member.Filter{SearchTerm: req.SearchTerm},
		Pagination: member.Pagination{Page: page, PageSize: pageSize},
	}

	result, err := h.svc.ListMembers(ctx, qry)
	if err != nil {
		return nil, toGRPCError(err)
	}

	protoMembers := make([]*memberv1.Member, 0, len(result.Members))
	for _, m := range result.Members {
		protoMembers = append(protoMembers, toProtoMember(m))
	}

	return &memberv1.ListMembersResponse{
		Members:    protoMembers,
		TotalCount: result.TotalCount,
		Page:       int32(result.Page),
		PageSize:   int32(result.PageSize),
	}, nil
}

// EnrollMember handles the EnrollMember RPC.
func (h *MemberHandler) EnrollMember(ctx context.Context, req *memberv1.EnrollMemberRequest) (*memberv1.Member, error) {
	id, err := parseUUID(req.MemberId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid member_id: %v", err)
	}

	cmd := commands.EnrollMemberCommand{
		MemberID:       id,
		PlanID:         req.PlanId,
		EnrollmentDate: req.EnrollmentDate.AsTime(),
	}

	result, err := h.svc.EnrollMember(ctx, cmd)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return toProtoMember(result), nil
}

// GetMemberRiskProfile handles the GetMemberRiskProfile RPC.
func (h *MemberHandler) GetMemberRiskProfile(ctx context.Context, req *memberv1.GetMemberRiskProfileRequest) (*memberv1.RiskProfile, error) {
	id, err := parseUUID(req.MemberId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid member_id: %v", err)
	}

	result, err := h.svc.GetMember(ctx, queries.GetMemberQuery{MemberID: id})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &memberv1.RiskProfile{
		MemberId:  result.MemberID,
		RiskScore: result.RiskScore,
		CareLevel: result.CareLevel,
		UpdatedAt: timestamppb.Now(),
	}, nil
}

// toGRPCError maps domain errors to gRPC status codes.
func toGRPCError(err error) error {
	switch {
	case errors.Is(err, member.ErrMemberNotFound):
		return status.Errorf(codes.NotFound, "member not found")
	case errors.Is(err, member.ErrMemberAlreadyExists):
		return status.Errorf(codes.AlreadyExists, "member already exists")
	case errors.Is(err, member.ErrInvalidMemberData):
		return status.Errorf(codes.InvalidArgument, "%s", err.Error())
	case errors.Is(err, member.ErrMemberEnrollmentFailed):
		return status.Errorf(codes.FailedPrecondition, "%s", err.Error())
	case errors.Is(err, member.ErrMemberNotActive):
		return status.Errorf(codes.FailedPrecondition, "member is not active")
	default:
		return status.Errorf(codes.Internal, "internal server error")
	}
}

// toProtoMember converts an application read model to a proto Member message.
func toProtoMember(r *queries.MemberResult) *memberv1.Member {
	return &memberv1.Member{
		MemberId:  r.MemberID,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Email:     r.Email,
		Phone:     r.Phone,
		Gender:    domainGenderToProto(r.Gender),
		Status:    domainStatusToProto(r.Status),
		PlanId:    r.PlanID,
		RiskScore: r.RiskScore,
		CareLevel: r.CareLevel,
	}
}

// domainGenderToProto maps a domain gender string to the proto enum constant.
// Real protoc output uses full prefixed names (GENDER_MALE) not short aliases (MALE).
func domainGenderToProto(g string) memberv1.Gender {
	switch g {
	case "MALE":
		return memberv1.Gender_GENDER_MALE
	case "FEMALE":
		return memberv1.Gender_GENDER_FEMALE
	case "NON_BINARY":
		return memberv1.Gender_GENDER_NON_BINARY
	default:
		return memberv1.Gender_GENDER_UNSPECIFIED
	}
}

// domainStatusToProto maps a domain status string to the proto enum constant.
func domainStatusToProto(s string) memberv1.MemberStatus {
	switch s {
	case "ACTIVE":
		return memberv1.MemberStatus_MEMBER_STATUS_ACTIVE
	case "INACTIVE":
		return memberv1.MemberStatus_MEMBER_STATUS_INACTIVE
	case "SUSPENDED":
		return memberv1.MemberStatus_MEMBER_STATUS_SUSPENDED
	default:
		return memberv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED
	}
}

// protoGenderToDomain maps a proto Gender enum to the domain string constant.
// Real protoc output's Gender.String() returns "GENDER_MALE", not "MALE".
func protoGenderToDomain(g memberv1.Gender) member.Gender {
	switch g {
	case memberv1.Gender_GENDER_MALE:
		return member.GenderMale
	case memberv1.Gender_GENDER_FEMALE:
		return member.GenderFemale
	case memberv1.Gender_GENDER_NON_BINARY:
		return member.GenderNonBinary
	default:
		return member.GenderUnspecified
	}
}

// parseUUID parses a string UUID, returning an error on invalid input.
func parseUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.UUID{}, errors.New("member_id is required")
	}
	return uuid.Parse(s)
}
