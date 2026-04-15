package utils

import (
	"context"
	"errors"
	"testing"

	postservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	servicemocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/golang/mock/gomock"
)

func TestMakeMock_StopsAfterFirstPostAttachmentError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mediaRepo := repomocks.NewMockMediaRepo(ctrl)
	userService := servicemocks.NewMockUserService(ctrl)
	postRepo := repomocks.NewMockPostRepo(ctrl)
	postWithMediaRepo := repomocks.NewMockPostWithMediaRepo(ctrl)
	commentRepo := repomocks.NewMockCommentRepo(ctrl)
	repostRepo := repomocks.NewMockRepostRepo(ctrl)
	likeRepo := repomocks.NewMockLikeRepo(ctrl)
	profileRepo := repomocks.NewMockProfileRepo(ctrl)
	postService := postservice.NewPostService(postRepo, postWithMediaRepo, profileRepo, commentRepo, repostRepo, likeRepo)

	userID := int64(1)
	userService.EXPECT().
		CreateRealUserProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, *string, *string, string, string, string, string, interface{}, interface{}, *int64) (*models.Profile, error) {
			profile := &models.Profile{ID: userID}
			userID++
			return profile, nil
		}).
		AnyTimes()

	saveCalls := 0
	mediaRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, models.Media) (int64, error) {
			saveCalls++
			return int64(saveCalls), nil
		}).
		AnyTimes()

	postID := int64(1)
	postRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, models.Post) (int64, error) {
			current := postID
			postID++
			return current, nil
		}).
		AnyTimes()

	postWithMediaRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(errors.New("stop at first post attachment")).
		AnyTimes()
	MakeMock(mediaRepo, userService, postService, postWithMediaRepo, nil, nil, nil, nil)
}
