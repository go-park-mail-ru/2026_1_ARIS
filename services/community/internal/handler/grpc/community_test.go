package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	communitypb "github.com/go-park-mail-ru/2026_1_ARIS/proto/community"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/repository"
	repositorymock "github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerGetCommunity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repositorymock.NewMockCommunityRepo(ctrl)
	server := New(usecase.New(repo, nil))

	id := uuid.New()
	avatarID := int64(15)
	coverID := int64(25)
	bio := "bio"
	created := time.Date(2026, 5, 27, 10, 0, 0, 123, time.UTC)
	updated := created.Add(time.Hour)
	repo.EXPECT().Get(gomock.Any(), int64(1)).Return(&model.Community{
		ID: 1, Uid: id, ProfileID: 10, Title: "Title", Bio: &bio, Type: model.PublicGroup, Username: "community", CoverMediaID: &coverID, IsActive: true, CreatedAt: created, UpdatedAt: updated,
	}, nil)
	repo.EXPECT().GetAvatarID(gomock.Any(), int64(10)).Return(&avatarID, nil)

	resp, err := server.GetCommunity(context.Background(), &communitypb.GetCommunityRequest{CommunityId: 1})
	if err != nil {
		t.Fatalf("GetCommunity() error = %v", err)
	}
	if resp.GetCommunityId() != 1 || resp.GetUid() != id.String() || resp.GetAvatarId() != avatarID || resp.GetCoverMediaId() != coverID || resp.GetBio() != bio {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestServerGetMemberNormalizesRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repositorymock.NewMockCommunityRepo(ctrl)
	server := New(usecase.New(repo, nil))

	repo.EXPECT().GetMember(gomock.Any(), int64(2), int64(30)).Return(&model.CommunityMember{
		ID: 9, CommunityID: 2, MemberID: 30, Role: model.CommunityMemberRole("manager"), IsActive: true,
	}, nil)
	resp, err := server.GetMember(context.Background(), &communitypb.GetMemberRequest{CommunityId: 2, ProfileId: 30})
	if err != nil {
		t.Fatalf("GetMember() error = %v", err)
	}
	if resp.GetRole() != string(model.Moderator) || !resp.GetIsActive() {
		t.Fatalf("unexpected member response: %+v", resp)
	}
}

func TestServerCanPostAndSearch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repositorymock.NewMockCommunityRepo(ctrl)
	server := New(usecase.New(repo, nil))

	repo.EXPECT().GetByProfileID(gomock.Any(), int64(100)).Return(&model.Community{ID: 3, ProfileID: 100}, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(3), int64(200)).Return(&model.CommunityMember{Role: model.Moderator}, nil)
	canPost, err := server.CanPostByProfile(context.Background(), &communitypb.CanPostByProfileRequest{CommunityProfileId: 100, ActorProfileId: 200})
	if err != nil {
		t.Fatalf("CanPostByProfile() error = %v", err)
	}
	if !canPost.GetOk() {
		t.Fatal("expected moderator to post by community profile")
	}

	avatarID := int64(5)
	repo.EXPECT().Search(gomock.Any(), "query", 10).Return(usecaseSearchResult(), nil)
	resp, err := server.SearchCommunities(context.Background(), &communitypb.SearchCommunitiesRequest{Query: " query ", Limit: 0})
	if err != nil {
		t.Fatalf("SearchCommunities() error = %v", err)
	}
	if len(resp.GetCommunities()) != 1 || resp.GetCommunities()[0].GetAvatarId() != avatarID {
		t.Fatalf("unexpected search response: %+v", resp)
	}
}

func TestToStatus(t *testing.T) {
	cases := map[error]codes.Code{
		usecase.ErrInvalidInput:            codes.InvalidArgument,
		usecase.ErrCommunityNotFound:       codes.NotFound,
		usecase.ErrCommunityMemberNotFound: codes.NotFound,
		errors.New("db down"):              codes.Internal,
	}
	for err, want := range cases {
		if got := status.Code(toStatus(err)); got != want {
			t.Fatalf("toStatus(%v) = %v, want %v", err, got, want)
		}
	}
}

func usecaseSearchResult() []repository.SearchCommunityResult {
	avatarID := int64(5)
	coverID := int64(6)
	bio := "bio"
	return []repository.SearchCommunityResult{
		{ID: 1, ProfileID: 10, Username: "community", Title: "Title", Bio: &bio, Type: model.PublicGroup, AvatarID: &avatarID, CoverMediaID: &coverID},
	}
}
