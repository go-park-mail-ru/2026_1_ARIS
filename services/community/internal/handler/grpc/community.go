package grpc

import (
	"context"
	"errors"
	"time"

	communitypb "github.com/go-park-mail-ru/2026_1_ARIS/proto/community"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	communitypb.UnimplementedCommunityServiceServer
	community *usecase.Service
}

func New(community *usecase.Service) *Server {
	return &Server{community: community}
}

func (s *Server) GetCommunity(ctx context.Context, req *communitypb.GetCommunityRequest) (*communitypb.CommunityResponse, error) {
	community, err := s.community.Get(ctx, req.GetCommunityId())
	if err != nil {
		return nil, toStatus(err)
	}
	return toCommunityResponse(community), nil
}

func (s *Server) GetCommunityByProfile(ctx context.Context, req *communitypb.GetCommunityByProfileRequest) (*communitypb.CommunityResponse, error) {
	community, err := s.community.GetByProfileID(ctx, req.GetProfileId())
	if err != nil {
		return nil, toStatus(err)
	}
	return toCommunityResponse(community), nil
}

func (s *Server) GetMember(ctx context.Context, req *communitypb.GetMemberRequest) (*communitypb.MemberResponse, error) {
	member, err := s.community.GetMember(ctx, req.GetCommunityId(), req.GetProfileId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &communitypb.MemberResponse{
		MemberId:    member.ID,
		CommunityId: member.CommunityID,
		ProfileId:   member.MemberID,
		Role:        string(usecase.NormalizeRole(member.Role)),
		IsActive:    member.IsActive,
	}, nil
}

func (s *Server) CanPostByProfile(ctx context.Context, req *communitypb.CanPostByProfileRequest) (*communitypb.CanPostResponse, error) {
	ok, err := s.community.CanPostByProfile(ctx, req.GetCommunityProfileId(), req.GetActorProfileId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &communitypb.CanPostResponse{Ok: ok}, nil
}

func (s *Server) CanPostAsMember(ctx context.Context, req *communitypb.CanPostAsMemberRequest) (*communitypb.CanPostResponse, error) {
	ok, err := s.community.CanPostAsMember(ctx, req.GetCommunityId(), req.GetActorProfileId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &communitypb.CanPostResponse{Ok: ok}, nil
}

func (s *Server) SearchCommunities(ctx context.Context, req *communitypb.SearchCommunitiesRequest) (*communitypb.SearchCommunitiesResponse, error) {
	communities, err := s.community.Search(ctx, req.GetQuery(), int(req.GetLimit()))
	if err != nil {
		return nil, toStatus(err)
	}
	resp := &communitypb.SearchCommunitiesResponse{Communities: make([]*communitypb.SearchCommunityItem, 0, len(communities))}
	for _, community := range communities {
		item := &communitypb.SearchCommunityItem{
			CommunityId: community.ID,
			ProfileId:   community.ProfileID,
			Username:    community.Username,
			Title:       community.Title,
			Type:        string(community.Type),
		}
		if community.Bio != nil {
			item.Bio = community.Bio
		}
		if community.AvatarID != nil {
			item.AvatarId = community.AvatarID
		}
		if community.CoverMediaID != nil {
			item.CoverMediaId = community.CoverMediaID
		}
		resp.Communities = append(resp.Communities, item)
	}
	return resp, nil
}

func toCommunityResponse(details *usecase.CommunityDetails) *communitypb.CommunityResponse {
	community := details.Community
	resp := &communitypb.CommunityResponse{
		CommunityId: community.ID,
		Uid:         community.Uid.String(),
		Title:       community.Title,
		Type:        string(community.Type),
		ProfileId:   community.ProfileID,
		Username:    community.Username,
		IsActive:    community.IsActive,
		CreatedAt:   community.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   community.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if community.Bio != nil {
		resp.Bio = community.Bio
	}
	if community.CoverMediaID != nil {
		resp.CoverMediaId = community.CoverMediaID
	}
	if details.AvatarID != nil {
		resp.AvatarId = details.AvatarID
	}
	return resp
}

func toStatus(err error) error {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, usecase.ErrCommunityNotFound), errors.Is(err, usecase.ErrCommunityMemberNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
