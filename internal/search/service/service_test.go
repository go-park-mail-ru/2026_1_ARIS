package service

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/search/repository"
	searchmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/search/repository/mock"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSearchMapsUsersAndCommunities(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	searchRepo := searchmock.NewMockSearchRepo(ctrl)
	mediaClient := mediamock.NewMockMediaServiceClient(ctrl)
	svc := New(repository.NewStore(searchRepo), mediaClient)
	avatarID := int64(1)
	coverID := int64(2)
	bio := "bio"

	searchRepo.EXPECT().SearchUsers(ctx, "neo", maxLimit).Return([]repository.UserResult{
		{ProfileID: 10, UserAccountID: 20, Username: "neo", FirstName: "Neo", LastName: "Anderson", AvatarID: &avatarID},
	}, nil)
	searchRepo.EXPECT().SearchCommunities(ctx, "neo", maxLimit).Return([]repository.CommunityResult{
		{ID: 30, ProfileID: 40, Username: "team", Title: "Team", Bio: &bio, Type: models.PublicGroup, AvatarID: &avatarID, CoverMediaID: &coverID},
	}, nil)
	mediaClient.EXPECT().GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: avatarID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/a.png"}, nil).Times(2)
	mediaClient.EXPECT().GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: coverID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/c.png"}, nil)

	result, err := svc.Search(ctx, " neo ", 100)

	require.NoError(t, err)
	require.Len(t, result.Users, 1)
	require.Equal(t, "https://cdn.test/a.png", *result.Users[0].AvatarURL)
	require.Len(t, result.Communities, 1)
	require.Equal(t, "https://cdn.test/c.png", *result.Communities[0].CoverURL)
}

func TestSearchValidationAndErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	searchRepo := searchmock.NewMockSearchRepo(ctrl)
	svc := New(repository.NewStore(searchRepo), nil)

	result, err := svc.Search(ctx, " ", 10)
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrInvalidInput)

	searchRepo.EXPECT().SearchUsers(ctx, "neo", defaultLimit).Return(nil, errors.New("boom"))
	result, err = svc.Search(ctx, "neo", 0)
	require.Nil(t, result)
	require.Error(t, err)
}
