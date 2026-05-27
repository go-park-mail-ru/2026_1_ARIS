package usecase

import (
	"context"
	"errors"
	"testing"

	communitypb "github.com/go-park-mail-ru/2026_1_ARIS/proto/community"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type commentCommunityClient struct {
	member *communitypb.MemberResponse
	err    error
}

func (c *commentCommunityClient) GetCommunity(context.Context, *communitypb.GetCommunityRequest, ...grpc.CallOption) (*communitypb.CommunityResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *commentCommunityClient) GetCommunityByProfile(context.Context, *communitypb.GetCommunityByProfileRequest, ...grpc.CallOption) (*communitypb.CommunityResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *commentCommunityClient) GetMember(context.Context, *communitypb.GetMemberRequest, ...grpc.CallOption) (*communitypb.MemberResponse, error) {
	return c.member, c.err
}

func (c *commentCommunityClient) CanPostByProfile(context.Context, *communitypb.CanPostByProfileRequest, ...grpc.CallOption) (*communitypb.CanPostResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *commentCommunityClient) CanPostAsMember(context.Context, *communitypb.CanPostAsMemberRequest, ...grpc.CallOption) (*communitypb.CanPostResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *commentCommunityClient) SearchCommunities(context.Context, *communitypb.SearchCommunitiesRequest, ...grpc.CallOption) (*communitypb.SearchCommunitiesResponse, error) {
	return nil, errors.New("not implemented")
}

func TestEnsureCanCommentCommunityPostAllowsNonMembers(t *testing.T) {
	service := &Service{
		communityClient: &commentCommunityClient{
			err: status.Error(codes.NotFound, "member not found"),
		},
	}

	if err := service.ensureCanCommentCommunityPost(context.Background(), 10, 20); err != nil {
		t.Fatalf("ensureCanCommentCommunityPost() error = %v, want nil", err)
	}
}

func TestEnsureCanCommentCommunityPostDeniesBlockedMembers(t *testing.T) {
	service := &Service{
		communityClient: &commentCommunityClient{
			member: &communitypb.MemberResponse{Role: "blocked", IsActive: true},
		},
	}

	err := service.ensureCanCommentCommunityPost(context.Background(), 10, 20)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ensureCanCommentCommunityPost() error = %v, want %v", err, ErrForbidden)
	}
}

func TestEnsureCanCommentCommunityPostAllowsRegularMembers(t *testing.T) {
	service := &Service{
		communityClient: &commentCommunityClient{
			member: &communitypb.MemberResponse{Role: "member", IsActive: true},
		},
	}

	if err := service.ensureCanCommentCommunityPost(context.Background(), 10, 20); err != nil {
		t.Fatalf("ensureCanCommentCommunityPost() error = %v, want nil", err)
	}
}

func TestEnsureCanViewPostHidesBlockedCommunityPost(t *testing.T) {
	communityID := int64(10)
	service := &Service{
		communityClient: &commentCommunityClient{
			member: &communitypb.MemberResponse{Role: "blocked", IsActive: true},
		},
	}

	err := service.ensureCanViewPost(context.Background(), model.Post{CommunityID: &communityID}, 20)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("ensureCanViewPost() error = %v, want %v", err, ErrPostNotFound)
	}
}

func TestEnsureCanViewPostAllowsNonMemberCommunityPost(t *testing.T) {
	communityID := int64(10)
	service := &Service{
		communityClient: &commentCommunityClient{
			err: status.Error(codes.NotFound, "member not found"),
		},
	}

	err := service.ensureCanViewPost(context.Background(), model.Post{CommunityID: &communityID}, 20)
	if err != nil {
		t.Fatalf("ensureCanViewPost() error = %v, want nil", err)
	}
}
