package user

import (
	"context"
	"errors"
	"testing"
	"time"

	hdto "github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserServiceCreateRealUserProfile(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ua := repomocks.NewMockUserAccountRepo(ctrl)
	pr := repomocks.NewMockProfileRepo(ctrl)
	up := repomocks.NewMockUserProfileRepo(ctrl)
	svc := NewUserService(ua, pr, up)

	ua.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(10), nil)
	pr.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(20), nil)
	up.EXPECT().Save(gomock.Any(), gomock.Any()).Return(int64(30), nil)
	want := &models.Profile{ID: 20}
	pr.EXPECT().Get(gomock.Any(), int64(20)).Return(want, nil)

	got, err := svc.CreateRealUserProfile(context.Background(), nil, nil, "secret", "login", "A", "B", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), models.Male, nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestUserServiceGetUserList(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ua := repomocks.NewMockUserAccountRepo(ctrl)
	svc := NewUserService(ua, repomocks.NewMockProfileRepo(ctrl), repomocks.NewMockUserProfileRepo(ctrl))
	want := []models.UserAccount{{ID: 1}}
	ua.EXPECT().List(gomock.Any(), 0, 10).Return(want, nil)
	got := svc.GetUserList(context.Background(), 0, 10)
	require.Equal(t, want, got)
}

func TestUserServiceDelegates(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	up := repomocks.NewMockUserProfileRepo(ctrl)
	pr := repomocks.NewMockProfileRepo(ctrl)
	svc := NewUserService(repomocks.NewMockUserAccountRepo(ctrl), pr, up)

	wantUP := &models.UserProfile{ID: 5}
	up.EXPECT().GetByProfileID(gomock.Any(), int64(3)).Return(wantUP, nil)
	got, err := svc.GetUserProfileByProfileID(context.Background(), 3)
	require.NoError(t, err)
	require.Equal(t, wantUP, got)

	up.EXPECT().Get(gomock.Any(), int64(4)).Return(wantUP, nil)
	got2, err := svc.GetUserProfileByUserProfileID(context.Background(), 4)
	require.NoError(t, err)
	require.Equal(t, wantUP, got2)

	up.EXPECT().GetByUserAccountID(gomock.Any(), int64(6)).Return(wantUP, nil)
	got3, err := svc.GetUserProfileByUserID(context.Background(), 6)
	require.NoError(t, err)
	require.Equal(t, wantUP, got3)

	up.EXPECT().GetByUserAccountID(gomock.Any(), int64(7)).Return(wantUP, nil)
	got4, err := svc.GetUserProfileByUserAccountID(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, wantUP, got4)

	pr.EXPECT().Get(gomock.Any(), int64(8)).Return(&models.Profile{ID: 8}, nil)
	p, err := svc.GetProfileByProfileID(context.Background(), 8)
	require.NoError(t, err)
	require.Equal(t, int64(8), p.ID)

	pr.EXPECT().GetByUserAccountID(gomock.Any(), int64(9)).Return(&models.Profile{ID: 9}, nil)
	p2, err := svc.GetProfileByUserAccountID(context.Background(), 9)
	require.NoError(t, err)
	require.Equal(t, int64(9), p2.ID)
}

func TestUserServiceGetUserAccountChains(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ua := repomocks.NewMockUserAccountRepo(ctrl)
	up := repomocks.NewMockUserProfileRepo(ctrl)
	svc := NewUserService(ua, repomocks.NewMockProfileRepo(ctrl), up)

	up.EXPECT().Get(gomock.Any(), int64(1)).Return(&models.UserProfile{UserAccountID: 10}, nil)
	ua.EXPECT().Get(gomock.Any(), int64(10)).Return(&models.UserAccount{ID: 10}, nil)
	acc, err := svc.GetUserAccountByUserProfileID(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, int64(10), acc.ID)

	up.EXPECT().GetByProfileID(gomock.Any(), int64(2)).Return(&models.UserProfile{UserAccountID: 11}, nil)
	ua.EXPECT().Get(gomock.Any(), int64(11)).Return(&models.UserAccount{ID: 11}, nil)
	acc2, err := svc.GetUserAccountByProfileID(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, int64(11), acc2.ID)

	uid := uuid.New()
	ua.EXPECT().GetByUid(gomock.Any(), uid).Return(&models.UserAccount{ID: 12}, nil)
	up.EXPECT().GetByUserAccountID(gomock.Any(), int64(12)).Return(&models.UserProfile{ID: 99}, nil)
	uprof, err := svc.GetUserProfileByUserAccountUid(context.Background(), uid)
	require.NoError(t, err)
	require.Equal(t, int64(99), uprof.ID)

	ua.EXPECT().GetByUid(gomock.Any(), uid).Return(&models.UserAccount{ID: 13}, nil)
	a, err := svc.GetUserAccountByUserAccountUid(context.Background(), uid)
	require.NoError(t, err)
	require.Equal(t, int64(13), a.ID)
}

