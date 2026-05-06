package grpc

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	friendmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/friend/mock"
	profilemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile/mock"
	settingsmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/settings/mock"
	accountmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account/mock"
	userprofilemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_profile/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/user/repository"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/user/service"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestUserGrpcHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accounts := accountmock.NewMockUserAccountRepo(ctrl)
	profiles := profilemock.NewMockProfileRepo(ctrl)
	userProfiles := userprofilemock.NewMockUserProfileRepo(ctrl)
	settings := settingsmock.NewMockUserSettingsRepository(ctrl)
	friendships := friendmock.NewMockFriendshipRepo(ctrl)
	svc := userservice.New(repository.NewStore(accounts, profiles, userProfiles, settings, friendships), nil)
	server := New(svc)
	ctx := context.Background()

	profiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.Profile{ID: 20}, nil)
	profileResp, err := server.GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: 10})
	require.NoError(t, err)
	require.Equal(t, int64(20), profileResp.ProfileId)

	userProfiles.EXPECT().GetByUserAccountID(ctx, int64(10)).Return(&models.UserProfile{ID: 30}, nil)
	userProfileResp, err := server.GetUserProfileByUserAccount(ctx, &userpb.GetUserProfileByUserAccountRequest{UserAccountId: 10})
	require.NoError(t, err)
	require.Equal(t, int64(30), userProfileResp.UserProfileId)

	avatarID := int64(40)
	userProfiles.EXPECT().GetByProfileID(ctx, int64(20)).Return(&models.UserProfile{
		UserAccountID: 10, ProfileID: 20, FirstName: "Neo", LastName: "Anderson",
	}, nil)
	accounts.EXPECT().Get(ctx, int64(10)).Return(&models.UserAccount{ID: 10, Username: "neo"}, nil)
	profiles.EXPECT().Get(ctx, int64(20)).Return(&models.Profile{ID: 20, AvatarID: &avatarID}, nil)
	summaryResp, err := server.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: 20})
	require.NoError(t, err)
	require.Equal(t, int64(20), summaryResp.ProfileId)
	require.Equal(t, int64(10), summaryResp.UserAccountId)
	require.Equal(t, "Neo", summaryResp.FirstName)
	require.Equal(t, "neo", summaryResp.Username)
	require.Equal(t, &avatarID, summaryResp.AvatarId)
}
