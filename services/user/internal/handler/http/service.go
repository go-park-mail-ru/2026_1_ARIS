package http

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/usecase"
)

type UserService interface {
	GetProfileMe(context.Context, int64) (*usecase.ProfileDetails, error)
	GetProfileByID(context.Context, int64) (*usecase.ProfileDetails, error)
	UpdateMe(context.Context, int64, usecase.UpdateFullProfileInput) error
	GetSuggestedUsers(context.Context, int64) ([]usecase.UserCard, error)
	GetPublicPopularUsers(context.Context) ([]usecase.UserCard, error)
	GetLatestEvents(context.Context) ([]usecase.LatestEvent, error)
	GetSettings(context.Context, int64) (*model.UserSettings, error)
	UpdateSettings(context.Context, int64, repository.SettingsUpdate) (*model.UserSettings, error)
	GetFriends(context.Context, int64, model.FriendshipStatus) ([]model.Friend, error)
	GetUsersFriends(context.Context, int64) ([]model.Friend, error)
	DeleteFriend(context.Context, int64, int64) error
	RequestFriendship(context.Context, int64, int64) error
	GetIncomingFriendRequests(context.Context, int64, string) ([]model.Friend, error)
	GetOutgoingFriendRequests(context.Context, int64, string) ([]model.Friend, error)
	AcceptFriendRequest(context.Context, int64, int64) error
	DeclineFriendRequest(context.Context, int64, int64) error
	RevokeFriendRequest(context.Context, int64, int64) error
}
