package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/model"
	repositorymock "github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestValidateCreate(t *testing.T) {
	bio := " short bio "
	avatarID := int64(1)
	coverID := int64(2)

	if err := validateCreate(CreateInput{
		Title:        " Community ",
		Bio:          &bio,
		Type:         model.PublicGroup,
		Username:     "My-Community",
		AvatarID:     &avatarID,
		CoverMediaID: &coverID,
	}); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}

	longBio := makeString(2048)
	tests := []CreateInput{
		{Title: "", Type: model.PublicGroup, Username: "valid"},
		{Title: makeString(65), Type: model.PublicGroup, Username: "valid"},
		{Title: "title", Type: model.CommunityType("secret"), Username: "valid"},
		{Title: "title", Type: model.PublicGroup, Username: "-bad"},
		{Title: "title", Type: model.PublicGroup, Username: "bad_underscore"},
		{Title: "title", Type: model.PublicGroup, Username: "valid", Bio: &longBio},
		{Title: "title", Type: model.PublicGroup, Username: "valid", AvatarID: int64Ptr(0)},
		{Title: "title", Type: model.PublicGroup, Username: "valid", CoverMediaID: int64Ptr(-1)},
	}
	for _, tc := range tests {
		if err := validateCreate(tc); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected invalid input for %+v, got %v", tc, err)
		}
	}
}

func TestCommunityHelpers(t *testing.T) {
	if got := normalizeUsername("  MIXED-Name "); got != "mixed-name" {
		t.Fatalf("normalizeUsername() = %q", got)
	}
	if !isValidCommunityUsername("abc-123") {
		t.Fatal("expected username to be valid")
	}
	for _, username := range []string{"ab", "too-long-community-name", "-bad", "bad-", "bad_name", "кириллица"} {
		if isValidCommunityUsername(username) {
			t.Fatalf("expected %q to be invalid", username)
		}
	}

	raw := "  bio  "
	if got := trimPtr(&raw); got == nil || *got != "bio" {
		t.Fatalf("trimPtr() = %#v", got)
	}
	blank := "   "
	if got := trimPtr(&blank); got != nil {
		t.Fatalf("expected blank trimPtr() to return nil, got %#v", got)
	}
	if NormalizeRole(model.CommunityMemberRole("manager")) != model.Moderator {
		t.Fatal("legacy manager role should normalize to moderator")
	}
}

func TestCommunityRolePermissions(t *testing.T) {
	if !canManage(&model.CommunityMember{Role: model.Moderator, IsActive: true}) {
		t.Fatal("active moderator should manage community content")
	}
	if canManage(&model.CommunityMember{Role: model.Owner, IsActive: false}) {
		t.Fatal("inactive owner should not manage community")
	}
	if !canEditCommunity(model.Admin) || canEditCommunity(model.Moderator) {
		t.Fatal("unexpected edit permissions")
	}
	if !canPostAsCommunityRole(model.Moderator) || canPostAsCommunityRole(model.Member) {
		t.Fatal("unexpected community posting permissions")
	}
	if !canPostAsMemberRole(model.Member) || canPostAsMemberRole(model.Blocked) {
		t.Fatal("unexpected member posting permissions")
	}
	if !canManageMembers(model.Admin) || canManageMembers(model.Moderator) {
		t.Fatal("unexpected member management permissions")
	}
	if !canRemoveMember(model.Admin, model.Blocked) || canRemoveMember(model.Admin, model.Owner) || canRemoveMember(model.Member, model.Blocked) {
		t.Fatal("unexpected remove-member permissions")
	}
	if !isAssignableRole(model.Admin) || !isAssignableRole(model.Blocked) || isAssignableRole(model.Owner) {
		t.Fatal("unexpected assignable role result")
	}
}

func TestServiceCreateNormalizesAndDecorates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := repositorymock.NewMockCommunityRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	media := mediamock.NewMockMediaServiceClient(ctrl)
	service := New(repo, users, media)

	avatarID := int64(10)
	coverID := int64(20)
	bio := "  bio  "
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 7}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 70}, nil)
	repo.EXPECT().
		Create(gomock.Any(), gomock.Any(), int64(70), &avatarID).
		DoAndReturn(func(_ context.Context, community model.Community, ownerProfileID int64, gotAvatarID *int64) (*model.Community, error) {
			if community.Title != "Title" || community.Bio == nil || *community.Bio != "bio" || community.Username != "mixed-name" {
				t.Fatalf("community was not normalized: %+v", community)
			}
			if community.CoverMediaID == nil || *community.CoverMediaID != coverID {
				t.Fatalf("cover id was not propagated: %+v", community.CoverMediaID)
			}
			return &model.Community{ID: 1, ProfileID: 100, Title: community.Title, Bio: community.Bio, Type: community.Type, Username: community.Username, CoverMediaID: community.CoverMediaID, IsActive: true}, nil
		})
	repo.EXPECT().GetAvatarID(gomock.Any(), int64(100)).Return(&avatarID, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(1), int64(70)).Return(&model.CommunityMember{Role: model.Owner, IsActive: true}, nil)
	media.EXPECT().GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: avatarID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn/avatar"}, nil)
	media.EXPECT().GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: coverID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn/cover"}, nil)

	details, err := service.Create(ctx, 7, CreateInput{Title: " Title ", Bio: &bio, Type: model.PublicGroup, Username: " MIXED-Name ", AvatarID: &avatarID, CoverMediaID: &coverID})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if details.AvatarURL == nil || *details.AvatarURL != "https://cdn/avatar" || details.CoverURL == nil || *details.CoverURL != "https://cdn/cover" {
		t.Fatalf("unexpected media urls: %+v", details)
	}
	if !details.Permission.CanDeleteCommunity || !details.Permission.CanPostAsCommunity || details.Membership.Role == nil || *details.Membership.Role != model.Owner {
		t.Fatalf("unexpected owner permissions: %+v", details)
	}
}

func TestServiceCheckExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := repositorymock.NewMockCommunityRepo(ctrl)
	service := New(repo, nil)

	if _, err := service.CheckExists(context.Background(), CheckExistsInput{Title: "", Username: "ok-name"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	repo.EXPECT().ExistsByTitleOrUsername(gomock.Any(), "", "existing").Return(false, true, nil)
	result, err := service.CheckExists(context.Background(), CheckExistsInput{Title: "Existing", Username: " Existing "})
	if err != nil {
		t.Fatalf("CheckExists() error = %v", err)
	}
	if !result.Exists || !result.UsernameExists || result.TitleExists {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestServiceListMembersDecoratesProfiles(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := repositorymock.NewMockCommunityRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	media := mediamock.NewMockMediaServiceClient(ctrl)
	service := New(repo, users, media)

	viewerAccountID := int64(5)
	avatarID := int64(99)
	joinedAt := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: viewerAccountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 50}, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(11), int64(50)).Return(&model.CommunityMember{Role: model.Admin, IsActive: true}, nil)
	repo.EXPECT().ListMembers(gomock.Any(), int64(11), true).Return([]model.CommunityMember{
		{MemberID: 50, Role: model.Member, JoinedAt: joinedAt, IsActive: true},
		{MemberID: 60, Role: model.CommunityMemberRole("manager"), JoinedAt: joinedAt.Add(time.Hour), IsActive: true},
	}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 50}).Return(&userpb.GetProfileSummaryResponse{
		ProfileId: 50, UserAccountId: 5, FirstName: "Ann", LastName: "Owner", Username: "ann", AvatarId: &avatarID,
	}, nil)
	media.EXPECT().GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: avatarID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn/99"}, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 60}).Return(&userpb.GetProfileSummaryResponse{
		ProfileId: 60, UserAccountId: 6, FirstName: "Bob", LastName: "Mod", Username: "bob",
	}, nil)

	members, err := service.ListMembers(ctx, 11, &viewerAccountID, true)
	if err != nil {
		t.Fatalf("ListMembers() error = %v", err)
	}
	if len(members) != 2 || !members[0].IsSelf || members[1].Role != model.Moderator {
		t.Fatalf("unexpected members: %+v", members)
	}
	if members[0].AvatarURL == nil || *members[0].AvatarURL != "https://cdn/99" || members[0].JoinedAt != joinedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected first member details: %+v", members[0])
	}
}

func TestServiceJoinLeaveAndChangeRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := repositorymock.NewMockCommunityRepo(ctrl)
	users := usermock.NewMockUserServiceClient(ctrl)
	service := New(repo, users)

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 10}).Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 100}, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(1), int64(100)).Return(nil, ErrCommunityMemberNotFound)
	joined := model.CommunityMember{CommunityID: 1, MemberID: 100, Role: model.Member, JoinedAt: time.Unix(1, 0).UTC(), IsActive: true}
	repo.EXPECT().UpsertMemberRole(gomock.Any(), int64(1), int64(100), model.Member).Return(&joined, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 100}).Return(&userpb.GetProfileSummaryResponse{ProfileId: 100, UserAccountId: 10, Username: "member"}, nil)
	member, err := service.Join(ctx, 10, 1)
	if err != nil {
		t.Fatalf("Join() error = %v", err)
	}
	if member.Role != model.Member || !member.IsSelf {
		t.Fatalf("unexpected joined member: %+v", member)
	}

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 10}).Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 100}, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(1), int64(100)).Return(&model.CommunityMember{Role: model.Owner}, nil)
	if err := service.Leave(ctx, 10, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected owner leave to be forbidden, got %v", err)
	}

	users.EXPECT().GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 20}).Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 200}, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(1), int64(200)).Return(&model.CommunityMember{Role: model.Owner, IsActive: true}, nil)
	repo.EXPECT().GetMember(gomock.Any(), int64(1), int64(300)).Return(&model.CommunityMember{Role: model.Member, IsActive: true}, nil)
	updated := model.CommunityMember{CommunityID: 1, MemberID: 300, Role: model.Admin, JoinedAt: time.Unix(2, 0).UTC(), IsActive: true}
	repo.EXPECT().UpsertMemberRole(gomock.Any(), int64(1), int64(300), model.Admin).Return(&updated, nil)
	users.EXPECT().GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: 300}).Return(&userpb.GetProfileSummaryResponse{ProfileId: 300, UserAccountId: 30, Username: "admin"}, nil)
	changed, err := service.ChangeMemberRole(ctx, 20, 1, 300, model.Admin)
	if err != nil {
		t.Fatalf("ChangeMemberRole() error = %v", err)
	}
	if changed.Role != model.Admin || changed.IsSelf {
		t.Fatalf("unexpected changed member: %+v", changed)
	}
}

func TestServiceProfileIDByUserAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	users := usermock.NewMockUserServiceClient(ctrl)
	service := New(nil, users)

	if _, err := service.profileIDByUserAccount(ctx, 0); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 1}).
		Return(nil, status.Error(codes.NotFound, "not found"))
	if _, err := service.profileIDByUserAccount(ctx, 1); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected profile not found, got %v", err)
	}
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: 2}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: 0}, nil)
	if _, err := service.profileIDByUserAccount(ctx, 2); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected empty profile to be not found, got %v", err)
	}
}

func makeString(length int) string {
	value := make([]byte, length)
	for i := range value {
		value[i] = 'a'
	}
	return string(value)
}

func int64Ptr(value int64) *int64 {
	return &value
}