func TestUserServiceGetProfileByUserProfileID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	up := repomocks.NewMockUserProfileRepo(ctrl)
	pr := repomocks.NewMockProfileRepo(ctrl)
	svc := NewUserService(repomocks.NewMockUserAccountRepo(ctrl), pr, up)

	up.EXPECT().Get(gomock.Any(), int64(5)).Return(&models.UserProfile{ProfileID: 77}, nil)
	pr.EXPECT().Get(gomock.Any(), int64(77)).Return(&models.Profile{ID: 77}, nil)
	got, err := svc.GetProfileByUserProfileID(context.Background(), 5)
	require.NoError(t, err)
	require.Equal(t, int64(77), got.ID)
}

func TestUserServiceGetSuggestedAndPopular(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ua := repomocks.NewMockUserAccountRepo(ctrl)
	pr := repomocks.NewMockProfileRepo(ctrl)
	up := repomocks.NewMockUserProfileRepo(ctrl)
	svc := NewUserService(ua, pr, up)

	pr.EXPECT().GetAll(gomock.Any()).Return([]models.Profile{{ID: 1}, {ID: 2}}, nil)
	up.EXPECT().GetByUserAccountID(gomock.Any(), int64(100)).Return(&models.UserProfile{ProfileID: 1}, nil)
	up.EXPECT().GetByProfileID(gomock.Any(), int64(2)).Return(&models.UserProfile{UserAccountID: 2}, nil).AnyTimes()
	ua.EXPECT().Get(gomock.Any(), int64(2)).Return(&models.UserAccount{Username: "other"}, nil).AnyTimes()

	got, err := svc.GetSuggestedUsers(context.Background(), 100)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(2), got[0].ID)

	pr.EXPECT().GetAll(gomock.Any()).Return(nil, nil)
	pop, err := svc.GetPublicPopularUsers(context.Background())
	require.NoError(t, err)
	require.Empty(t, pop)

	pr.EXPECT().GetAll(gomock.Any()).Return(nil, nil)
	ev, err := svc.GetLatestEvents(context.Background())
	require.NoError(t, err)
	require.Empty(t, ev)
}

func TestUserServiceUpdateMe(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ua := repomocks.NewMockUserAccountRepo(ctrl)
	pr := repomocks.NewMockProfileRepo(ctrl)
	up := repomocks.NewMockUserProfileRepo(ctrl)
	svc := NewUserService(ua, pr, up)

	login := "NEWUSER"
	ua.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	require.NoError(t, svc.UpdateMe(context.Background(), hdto.UpdateFullProfileRequestDTO{
		UserAccountID: 1,
		Username:      &login,
	}))

	bday := "2000-05-05"
	up.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	require.NoError(t, svc.UpdateMe(context.Background(), hdto.UpdateFullProfileRequestDTO{
		UserProfileID: 2,
		BirthdayDate:  &bday,
	}))

	rm := true
	aid := int64(9)
	pr.EXPECT().UpdateAvatar(gomock.Any(), int64(3), (*int64)(nil)).Return(nil)
	require.NoError(t, svc.UpdateMe(context.Background(), hdto.UpdateFullProfileRequestDTO{
		ProfileID:    3,
		RemoveAvatar: &rm,
	}))

	pr.EXPECT().UpdateAvatar(gomock.Any(), int64(3), &aid).Return(nil)
	require.NoError(t, svc.UpdateMe(context.Background(), hdto.UpdateFullProfileRequestDTO{
		ProfileID: 3,
		AvatarID:  &aid,
	}))

	ua.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errors.New("db"))
	u2 := "someuser"
	err := svc.UpdateMe(context.Background(), hdto.UpdateFullProfileRequestDTO{UserAccountID: 1, Username: &u2})
	require.Error(t, err)
}
